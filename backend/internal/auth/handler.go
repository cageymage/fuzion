package auth

import (
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(r chi.Router) {
	r.Get("/auth/login", h.login)
	r.Get("/auth/callback", h.callback)
	r.Post("/auth/logout", h.logout)
	r.Get("/auth/me", h.me)
}

type meResponse struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	AvatarURL *string   `json:"avatarUrl"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	state, err := newState()
	if err != nil {
		slog.ErrorContext(r.Context(), "generate oauth state", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "login could not be started"})
		return
	}
	http.SetCookie(w, stateCookie(r, state))
	http.Redirect(w, r, h.service.AuthURL(state), http.StatusFound)
}

func (h *Handler) callback(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, expiredCookie(r, stateCookieName))

	stateCookie, err := r.Cookie(stateCookieName)
	if err != nil || subtle.ConstantTimeCompare([]byte(stateCookie.Value), []byte(r.URL.Query().Get("state"))) != 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "login state mismatch"})
		return
	}

	// The user pressed Cancel on the consent screen; send them home logged out.
	if r.URL.Query().Get("error") != "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	user, token, err := h.service.Login(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		slog.ErrorContext(r.Context(), "complete login", "error", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "login could not be completed"})
		return
	}
	slog.InfoContext(r.Context(), "user logged in", "user_id", user.ID, "username", user.Username)
	http.SetCookie(w, sessionCookie(r, token))
	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(SessionCookieName); err == nil && cookie.Value != "" {
		if err := h.service.Logout(r.Context(), cookie.Value); err != nil {
			slog.ErrorContext(r.Context(), "logout", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "logout failed"})
			return
		}
	}
	http.SetCookie(w, expiredCookie(r, SessionCookieName))
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFrom(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "login required"})
		return
	}
	writeJSON(w, http.StatusOK, meResponse{ID: user.ID, Username: user.Username, AvatarURL: user.AvatarURL})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("encode auth response", "error", err)
	}
}
