package roster

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/cageymage/fuzion/backend/internal/auth"
)

const maxBodyBytes = 64 << 10

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(r chi.Router) {
	r.Get("/roster", h.list)
	r.With(auth.RequireOfficer).Post("/roster", h.create)
	r.With(auth.RequireOfficer).Patch("/roster/{id}", h.update)
	r.With(auth.RequireOfficer).Delete("/roster/{id}", h.delete)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if !decodeBody(w, r, &req) {
		return
	}

	created, err := h.service.Create(r.Context(), req)
	if err != nil {
		writeError(w, r, "create character", err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var req UpdateRequest
	if !decodeBody(w, r, &req) {
		return
	}

	updated, err := h.service.Update(r.Context(), id, req)
	if err != nil {
		writeError(w, r, "update character", err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		writeError(w, r, "delete character", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeBody(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body: must be valid JSON"})
		return false
	}
	return true
}

func parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": ErrNotFound.Error()})
		return uuid.UUID{}, false
	}
	return id, true
}

func writeError(w http.ResponseWriter, r *http.Request, action string, err error) {
	var validationErr *ValidationError
	switch {
	case errors.As(err, &validationErr):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": validationErr.Error()})
	case errors.Is(err, ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": ErrNotFound.Error()})
	case errors.Is(err, ErrConflict):
		writeJSON(w, http.StatusConflict, map[string]string{"error": ErrConflict.Error()})
	default:
		slog.ErrorContext(r.Context(), action, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": action + " failed"})
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	characters, err := h.service.Roster(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "list roster", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "roster could not be loaded"})
		return
	}
	writeJSON(w, http.StatusOK, characters)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("encode roster response", "error", err)
	}
}
