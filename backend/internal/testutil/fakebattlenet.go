package testutil

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
)

const (
	BattleNetClientID     = "battlenet-client-id"
	BattleNetClientSecret = "battlenet-client-secret"
	BattleNetRedirectURL  = "http://localhost:5173/api/auth/battlenet/callback"
)

type BattleNetUser struct {
	ID        int64  `json:"id"`
	BattleTag string `json:"battletag"`
}

// FakeBattleNet stands in for oauth.battle.net. Codes handed to GrantCode are
// accepted by /token (which, like Blizzard, wants the client credentials as HTTP
// Basic auth) and resolve to their user on /userinfo.
type FakeBattleNet struct {
	*httptest.Server

	mu    sync.Mutex
	codes map[string]BattleNetUser
}

func NewFakeBattleNet(t *testing.T) *FakeBattleNet {
	t.Helper()

	fake := &FakeBattleNet{codes: map[string]BattleNetUser{}}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /token", fake.token(t))
	mux.HandleFunc("GET /userinfo", fake.userinfo)
	fake.Server = httptest.NewServer(mux)
	t.Cleanup(fake.Close)
	return fake
}

func (f *FakeBattleNet) GrantCode(code string, user BattleNetUser) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.codes[code] = user
}

func (f *FakeBattleNet) token(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse token form: %v", err)
		}
		id, secret, ok := r.BasicAuth()
		if !ok || id != BattleNetClientID || secret != BattleNetClientSecret {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_client"})
			return
		}
		code := r.PostForm.Get("code")
		wantForm := url.Values{
			"grant_type":   {"authorization_code"},
			"code":         {code},
			"redirect_uri": {BattleNetRedirectURL},
		}
		if diff := cmp.Diff(wantForm, r.PostForm); diff != "" {
			t.Errorf("unexpected token request form (-want +got):\n%s", diff)
		}

		f.mu.Lock()
		_, known := f.codes[code]
		f.mu.Unlock()
		if !known {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"access_token": "token-for-" + code,
			"token_type":   "bearer",
			"expires_in":   86399,
			"scope":        "wow.profile",
		})
	}
}

func (f *FakeBattleNet) userinfo(w http.ResponseWriter, r *http.Request) {
	code, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer token-for-")
	f.mu.Lock()
	user, known := f.codes[code]
	f.mu.Unlock()
	if !ok || !known {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_token"})
		return
	}
	writeJSON(w, http.StatusOK, user)
}
