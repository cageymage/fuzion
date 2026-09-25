package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
)

type userKey struct{}

func UserFrom(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(userKey{}).(User)
	return user, ok
}

// Middleware resolves the session cookie into a User on the request context.
// Anonymous requests pass through untouched; only RequireUser turns that into a 401.
func (h *Handler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(SessionCookieName)
		if err != nil || cookie.Value == "" {
			next.ServeHTTP(w, r)
			return
		}
		user, err := h.service.UserBySession(r.Context(), cookie.Value)
		if err != nil {
			if !errors.Is(err, ErrSessionNotFound) {
				slog.ErrorContext(r.Context(), "resolve session", "error", err)
			}
			next.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey{}, user)))
	})
}

func RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := UserFrom(r.Context()); !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "login required"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireOfficer gates a route to officers and admins: 401 anonymous, 403 member.
func RequireOfficer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := UserFrom(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "login required"})
			return
		}
		if !user.HasOfficerAccess() {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "officer access required"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAdmin gates a route to admins only: 401 anonymous, 403 non-admin.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := UserFrom(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "login required"})
			return
		}
		if !user.IsAdmin {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin access required"})
			return
		}
		next.ServeHTTP(w, r)
	})
}
