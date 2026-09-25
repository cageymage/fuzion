package auth

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
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

	r.Route("/admin/users", func(admin chi.Router) {
		admin.Use(RequireAdmin)
		admin.Get("/", h.listUsers)
		admin.Patch("/{id}", h.updateUserRoles)
	})
}

type meResponse struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	AvatarURL *string   `json:"avatarUrl"`
	IsOfficer bool      `json:"isOfficer"`
	IsAdmin   bool      `json:"isAdmin"`
}

type userResponse struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	AvatarURL *string   `json:"avatarUrl"`
	IsOfficer bool      `json:"isOfficer"`
	IsAdmin   bool      `json:"isAdmin"`
}

func toUserResponse(u User) userResponse {
	return userResponse{ID: u.ID, Username: u.Username, AvatarURL: u.AvatarURL, IsOfficer: u.IsOfficer, IsAdmin: u.IsAdmin}
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
	if err != nil {
		slog.WarnContext(r.Context(), "login callback without state cookie", "host", r.Host, "has_code", r.URL.Query().Has("code"))
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "login state mismatch"})
		return
	}
	if subtle.ConstantTimeCompare([]byte(stateCookie.Value), []byte(r.URL.Query().Get("state"))) != 1 {
		slog.WarnContext(r.Context(), "login callback state differs from cookie", "host", r.Host)
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
	writeJSON(w, http.StatusOK, meResponse{
		ID:        user.ID,
		Username:  user.Username,
		AvatarURL: user.AvatarURL,
		IsOfficer: user.HasOfficerAccess(),
		IsAdmin:   user.IsAdmin,
	})
}

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.ListUsers(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "list users", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not list users"})
		return
	}
	responses := make([]userResponse, len(users))
	for i, u := range users {
		responses[i] = toUserResponse(u)
	}
	writeJSON(w, http.StatusOK, responses)
}

type updateUserRolesRequest struct {
	IsOfficer bool `json:"isOfficer"`
	IsAdmin   bool `json:"isAdmin"`
}

func (h *Handler) updateUserRoles(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user id"})
		return
	}

	var req updateUserRolesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	caller, _ := UserFrom(r.Context())
	user, err := h.service.UpdateUserRoles(r.Context(), caller, id, req.IsOfficer, req.IsAdmin)
	switch {
	case errors.Is(err, ErrCannotRemoveOwnAdmin):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot remove your own admin access"})
		return
	case errors.Is(err, ErrUserNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	case err != nil:
		slog.ErrorContext(r.Context(), "update user roles", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not update user roles"})
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(user))
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("encode auth response", "error", err)
	}
}
