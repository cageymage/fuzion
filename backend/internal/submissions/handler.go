package submissions

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/cageymage/fuzion/backend/internal/auth"
)

const (
	metadataField     = "metadata"
	filesField        = "files"
	multipartOverhead = 64 << 10
	maxMetadataBytes  = 64 << 10
	defaultPageSize   = 48
	maxPageSize       = 100
	maxBulkDelete     = 100
)

// DefaultUploadSlots is how many uploads may be read and decoded at once. Each can
// hold a 50 MiB body and a 25-megapixel decode (about 150 MB together), so two fit
// on a 512 MB instance with room for everything else.
const DefaultUploadSlots = 2

type Handler struct {
	service *Service
	uploads chan struct{}
}

func NewHandler(service *Service, uploadSlots int) *Handler {
	return &Handler{service: service, uploads: make(chan struct{}, uploadSlots)}
}

func (h *Handler) Register(r chi.Router) {
	r.Route("/submissions", func(s chi.Router) {
		s.Use(privateResponses)
		s.Get("/status", h.status)
		s.With(auth.RequireGuildMember).Post("/", h.upload)
		s.Group(func(officer chi.Router) {
			officer.Use(auth.RequireOfficer)
			officer.Get("/", h.list)
			officer.Get("/{id}/thumb", h.thumb)
			officer.Get("/{id}/full", h.full)
			officer.Get("/{id}/download", h.download)
			officer.Delete("/{id}", h.deleteOne)
			officer.Post("/bulk-delete", h.deleteMany)
		})
	})
}

// Submissions are private to officers, so nothing is cached by shared caches, and
// browsers must not second-guess the content type the upload was sniffed as.
func privateResponses(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "private")
		next.ServeHTTP(w, r)
	})
}

type submissionJSON struct {
	ID               uuid.UUID     `json:"id"`
	Submitter        submitterJSON `json:"submitter"`
	Description      string        `json:"description"`
	OriginalFilename string        `json:"originalFilename"`
	ContentType      string        `json:"contentType"`
	Width            int           `json:"width"`
	Height           int           `json:"height"`
	ByteSize         int           `json:"byteSize"`
	CreatedAt        time.Time     `json:"createdAt"`
	ThumbURL         string        `json:"thumbUrl"`
	FullURL          string        `json:"fullUrl"`
	DownloadURL      string        `json:"downloadUrl"`
}

type submitterJSON struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
}

func toJSON(s Summary) submissionJSON {
	base := "/api/submissions/" + s.ID.String()
	return submissionJSON{
		ID:               s.ID,
		Submitter:        submitterJSON{ID: s.SubmitterID, Username: s.SubmitterUsername},
		Description:      s.Description,
		OriginalFilename: s.OriginalFilename,
		ContentType:      s.ContentType,
		Width:            s.Width,
		Height:           s.Height,
		ByteSize:         s.ByteSize,
		CreatedAt:        s.CreatedAt,
		ThumbURL:         base + "/thumb",
		FullURL:          base + "/full",
		DownloadURL:      base + "/download",
	}
}

func (h *Handler) status(w http.ResponseWriter, r *http.Request) {
	open, err := h.service.Open(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "submission status", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "submission status could not be loaded"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"open": open, "attestationVersion": AttestationVersion})
}

type uploadMetadata struct {
	AttestationAccepted bool     `json:"attestationAccepted"`
	AttestationVersion  string   `json:"attestationVersion"`
	Descriptions        []string `json:"descriptions"`
}

type fileResultJSON struct {
	Filename   string          `json:"filename"`
	Submission *submissionJSON `json:"submission,omitempty"`
	Error      string          `json:"error,omitempty"`
}

// The body is multipart: a "metadata" JSON part first, then one "files" part per
// image. descriptions[i] belongs to the i-th file.
func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
	// Checked before the body is read, so a closed form costs a flood nothing.
	if !h.service.Enabled() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": ErrClosed.Error()})
		return
	}
	// Taken before the body is read, so a burst of uploads is refused instead of
	// buffered and decoded all at once.
	select {
	case h.uploads <- struct{}{}:
		defer func() { <-h.uploads }()
	default:
		w.Header().Set("Retry-After", "10")
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "too many uploads are in progress; try again in a few seconds"})
		return
	}

	req, err := readUpload(w, r)
	if err != nil {
		writeUploadError(w, r, err)
		return
	}

	user, _ := auth.UserFrom(r.Context())
	results, err := h.service.Upload(r.Context(), user.ID, req)
	if err != nil {
		writeUploadError(w, r, err)
		return
	}

	body := make([]fileResultJSON, len(results))
	created, duplicates := 0, 0
	for i, result := range results {
		body[i] = fileResultJSON{Filename: result.Filename, Error: result.Problem}
		if result.Submission != nil {
			s := toJSON(*result.Submission)
			body[i].Submission = &s
			created++
		}
		if result.Duplicate {
			duplicates++
		}
	}

	status := http.StatusCreated
	switch {
	case created > 0:
	case duplicates == len(results):
		status = http.StatusConflict
	default:
		status = http.StatusBadRequest
	}
	writeJSON(w, status, map[string]any{"results": body})
}

