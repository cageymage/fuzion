package rankrequests

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
	r.Group(func(member chi.Router) {
		member.Use(auth.RequireUser)
		member.Post("/me/characters/{id}/rank-requests", h.create)
		member.Get("/me/rank-requests", h.listMine)
		member.Delete("/me/rank-requests/{id}", h.cancel)
	})
	r.Group(func(officer chi.Router) {
		officer.Use(auth.RequireOfficer)
		officer.Get("/rank-requests", h.list)
		officer.Get("/rank-requests/count", h.count)
		officer.Patch("/rank-requests/{id}", h.resolve)
	})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	characterID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": ErrCharacterNotFound.Error()})
		return
	}
	var req CreateRequest
	if !decodeBody(w, r, &req) {
		return
	}

	user, _ := auth.UserFrom(r.Context())
	created, err := h.service.Create(r.Context(), user.ID, characterID, req)
	if err != nil {
		writeError(w, r, "create rank request", err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) listMine(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFrom(r.Context())
	requests, err := h.service.ListMine(r.Context(), user.ID)
	if err != nil {
		writeError(w, r, "list my rank requests", err)
		return
	}
	writeJSON(w, http.StatusOK, requests)
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	user, _ := auth.UserFrom(r.Context())
	if err := h.service.Cancel(r.Context(), user.ID, id); err != nil {
		writeError(w, r, "cancel rank request", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	requests, err := h.service.List(r.Context())
	if err != nil {
		writeError(w, r, "list rank requests", err)
		return
	}
	writeJSON(w, http.StatusOK, requests)
}

func (h *Handler) count(w http.ResponseWriter, r *http.Request) {
	n, err := h.service.CountOpen(r.Context())
	if err != nil {
		writeError(w, r, "count rank requests", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"open": n})
}

func (h *Handler) resolve(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var req ResolveRequest
	if !decodeBody(w, r, &req) {
		return
	}

	officer, _ := auth.UserFrom(r.Context())
	resolved, err := h.service.Resolve(r.Context(), officer.ID, id, req)
	if err != nil {
		writeError(w, r, "resolve rank request", err)
		return
	}
	writeJSON(w, http.StatusOK, resolved)
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
	if errors.As(err, &validationErr) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": validationErr.Error()})
		return
	}
	for _, mapped := range []struct {
		sentinel error
		status   int
	}{
		{ErrNotFound, http.StatusNotFound},
		{ErrCharacterNotFound, http.StatusNotFound},
		{ErrCharacterNotOwned, http.StatusForbidden},
		{ErrRequestNotOwned, http.StatusForbidden},
		{ErrOpenRequestExists, http.StatusConflict},
		{ErrNotOpen, http.StatusConflict},
	} {
		if errors.Is(err, mapped.sentinel) {
			writeJSON(w, mapped.status, map[string]string{"error": mapped.sentinel.Error()})
			return
		}
	}
	slog.ErrorContext(r.Context(), action, "error", err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": action + " failed"})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("encode rank requests response", "error", err)
	}
}
