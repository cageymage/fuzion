package auth_test

import (
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/cageymage/fuzion/backend/internal/testutil"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.Run(m))
}

type meJSON struct {
	ID        string  `json:"id"`
	Username  string  `json:"username"`
	AvatarURL *string `json:"avatarUrl"`
}

type userRow struct {
	DiscordID string  `db:"discord_id"`
	Username  string  `db:"username"`
	AvatarURL *string `db:"avatar_url"`
}

// setCookie pulls one Set-Cookie header out of a response so its attributes can be checked.
func setCookie(t *testing.T, resp testutil.Response, name string) *http.Cookie {
	t.Helper()
	for _, c := range (&http.Response{Header: resp.Header}).Cookies() {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("response did not set cookie %q; Set-Cookie: %v", name, resp.Header["Set-Cookie"])
	return nil
}

// startLogin walks the /login step and returns the state Discord will be asked to echo back.
func startLogin(t *testing.T, srv *testutil.Server) string {
	t.Helper()
	resp := srv.Get(t, "/api/auth/login")
	resp.RequireStatus(t, 302)
	location, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatalf("parse login redirect: %v", err)
	}
	return location.Query().Get("state")
}

func TestAuthLogin_RedirectsToProviderWithStateCookie(t *testing.T) {
	// given a server wired to the fake Discord
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I start a login
	resp := srv.Get(t, "/api/auth/login")

	// then I expect a redirect to Discord's authorize URL carrying the same state as the cookie
	resp.RequireStatus(t, 302)
	location, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatalf("parse redirect location: %v", err)
	}
	if got := location.Scheme + "://" + location.Host + location.Path; got != srv.Discord.URL+"/oauth2/authorize" {
		t.Errorf("expected redirect to the fake Discord authorize endpoint, got %s", got)
	}
	state := location.Query().Get("state")
	if state == "" {
		t.Fatal("expected a state query param on the redirect")
	}
	cookie := setCookie(t, resp, "fuzion_oauth_state")
	if cookie.Value != state {
		t.Errorf("state cookie %q does not match redirect state %q", cookie.Value, state)
	}
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" {
		t.Errorf("unexpected state cookie attributes: %s", cookie.String())
	}
}

func TestAuthCallback_CreatesUserAndSetsSessionCookie_WhenStateMatches(t *testing.T) {
	// given a login in progress and a Discord that will accept the code
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	state := startLogin(t, srv)
	avatar := "8342729096ea3675442027381ff50dfe"
	srv.Discord.GrantCode("code-1", testutil.DiscordUser{ID: "80351110224678912", Username: "thundermane", Avatar: &avatar})

	// when Discord sends the browser back with the code and matching state
	resp := srv.Get(t, "/api/auth/callback?code=code-1&state="+state)

	// then I expect a redirect home, a session cookie, and a user row backing that session
	resp.RequireStatus(t, 302)
	if got := resp.Header.Get("Location"); got != "/" {
		t.Errorf("expected redirect to /, got %q", got)
	}
	cookie := setCookie(t, resp, "fuzion_session")
	if cookie.Value == "" || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" {
		t.Errorf("unexpected session cookie attributes: %s", cookie.String())
	}
	if cookie.MaxAge != int(30*24*time.Hour/time.Second) {
		t.Errorf("expected a 30 day session cookie, got Max-Age %d", cookie.MaxAge)
	}

	var user userRow
	if err := db.Get(&user, `SELECT discord_id, username, avatar_url FROM users`); err != nil {
		t.Fatalf("select created user: %v", err)
	}
	avatarURL := "https://cdn.discordapp.com/avatars/80351110224678912/8342729096ea3675442027381ff50dfe.png"
	wantUser := userRow{DiscordID: "80351110224678912", Username: "thundermane", AvatarURL: &avatarURL}
	if diff := cmp.Diff(wantUser, user); diff != "" {
		t.Errorf("unexpected user row (-want +got):\n%s", diff)
	}

	var sessionOwner string
	if err := db.Get(&sessionOwner, `
		SELECT u.discord_id FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token = $1 AND s.expires_at > now()`, cookie.Value); err != nil {
		t.Fatalf("select session for cookie: %v", err)
	}
	if sessionOwner != "80351110224678912" {
		t.Errorf("session belongs to %q, want the logged-in user", sessionOwner)
	}
}

func TestAuthCallback_ReturnsBadRequest_WhenStateDoesNotMatch(t *testing.T) {
	// given a login in progress with a known state
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	startLogin(t, srv)
	srv.Discord.GrantCode("code-1", testutil.DiscordUser{ID: "80351110224678912", Username: "thundermane"})

	// when the callback arrives with a different state
	resp := srv.Get(t, "/api/auth/callback?code=code-1&state=forged-by-someone-else")

	// then I expect a 400 and no user to have been created
	resp.RequireStatus(t, 400)
	var body map[string]string
	resp.DecodeJSON(t, &body)
	if diff := cmp.Diff(map[string]string{"error": "login state mismatch"}, body); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
	var users int
	if err := db.Get(&users, `SELECT count(*) FROM users`); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if users != 0 {
		t.Errorf("expected no users, found %d", users)
	}
}

func TestAuthCallback_ReturnsBadRequest_WhenNoLoginWasStarted(t *testing.T) {
	// given no login in progress, so no state cookie
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.Discord.GrantCode("code-1", testutil.DiscordUser{ID: "80351110224678912", Username: "thundermane"})

	// when a callback arrives out of the blue
	resp := srv.Get(t, "/api/auth/callback?code=code-1&state=anything")

	// then I expect a 400
	resp.RequireStatus(t, 400)
}