// Streams the parts rather than calling ParseMultipartForm, so an oversized body
// is cut off instead of being spooled to a temp file.
func readUpload(w http.ResponseWriter, r *http.Request) (UploadRequest, error) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBatchBytes+multipartOverhead)
	reader, err := r.MultipartReader()
	if err != nil {
		return UploadRequest{}, &ValidationError{Field: "body", Problem: "must be multipart/form-data"}
	}

	var req UploadRequest
	sawMetadata := false
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return UploadRequest{}, err
		}
		switch part.FormName() {
		case metadataField:
			var meta uploadMetadata
			if err := json.NewDecoder(io.LimitReader(part, maxMetadataBytes)).Decode(&meta); err != nil {
				return UploadRequest{}, &ValidationError{Field: metadataField, Problem: "must be valid JSON"}
			}
			req.AttestationAccepted = meta.AttestationAccepted
			req.AttestationVersion = meta.AttestationVersion
			req.Descriptions = meta.Descriptions
			sawMetadata = true
		case filesField:
			if len(req.Files) == MaxBatchFiles {
				return UploadRequest{}, &ValidationError{Field: "files", Problem: "at most " + strconv.Itoa(MaxBatchFiles) + " images per submission"}
			}
			data, err := io.ReadAll(io.LimitReader(part, MaxFileBytes+1))
			if err != nil {
				return UploadRequest{}, err
			}
			file := UploadFile{Filename: part.FileName(), Data: data}
			if len(data) > MaxFileBytes {
				file = UploadFile{Filename: part.FileName(), TooLarge: true}
			}
			req.Files = append(req.Files, file)
		}
	}
	if !sawMetadata {
		return UploadRequest{}, &ValidationError{Field: metadataField, Problem: "is required"}
	}
	return req, nil
}

func writeUploadError(w http.ResponseWriter, r *http.Request, err error) {
	var validationErr *ValidationError
	var quotaErr *QuotaError
	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &validationErr):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": validationErr.Error()})
	case errors.As(err, &quotaErr):
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": quotaErr.Error()})
	case errors.As(err, &tooLarge):
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "a submission must be at most " + strconv.Itoa(MaxBatchBytes>>20) + " MiB in total"})
	case errors.Is(err, ErrClosed), errors.Is(err, ErrStorageFull):
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
	case errors.Is(err, context.Canceled):
		// The client went away mid-upload; there is no one to answer and nothing to log.
	default:
		slog.ErrorContext(r.Context(), "upload submission", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "submission could not be uploaded"})
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := pageParams(w, r)
	if !ok {
		return
	}
	summaries, total, err := h.service.List(r.Context(), limit, offset)
	if err != nil {
		slog.ErrorContext(r.Context(), "list submissions", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "submissions could not be loaded"})
		return
	}
	items := make([]submissionJSON, len(summaries))
	for i, s := range summaries {
		items[i] = toJSON(s)
	}
	writeJSON(w, http.StatusOK, map[string]any{"total": total, "limit": limit, "offset": offset, "items": items})
}

func pageParams(w http.ResponseWriter, r *http.Request) (limit, offset int, ok bool) {
	limit, offset = defaultPageSize, 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > maxPageSize {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "limit: must be between 1 and " + strconv.Itoa(maxPageSize)})
			return 0, 0, false
		}
		limit = n
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "offset: must be 0 or more"})
			return 0, 0, false
		}
		offset = n
	}
	return limit, offset, true
}

func (h *Handler) thumb(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	data, err := h.service.Thumb(r.Context(), id)
	if err != nil {
		writeReadError(w, r, "get submission thumbnail", err)
		return
	}
	writeBytes(w, "image/webp", data)
}

func (h *Handler) full(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	file, err := h.service.Original(r.Context(), id)
	if err != nil {
		writeReadError(w, r, "get submission", err)
		return
	}
	writeBytes(w, file.ContentType, file.Bytes)
}

func (h *Handler) download(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	file, err := h.service.Original(r.Context(), id)
	if err != nil {
		writeReadError(w, r, "download submission", err)
		return
	}
	// Sanitized again on the way out, so a stored name can never inject a header.
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": SafeFilename(file.Filename)})
	if disposition == "" {
		disposition = `attachment; filename="submission"`
	}
	w.Header().Set("Content-Disposition", disposition)
	writeBytes(w, file.ContentType, file.Bytes)
}

func writeBytes(w http.ResponseWriter, contentType string, data []byte) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	if _, err := w.Write(data); err != nil {
		slog.Error("write submission response", "error", err)
	}
}

func (h *Handler) deleteOne(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	officer, _ := auth.UserFrom(r.Context())
	deleted, err := h.service.Delete(r.Context(), officer.ID, []uuid.UUID{id})
	if err != nil {
		writeReadError(w, r, "delete submission", err)
		return
	}
	if len(deleted) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": ErrNotFound.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type bulkDeleteRequest struct {
	IDs []string `json:"ids"`
}

func (h *Handler) deleteMany(w http.ResponseWriter, r *http.Request) {
	var req bulkDeleteRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxMetadataBytes)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body: must be valid JSON"})
		return
	}
	if len(req.IDs) == 0 || len(req.IDs) > maxBulkDelete {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "ids: must list between 1 and " + strconv.Itoa(maxBulkDelete) + " submissions"})
		return
	}
	ids := make([]uuid.UUID, len(req.IDs))
	for i, raw := range req.IDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "ids[" + strconv.Itoa(i) + "]: must be a submission id"})
			return
		}
		ids[i] = id
	}

	officer, _ := auth.UserFrom(r.Context())
	deleted, err := h.service.Delete(r.Context(), officer.ID, ids)
	if err != nil {
		writeReadError(w, r, "delete submissions", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": deleted})
}

func parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": ErrNotFound.Error()})
		return uuid.UUID{}, false
	}
	return id, true
}

func writeReadError(w http.ResponseWriter, r *http.Request, action string, err error) {
	if errors.Is(err, ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": ErrNotFound.Error()})
		return
	}
	slog.ErrorContext(r.Context(), action, "error", err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": action + " failed"})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("encode submissions response", "error", err)
	}
}
