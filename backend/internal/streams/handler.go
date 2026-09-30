package streams

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service         *Service
	suggestedVideos *SuggestedVideos
}

func NewHandler(service *Service, suggestedVideos *SuggestedVideos) *Handler {
	return &Handler{service: service, suggestedVideos: suggestedVideos}
}

func (h *Handler) Register(r chi.Router) {
	r.Get("/streams", h.listAll)
	r.Get("/streams/live", h.listLive)
	r.Get("/streams/suggested-video", h.suggestedVideo)
}

func (h *Handler) suggestedVideo(w http.ResponseWriter, r *http.Request) {
	video, ok, err := h.suggestedVideos.Suggest(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "suggest video", "error", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "suggested video could not be loaded"})
		return
	}
	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"id": video.ID, "title": video.Title})
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