func TestAuthCallback_ReusesExistingUser_WhenDiscordIDAlreadyExists(t *testing.T) {
	// given a user who logged in before under an older username
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO users (id, discord_id, username, avatar_url, created_at, last_seen_at)
		VALUES ('11111111-1111-1111-1111-111111111111', '80351110224678912', 'oldname', NULL, '2024-01-01T00:00:00Z', '2024-01-01T00:00:00Z')`)
	state := startLogin(t, srv)
	srv.Discord.GrantCode("code-2", testutil.DiscordUser{ID: "80351110224678912", Username: "thundermane"})

	// when they complete a fresh login
	resp := srv.Get(t, "/api/auth/callback?code=code-2&state="+state)

	// then I expect the same row updated, not a second one
	resp.RequireStatus(t, 302)
	var users []struct {
		ID         string    `db:"id"`
		Username   string    `db:"username"`
		LastSeenAt time.Time `db:"last_seen_at"`
	}
	if err := db.Select(&users, `SELECT id, username, last_seen_at FROM users WHERE discord_id = '80351110224678912'`); err != nil {
		t.Fatalf("select users: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected exactly one user for the discord id, got %d", len(users))
	}
	if users[0].ID != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("expected the existing user id to be kept, got %s", users[0].ID)
	}
	if users[0].Username != "thundermane" {
		t.Errorf("expected username refreshed from Discord, got %q", users[0].Username)
	}
	if !users[0].LastSeenAt.After(time.Now().Add(-time.Minute)) {
		t.Errorf("expected last_seen_at to be bumped, got %s", users[0].LastSeenAt)
	}
}

func TestAuthCallback_RedirectsHomeLoggedOut_WhenUserDeniesConsent(t *testing.T) {
	// given a login in progress
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	state := startLogin(t, srv)

	// when Discord sends the browser back with an error instead of a code
	resp := srv.Get(t, "/api/auth/callback?error=access_denied&state="+state)

	// then I expect a redirect home with no session cookie
	resp.RequireStatus(t, 302)
	if got := resp.Header.Get("Location"); got != "/" {
		t.Errorf("expected redirect to /, got %q", got)
	}
	if _, ok := srv.Cookie(t, "fuzion_session"); ok {
		t.Error("expected no session cookie after a denied login")
	}
}

func TestAuthCallback_ReturnsBadGateway_WhenDiscordRejectsTheCode(t *testing.T) {
	// given a login in progress but a code Discord does not recognise
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	state := startLogin(t, srv)

	// when the callback arrives with that code
	resp := srv.Get(t, "/api/auth/callback?code=bogus&state="+state)

	// then I expect a 502 and no session cookie
	resp.RequireStatus(t, 502)
	if _, ok := srv.Cookie(t, "fuzion_session"); ok {
		t.Error("expected no session cookie after a failed exchange")
	}
}

func TestAuthMe_ReturnsCurrentUser_WhenSessionCookieIsValid(t *testing.T) {
	// given a logged-in member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	user := srv.LoginAs(t, "80351110224678912", "thundermane")

	// when I ask who I am
	resp := srv.Get(t, "/api/auth/me")

	// then I expect my own profile back
	resp.RequireStatus(t, 200)
	var me meJSON
	resp.DecodeJSON(t, &me)
	want := meJSON{ID: user.ID.String(), Username: "thundermane", AvatarURL: nil}
	if diff := cmp.Diff(want, me); diff != "" {
		t.Errorf("unexpected /me response (-want +got):\n%s", diff)
	}
}

func TestAuthMe_ReturnsUnauthorized_WhenNoSessionCookie(t *testing.T) {
	// given nobody is logged in
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I ask who I am
	resp := srv.Get(t, "/api/auth/me")

	// then I expect a 401
	resp.RequireStatus(t, 401)
	var body map[string]string
	resp.DecodeJSON(t, &body)
	if diff := cmp.Diff(map[string]string{"error": "login required"}, body); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
}

func TestAuthMe_ReturnsUnauthorized_WhenSessionHasExpired(t *testing.T) {
	// given a member whose session expired an hour ago
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "80351110224678912", "thundermane")
	db.MustExec(`UPDATE sessions SET expires_at = now() - interval '1 hour'`)

	// when I ask who I am
	resp := srv.Get(t, "/api/auth/me")

	// then I expect a 401
	resp.RequireStatus(t, 401)
}

func TestAuthLogout_InvalidatesSession(t *testing.T) {
	// given a logged-in member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "80351110224678912", "thundermane")
	sessionCookie, _ := srv.Cookie(t, "fuzion_session")

	// when I log out
	resp := srv.Post(t, "/api/auth/logout", nil)

	// then I expect a 204, the cookie cleared, and the old token to be useless even if replayed
	resp.RequireStatus(t, 204)
	cleared := setCookie(t, resp, "fuzion_session")
	if cleared.Value != "" || cleared.MaxAge >= 0 {
		t.Errorf("expected the session cookie to be expired, got %s", cleared.String())
	}
	var sessions int
	if err := db.Get(&sessions, `SELECT count(*) FROM sessions`); err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if sessions != 0 {
		t.Errorf("expected the session row to be deleted, found %d", sessions)
	}

	srv.SetCookie(t, &http.Cookie{Name: "fuzion_session", Value: sessionCookie.Value, Path: "/"})
	replay := srv.Get(t, "/api/auth/me")
	if replay.StatusCode != 401 {
		t.Errorf("expected 401 from /me with the replayed cookie, got %d with body %s", replay.StatusCode, strings.TrimSpace(string(replay.Body)))
	}
}

func TestAuthLogout_ReturnsNoContent_WhenNobodyIsLoggedIn(t *testing.T) {
	// given nobody is logged in
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I log out anyway
	resp := srv.Post(t, "/api/auth/logout", nil)

	// then I expect a 204
	resp.RequireStatus(t, 204)
}
