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
	BattleNetNamespace    = "profile-us"
)

type BattleNetUser struct {
	ID        int64  `json:"id"`
	BattleTag string `json:"battletag"`
	// Characters are served by /profile/user/wow, not /userinfo.
	Characters []BattleNetCharacter `json:"-"`
}

type BattleNetCharacter struct {
	Name      string
	RealmSlug string
}

// FakeBattleNet stands in for oauth.battle.net. Codes handed to GrantCode are
// accepted by /token (which, like Blizzard, wants the client credentials as HTTP
// Basic auth) and resolve to their user on /userinfo and their characters on
// /profile/user/wow, which it serves too, standing in for the regional game API.
type FakeBattleNet struct {
	*httptest.Server

	mu    sync.Mutex
	codes map[string]BattleNetUser
	// profileStatus, when set, is what /profile/user/wow answers instead of the list.
	profileStatus int
}

// FailProfile makes /profile/user/wow answer status, as a wrong namespace (404) or a
// game API outage (500) would, while sign-in on oauth.battle.net keeps working.
func (f *FakeBattleNet) FailProfile(status int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.profileStatus = status
}

func NewFakeBattleNet(t *testing.T) *FakeBattleNet {
	t.Helper()

	fake := &FakeBattleNet{codes: map[string]BattleNetUser{}}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /token", fake.token(t))
	mux.HandleFunc("GET /userinfo", fake.userinfo)
	mux.HandleFunc("GET /profile/user/wow", fake.profile(t))
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
	user, ok := f.userFromToken(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_token"})
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (f *FakeBattleNet) userFromToken(r *http.Request) (BattleNetUser, bool) {
	code, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer token-for-")
	f.mu.Lock()
	user, known := f.codes[code]
	f.mu.Unlock()
	return user, ok && known
}

func (f *FakeBattleNet) profile(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("namespace"); got != BattleNetNamespace {
			t.Errorf("profile request namespace = %q, want %q", got, BattleNetNamespace)
		}
		user, ok := f.userFromToken(r)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_token"})
			return
		}
		f.mu.Lock()
		status := f.profileStatus
		f.mu.Unlock()
		if status != 0 {
			writeJSON(w, status, map[string]any{"code": status, "detail": http.StatusText(status)})
			return
		}
		characters := []map[string]any{}
		for _, c := range user.Characters {
			characters = append(characters, map[string]any{
				"name":  c.Name,
				"level": 60,
				"realm": map[string]any{"name": c.RealmSlug, "slug": c.RealmSlug},
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"id":           user.ID,
			"wow_accounts": []map[string]any{{"id": 1, "characters": characters}},
		})
	}
}
