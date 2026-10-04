package images

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/cageymage/fuzion/backend/internal/auth"
)

const (
	maxUploadBytes    = 10 << 20
	multipartOverhead = 64 << 10
	fileField         = "file"
	// Image ids are never rewritten, so clients may cache them forever.
	immutableCache = "public, max-age=31536000, immutable"
)

type uploadResponse struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(r chi.Router) {
	r.Get("/images/{id}", h.getFull)
	r.Get("/images/{id}/thumb", h.getThumb)
	r.Group(func(officer chi.Router) {
		officer.Use(auth.RequireOfficer)
		officer.Post("/images", h.upload)
	})
}

func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
	data, declaredType, err := readFilePart(w, r)
	if err != nil {
		h.writeError(w, r, "read image upload", err)
		return
	}

	id, err := h.service.Upload(r.Context(), data, declaredType)
	if err != nil {
		h.writeError(w, r, "upload image", err)
		return
	}
	writeJSON(w, http.StatusCreated, uploadResponse{ID: id.String(), URL: "/api/images/" + id.String()})
}

// Streams the multipart body rather than calling ParseMultipartForm so an
// oversized upload is cut off instead of being spooled to a temp file.
func readFilePart(w http.ResponseWriter, r *http.Request) ([]byte, string, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes+multipartOverhead)
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, "", &ValidationError{Field: "body", Problem: "must be multipart/form-data"}
	}
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			return nil, "", &ValidationError{Field: fileField, Problem: "is required"}
		}
		if err != nil {
			return nil, "", err
		}
		if part.FormName() != fileField {
			continue
		}
		data, err := io.ReadAll(io.LimitReader(part, maxUploadBytes+1))
		if err != nil {
			return nil, "", err
		}
		if len(data) > maxUploadBytes {
			return nil, "", &http.MaxBytesError{Limit: maxUploadBytes}
		}
		return data, part.Header.Get("Content-Type"), nil
	}
}

func (h *Handler) getFull(w http.ResponseWriter, r *http.Request) {
	data, err := h.service.Full(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		h.writeError(w, r, "get image", err)
		return
	}
	writeImage(w, data)
}

func (h *Handler) getThumb(w http.ResponseWriter, r *http.Request) {
	data, err := h.service.Thumb(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		h.writeError(w, r, "get image thumbnail", err)
		return
	}
	writeImage(w, data)
}

func writeImage(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", storedType)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("Cache-Control", immutableCache)
	if _, err := w.Write(data); err != nil {
		slog.Error("write image response", "error", err)
	}
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, action string, err error) {
	var validationErr *ValidationError
	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &validationErr):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": validationErr.Error()})
	case errors.As(err, &tooLarge):
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "file: must be at most 10 MiB"})
	case errors.Is(err, ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "image not found"})
	default:
		slog.ErrorContext(r.Context(), action, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": action + " failed"})
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("encode images response", "error", err)
	}
}
