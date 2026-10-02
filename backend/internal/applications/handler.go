package applications

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net"
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
	r.Post("/applications", h.submit)
	r.Group(func(officer chi.Router) {
		officer.Use(auth.RequireOfficer)
		officer.Get("/applications", h.list)
		officer.Get("/applications/{id}", h.get)
		officer.Patch("/applications/{id}", h.review)
	})
}

func (h *Handler) submit(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var req SubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body: must be valid JSON"})
		return
	}

	var validationErr *ValidationError
	err := h.service.VerifyHuman(r.Context(), req.TurnstileToken, clientIP(r))
	if errors.As(err, &validationErr) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": validationErr.Error()})
		return
	}
	if err != nil {
		slog.ErrorContext(r.Context(), "verify application bot check", "error", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "verification service unavailable"})
		return
	}

	submitted, err := h.service.Submit(r.Context(), req)
	if errors.As(err, &validationErr) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": validationErr.Error()})
		return
	}
	if err != nil {
		slog.ErrorContext(r.Context(), "submit application", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "application could not be submitted"})
		return
	}
	writeJSON(w, http.StatusCreated, submitted)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	apps, err := h.service.List(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		h.writeError(w, r, "list applications", err)
		return
	}
	writeJSON(w, http.StatusOK, apps)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	app, err := h.service.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		h.writeError(w, r, "get application", err)
		return
	}
	writeJSON(w, http.StatusOK, app)
}

func (h *Handler) review(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var req ReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body: must be valid JSON"})
		return
	}

	officer, _ := auth.UserFrom(r.Context())
	app, err := h.service.Review(r.Context(), chi.URLParam(r, "id"), officer.ID, req)
	if err != nil {
		h.writeError(w, r, "review application", err)
		return
	}
	writeJSON(w, http.StatusOK, app)
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, action string, err error) {
	var validationErr *ValidationError
	switch {
	case errors.As(err, &validationErr):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": validationErr.Error()})
	case errors.Is(err, ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "application not found"})
	default:
		slog.ErrorContext(r.Context(), action, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": action + " failed"})
	}
}

// chi's RealIP middleware may already have replaced RemoteAddr with a bare IP from a proxy header.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("encode applications response", "error", err)
	}
}
