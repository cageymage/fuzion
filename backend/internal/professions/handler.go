package professions

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
	r.Get("/professions", h.list)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	professions, err := h.service.List(r.Context(), r.URL.Query().Get("profession"))
	if err != nil {
		slog.ErrorContext(r.Context(), "list professions", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "professions could not be loaded"})
		return
	}
	writeJSON(w, http.StatusOK, professions)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("encode professions response", "error", err)
	}
}
