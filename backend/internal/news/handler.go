package news

import (
	"encoding/json"
	"log/slog"
	"net/http"

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
	posts, err := h.service.LatestPosts(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "list news posts", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "news posts could not be loaded"})
		return
	}
	writeJSON(w, http.StatusOK, posts)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("encode news response", "error", err)
	}
}
