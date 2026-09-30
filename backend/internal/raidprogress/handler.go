package raidprogress

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
	r.Get("/raid-progress", h.currentProgress)
	r.Get("/raid-tiers", h.listTiers)
	r.Group(func(officer chi.Router) {
		officer.Use(auth.RequireOfficer)
		officer.Post("/raid-tiers", h.createTier)
		officer.Put("/raid-tiers/order", h.reorderTiers)
		officer.Patch("/raid-tiers/{id}", h.updateTier)
		officer.Delete("/raid-tiers/{id}", h.deleteTier)
		officer.Post("/raid-tiers/{id}/bosses", h.addBoss)
		officer.Put("/raid-tiers/{id}/bosses/order", h.reorderBosses)
		officer.Patch("/raid-bosses/{id}", h.updateBoss)
		officer.Delete("/raid-bosses/{id}", h.deleteBoss)
	})
}

func (h *Handler) currentProgress(w http.ResponseWriter, r *http.Request) {
	progress, err := h.service.CurrentProgress(r.Context())
	if err != nil {
		h.writeError(w, r, "get raid progress", err)
		return
	}
	writeJSON(w, http.StatusOK, progress)
}

func (h *Handler) listTiers(w http.ResponseWriter, r *http.Request) {
	tiers, err := h.service.ListTiers(r.Context())
	if err != nil {
		h.writeError(w, r, "list raid tiers", err)
		return
	}
	writeJSON(w, http.StatusOK, tiers)
}

func (h *Handler) createTier(w http.ResponseWriter, r *http.Request) {
	var req CreateTierRequest
	if !decodeBody(w, r, &req) {
		return
	}

	created, err := h.service.CreateTier(r.Context(), req)
	if err != nil {
		h.writeError(w, r, "create raid tier", err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) updateTier(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, ErrTierNotFound)
	if !ok {
		return
	}
	var req UpdateTierRequest
	if !decodeBody(w, r, &req) {
		return
	}

	updated, err := h.service.UpdateTier(r.Context(), id, req)
	if err != nil {
		h.writeError(w, r, "update raid tier", err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) reorderTiers(w http.ResponseWriter, r *http.Request) {
	var req ReorderRequest
	if !decodeBody(w, r, &req) {
		return
	}

	if err := h.service.ReorderTiers(r.Context(), req); err != nil {
		h.writeError(w, r, "reorder raid tiers", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) addBoss(w http.ResponseWriter, r *http.Request) {
	tierID, ok := parseID(w, r, ErrTierNotFound)
	if !ok {
		return
	}
	var req AddBossRequest
	if !decodeBody(w, r, &req) {
		return
	}

	added, err := h.service.AddBoss(r.Context(), tierID, req)
	if err != nil {
		h.writeError(w, r, "add raid boss", err)
		return
	}
	writeJSON(w, http.StatusCreated, added)
}

func (h *Handler) reorderBosses(w http.ResponseWriter, r *http.Request) {
	tierID, ok := parseID(w, r, ErrTierNotFound)
	if !ok {
		return
	}
	var req ReorderRequest
	if !decodeBody(w, r, &req) {
		return
	}

	if err := h.service.ReorderBosses(r.Context(), tierID, req); err != nil {
		h.writeError(w, r, "reorder raid bosses", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) updateBoss(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, ErrBossNotFound)
	if !ok {
		return
	}
	var req UpdateBossRequest
	if !decodeBody(w, r, &req) {
		return
	}

	updated, err := h.service.UpdateBoss(r.Context(), id, req)
	if err != nil {
		h.writeError(w, r, "update raid boss", err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) deleteTier(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, ErrTierNotFound)
	if !ok {
		return
	}

	if err := h.service.DeleteTier(r.Context(), id); err != nil {
		h.writeError(w, r, "delete raid tier", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deleteBoss(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, ErrBossNotFound)
	if !ok {
		return
	}

	if err := h.service.DeleteBoss(r.Context(), id); err != nil {
		h.writeError(w, r, "delete raid boss", err)
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

func parseID(w http.ResponseWriter, r *http.Request, notFound error) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": notFound.Error()})
		return uuid.UUID{}, false
	}
	return id, true
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, action string, err error) {
	var validationErr *ValidationError
	switch {
	case errors.As(err, &validationErr):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": validationErr.Error()})
	case errors.Is(err, ErrNoCurrentTier):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": ErrNoCurrentTier.Error()})
	case errors.Is(err, ErrTierNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": ErrTierNotFound.Error()})
	case errors.Is(err, ErrBossNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": ErrBossNotFound.Error()})
	case errors.Is(err, ErrTierOrderMismatch):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": ErrTierOrderMismatch.Error()})
	case errors.Is(err, ErrBossOrderMismatch):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": ErrBossOrderMismatch.Error()})
	case errors.Is(err, ErrTierIsCurrent):
		writeJSON(w, http.StatusConflict, map[string]string{"error": ErrTierIsCurrent.Error()})
	default:
		slog.ErrorContext(r.Context(), action, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": action + " failed"})
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("encode raid progress response", "error", err)
	}
}
