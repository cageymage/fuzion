package streams

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
	r.Get("/streams", h.listAll)
	r.Get("/streams/live", h.listLive)
}

func (h *Handler) listLive(w http.ResponseWriter, r *http.Request) {
	live, err := h.service.LiveStreams(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "list live streams", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "live streams could not be loaded"})
		return
	}
	writeJSON(w, http.StatusOK, live)
}

func (h *Handler) listAll(w http.ResponseWriter, r *http.Request) {
	all, err := h.service.AllStreams(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "list all streams", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streams could not be loaded"})
		return
	}
	writeJSON(w, http.StatusOK, all)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("encode streams response", "error", err)
	}
}
