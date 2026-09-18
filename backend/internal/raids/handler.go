package raids

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
	r.Get("/raids/next", h.nextRaid)
}

// A guild with nothing on the calendar is a normal state, not an error, so an
// empty schedule answers 200 with a null body rather than 404.
func (h *Handler) nextRaid(w http.ResponseWriter, r *http.Request) {
	raid, err := h.service.NextRaid(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "get next raid", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "next raid could not be loaded"})
		return
	}
	writeJSON(w, http.StatusOK, raid)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("encode raids response", "error", err)
	}
}
