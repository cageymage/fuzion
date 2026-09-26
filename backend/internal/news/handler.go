package news

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(r chi.Router) {
	r.Get("/news", h.listPosts)
}

func (h *Handler) listPosts(w http.ResponseWriter, r *http.Request) {
	limit, err := parseLimit(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	posts, err := h.service.LatestPosts(r.Context(), limit)
	if err != nil {
		slog.ErrorContext(r.Context(), "list news posts", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "news posts could not be loaded"})
		return
	}
	writeJSON(w, http.StatusOK, posts)
}

func parseLimit(r *http.Request) (int, error) {
	if !r.URL.Query().Has("limit") {
		return 0, nil
	}
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit < 1 {
		return 0, errors.New("limit: must be a positive integer")
	}
	return limit, nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("encode news response", "error", err)
	}
}
