package professions

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
	r.Get("/professions", h.list)
	r.With(auth.RequireOfficer).Post("/professions", h.create)
	r.With(auth.RequireOfficer).Patch("/professions/{id}", h.update)
	r.With(auth.RequireOfficer).Delete("/professions/{id}", h.delete)
	r.With(auth.RequireUser).Post("/me/characters/{id}/professions", h.createMine)
	r.With(auth.RequireUser).Patch("/me/professions/{id}", h.updateMine)
	r.With(auth.RequireUser).Delete("/me/professions/{id}", h.deleteMine)
}

func (h *Handler) createMine(w http.ResponseWriter, r *http.Request) {
	characterID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": ErrUnknownCharacter.Error()})
		return
	}
	var req MyCreateRequest
	if !decodeBody(w, r, &req) {
		return
	}

	user, _ := auth.UserFrom(r.Context())
	created, err := h.service.CreateMine(r.Context(), user.ID, characterID, req)
	if err != nil {
		writeError(w, r, "create my profession", err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) updateMine(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var req UpdateRequest
	if !decodeBody(w, r, &req) {
		return
	}

	user, _ := auth.UserFrom(r.Context())
	updated, err := h.service.UpdateMine(r.Context(), user.ID, id, req)
	if err != nil {
		writeError(w, r, "update my profession", err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) deleteMine(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	user, _ := auth.UserFrom(r.Context())
	if err := h.service.DeleteMine(r.Context(), user.ID, id); err != nil {
		writeError(w, r, "delete my profession", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if !decodeBody(w, r, &req) {
		return
	}

	created, err := h.service.Create(r.Context(), req)
	if err != nil {
		writeError(w, r, "create profession", err)
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
		writeError(w, r, "update profession", err)
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
		writeError(w, r, "delete profession", err)
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
	case errors.Is(err, ErrUnknownCharacter):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": ErrUnknownCharacter.Error()})
	case errors.Is(err, ErrNotOwner):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": ErrNotOwner.Error()})
	default:
		slog.ErrorContext(r.Context(), action, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": action + " failed"})
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("encode professions response", "error", err)
	}
}
