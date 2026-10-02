package news

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/cageymage/fuzion/backend/internal/auth"
)

const (
	defaultLimit = 10
	maxLimit     = 50
)

var categories = []string{"raid-progress", "recruitment", "guild-news", "patch-notes"}

type embeddedPosts struct {
	News []Post `json:"news"`
}

type listResponse struct {
	Total    int             `json:"total"`
	Links    map[string]link `json:"_links"`
	Embedded embeddedPosts   `json:"_embedded"`
}

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(r chi.Router) {
	r.Get("/news", h.listPosts)
	r.Get("/news/{id}", h.getPost)
	r.Group(func(officer chi.Router) {
		officer.Use(auth.RequireOfficer)
		officer.Get("/news/drafts", h.listDrafts)
		officer.Post("/news", h.create)
		officer.Patch("/news/{id}", h.update)
		officer.Post("/news/{id}/publish", h.publish)
		officer.Delete("/news/{id}", h.delete)
	})
}

func (h *Handler) getPost(w http.ResponseWriter, r *http.Request) {
	post, err := h.service.GetPublished(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		h.writeError(w, r, "get news post", err)
		return
	}
	writeJSON(w, http.StatusOK, post)
}

func (h *Handler) listDrafts(w http.ResponseWriter, r *http.Request) {
	posts, err := h.service.ListDrafts(r.Context())
	if err != nil {
		h.writeError(w, r, "list news drafts", err)
		return
	}
	writeJSON(w, http.StatusOK, posts)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if !decodeBody(w, r, &req) {
		return
	}
	officer, _ := auth.UserFrom(r.Context())
	post, err := h.service.Create(r.Context(), officer.ID, officer.Username, req)
	if err != nil {
		h.writeError(w, r, "create news post", err)
		return
	}
	writeJSON(w, http.StatusCreated, post)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var req UpdateRequest
	if !decodeBody(w, r, &req) {
		return
	}
	post, err := h.service.Update(r.Context(), chi.URLParam(r, "id"), req)
	if err != nil {
		h.writeError(w, r, "update news post", err)
		return
	}
	writeJSON(w, http.StatusOK, post)
}

func (h *Handler) publish(w http.ResponseWriter, r *http.Request) {
	post, err := h.service.Publish(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		h.writeError(w, r, "publish news post", err)
		return
	}
	writeJSON(w, http.StatusOK, post)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		h.writeError(w, r, "delete news post", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// maxBodyBytes leaves room for a 50 000 character body of multi-byte runes plus JSON escaping.
const maxBodyBytes = 512 << 10

func decodeBody(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body: must be valid JSON"})
		return false
	}
	return true
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, action string, err error) {
	var validationErr *ValidationError
	switch {
	case errors.As(err, &validationErr):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": validationErr.Error()})
	case errors.Is(err, ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "news post not found"})
	default:
		slog.ErrorContext(r.Context(), action, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": action + " failed"})
	}
}

func (h *Handler) listPosts(w http.ResponseWriter, r *http.Request) {
	params, err := parseListParams(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	page, err := h.service.ListPosts(r.Context(), params)
	if err != nil {
		slog.ErrorContext(r.Context(), "list news posts", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "news posts could not be loaded"})
		return
	}
	writeJSON(w, http.StatusOK, listResponse{
		Total:    page.Total,
		Links:    pageLinks(r.URL.Path, params, page.Total, r.URL.Query().Has("limit")),
		Embedded: embeddedPosts{News: page.Posts},
	})
}

func parseListParams(r *http.Request) (ListParams, error) {
	limit, err := parseLimit(r)
	if err != nil {
		return ListParams{}, err
	}
	offset, err := parseOffset(r)
	if err != nil {
		return ListParams{}, err
	}
	category, err := parseCategory(r)
	if err != nil {
		return ListParams{}, err
	}
	return ListParams{Limit: limit, Offset: offset, Category: category}, nil
}

func parseLimit(r *http.Request) (int, error) {
	if !r.URL.Query().Has("limit") {
		return defaultLimit, nil
	}
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit < 1 || limit > maxLimit {
		return 0, errors.New("limit: must be an integer between 1 and 50")
	}
	return limit, nil
}

func parseOffset(r *http.Request) (int, error) {
	if !r.URL.Query().Has("offset") {
		return 0, nil
	}
	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err != nil || offset < 0 {
		return 0, errors.New("offset: must be a non-negative integer")
	}
	return offset, nil
}

func parseCategory(r *http.Request) (string, error) {
	if !r.URL.Query().Has("category") {
		return "", nil
	}
	category := r.URL.Query().Get("category")
	if !slices.Contains(categories, category) {
		return "", errors.New("category: must be one of " + strings.Join(categories, ", "))
	}
	return category, nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("encode news response", "error", err)
	}
}
