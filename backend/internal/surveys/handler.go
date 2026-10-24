package surveys

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

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
	r.Get("/surveys", h.list)
	r.Get("/surveys/{id}", h.get)
	r.Get("/surveys/{id}/results", h.results)
	r.With(auth.RequireUser).Put("/surveys/{id}/response", h.submit)
	r.Group(func(officer chi.Router) {
		officer.Use(auth.RequireOfficer)
		officer.Post("/surveys", h.create)
		officer.Patch("/surveys/{id}", h.update)
		officer.Get("/surveys/{id}/responses", h.listResponses)
		officer.Get("/surveys/{id}/responses.csv", h.exportResponses)
		officer.Patch("/surveys/{id}/responses/{responseId}", h.voidResponse)
	})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	summaries, err := h.service.List(r.Context())
	if err != nil {
		h.writeError(w, r, "list surveys", err)
		return
	}
	writeJSON(w, http.StatusOK, summaries)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	var viewer *auth.User
	if user, ok := auth.UserFrom(r.Context()); ok {
		viewer = &user
	}
	detail, err := h.service.Get(r.Context(), chi.URLParam(r, "id"), viewer)
	if err != nil {
		h.writeError(w, r, "get survey", err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *Handler) submit(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var req SubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body: must be valid JSON"})
		return
	}

	user, _ := auth.UserFrom(r.Context())
	saved, err := h.service.Submit(r.Context(), chi.URLParam(r, "id"), user, req)
	if err != nil {
		h.writeError(w, r, "submit survey response", err)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (h *Handler) results(w http.ResponseWriter, r *http.Request) {
	results, err := h.service.Results(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		h.writeError(w, r, "get survey results", err)
		return
	}
	writeJSON(w, http.StatusOK, results)
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, action string, err error) {
	var validationErr *ValidationError
	switch {
	case errors.As(err, &validationErr):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": validationErr.Error()})
	case errors.Is(err, ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "survey not found"})
	case errors.Is(err, ErrResponseNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": ErrResponseNotFound.Error()})
	case errors.Is(err, ErrQuestionsLocked):
		writeJSON(w, http.StatusConflict, map[string]string{"error": ErrQuestionsLocked.Error()})
	case errors.Is(err, ErrClosed):
		writeJSON(w, http.StatusConflict, map[string]string{"error": ErrClosed.Error()})
	case errors.Is(err, ErrResultsNotAvailable):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": ErrResultsNotAvailable.Error()})
	default:
		slog.ErrorContext(r.Context(), action, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": action + " failed"})
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("encode surveys response", "error", err)
	}
}
