package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/cageymage/fuzion/backend/internal/applications"
	"github.com/cageymage/fuzion/backend/internal/auth"
	"github.com/cageymage/fuzion/backend/internal/news"
	"github.com/cageymage/fuzion/backend/internal/raids"
	"github.com/cageymage/fuzion/backend/internal/server"
	"github.com/cageymage/fuzion/backend/internal/streams"
)

type Server struct {
	*httptest.Server
	Discord *FakeDiscord

	db     *sqlx.DB
	client *http.Client
}

func NewServer(t *testing.T, db *sqlx.DB) *Server {
	t.Helper()

	discord := NewFakeDiscord(t)
	provider := auth.NewDiscord(auth.DiscordConfig{
		ClientID:     DiscordClientID,
		ClientSecret: DiscordClientSecret,
		RedirectURL:  DiscordRedirectURL,
		BaseURL:      discord.URL,
	}, discord.Client())

	router := server.New(server.Deps{
		Applications:   applications.NewHandler(applications.NewService(applications.NewRepo(db))),
		Auth:           auth.NewHandler(auth.NewService(provider, auth.NewRepo(db))),
		News:           news.NewHandler(news.NewService(news.NewRepo(db))),
		Raids:          raids.NewHandler(raids.NewService(raids.NewRepo(db))),
		Streams:        streams.NewHandler(streams.NewService(streams.NewRepo(db))),
		AllowedOrigins: []string{"http://localhost:5173"},
	})

	httpServer := httptest.NewServer(router)
	t.Cleanup(httpServer.Close)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("create cookie jar: %v", err)
	}
	// The client keeps cookies like a browser tab would, but stops at redirects
	// so tests can assert on the 302 and its Location.
	client := httpServer.Client()
	client.Jar = jar
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	return &Server{Server: httpServer, Discord: discord, db: db, client: client}
}

// LoginAs inserts a user and a live session for them straight into the
// database and puts the session cookie in the client's jar, so a test can act
// as a logged-in member without walking the OAuth dance.
func (s *Server) LoginAs(t *testing.T, discordID, username string) auth.User {
	t.Helper()

	var user auth.User
	err := s.db.Get(&user, `
		INSERT INTO users (id, discord_id, username)
		VALUES ($1, $2, $3)
		RETURNING id, discord_id, username, avatar_url, created_at, last_seen_at`,
		uuid.New(), discordID, username)
	if err != nil {
		t.Fatalf("insert user %q: %v", username, err)
	}

	token := "session-for-" + discordID
	s.db.MustExec(`INSERT INTO sessions (token, user_id, expires_at) VALUES ($1, $2, $3)`,
		token, user.ID, time.Now().Add(time.Hour))
	s.SetCookie(t, &http.Cookie{Name: auth.SessionCookieName, Value: token, Path: "/"})
	return user
}

func (s *Server) SetCookie(t *testing.T, cookie *http.Cookie) {
	t.Helper()

	u, err := url.Parse(s.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	s.client.Jar.SetCookies(u, []*http.Cookie{cookie})
}

func (s *Server) Cookie(t *testing.T, name string) (*http.Cookie, bool) {
	t.Helper()

	u, err := url.Parse(s.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	for _, c := range s.client.Jar.Cookies(u) {
		if c.Name == name {
			return c, true
		}
	}
	return nil, false
}

type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

func (s *Server) Get(t *testing.T, path string) Response {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, s.URL+path, nil)
	if err != nil {
		t.Fatalf("build GET %s: %v", path, err)
	}
	return s.do(t, req)
}

func (s *Server) Post(t *testing.T, path string, body any) Response {
	t.Helper()

	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("encode body of POST %s: %v", path, err)
		}
		payload = bytes.NewReader(encoded)
	}
	req, err := http.NewRequest(http.MethodPost, s.URL+path, payload)
	if err != nil {
		t.Fatalf("build POST %s: %v", path, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return s.do(t, req)
}

func (s *Server) PostRaw(t *testing.T, path, body string) Response {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, s.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatalf("build POST %s: %v", path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	return s.do(t, req)
}

func (s *Server) do(t *testing.T, req *http.Request) Response {
	t.Helper()

	resp, err := s.client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", req.Method, req.URL.Path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body of %s %s: %v", req.Method, req.URL.Path, err)
	}

	return Response{StatusCode: resp.StatusCode, Header: resp.Header, Body: body}
}

func (r Response) DecodeJSON(t *testing.T, target any) {
	t.Helper()

	if err := json.Unmarshal(r.Body, target); err != nil {
		t.Fatalf("decode JSON response %q: %v", r.Body, err)
	}
}

func (r Response) RequireStatus(t *testing.T, want int) {
	t.Helper()

	if r.StatusCode != want {
		t.Fatalf("expected status %d, got %d with body %q", want, r.StatusCode, r.Body)
	}
}
