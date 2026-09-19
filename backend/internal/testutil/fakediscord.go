package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
)

const (
	DiscordClientID     = "123456789012345678"
	DiscordClientSecret = "test-client-secret"
	DiscordRedirectURL  = "http://localhost:5173/api/auth/callback"
)

type DiscordUser struct {
	ID       string  `json:"id"`
	Username string  `json:"username"`
	Avatar   *string `json:"avatar"`
}

// FakeDiscord stands in for Discord's OAuth2 API. Codes handed to GrantCode are
// accepted by /oauth2/token and resolve to their user on /users/@me; anything
// else is rejected the way Discord rejects an unknown code.
type FakeDiscord struct {
	*httptest.Server

	// UserEndpointStatus, when non-zero, makes /users/@me fail with that status.
	UserEndpointStatus int

	mu    sync.Mutex
	codes map[string]DiscordUser
}

func NewFakeDiscord(t *testing.T) *FakeDiscord {
	t.Helper()

	fake := &FakeDiscord{codes: map[string]DiscordUser{}}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /oauth2/token", fake.token(t))
	mux.HandleFunc("GET /users/@me", fake.me)
	fake.Server = httptest.NewServer(mux)
	t.Cleanup(fake.Close)
	return fake
}

func (f *FakeDiscord) GrantCode(code string, user DiscordUser) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.codes[code] = user
}

func (f *FakeDiscord) token(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse token form: %v", err)
		}
		code := r.PostForm.Get("code")
		wantForm := url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {code},
			"redirect_uri":  {DiscordRedirectURL},
			"client_id":     {DiscordClientID},
			"client_secret": {DiscordClientSecret},
		}
		if diff := cmp.Diff(wantForm, r.PostForm); diff != "" {
			t.Errorf("unexpected token request form (-want +got):\n%s", diff)
		}

		f.mu.Lock()
		_, ok := f.codes[code]
		f.mu.Unlock()
		if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":             "invalid_grant",
				"error_description": `Invalid "code" in request.`,
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"access_token": "token-for-" + code,
			"token_type":   "Bearer",
			"expires_in":   604800,
			"scope":        "identify",
		})
	}
}

func (f *FakeDiscord) me(w http.ResponseWriter, r *http.Request) {
	if f.UserEndpointStatus != 0 {
		w.WriteHeader(f.UserEndpointStatus)
		return
	}
	code, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer token-for-")
	f.mu.Lock()
	user, known := f.codes[code]
	f.mu.Unlock()
	if !ok || !known {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"message": "401: Unauthorized", "code": 0})
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
