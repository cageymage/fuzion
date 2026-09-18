package testutil

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/jmoiron/sqlx"

	"github.com/cageymage/fuzion/backend/internal/news"
	"github.com/cageymage/fuzion/backend/internal/raids"
	"github.com/cageymage/fuzion/backend/internal/server"
	"github.com/cageymage/fuzion/backend/internal/streams"
)

type Server struct {
	*httptest.Server
}

func NewServer(t *testing.T, db *sqlx.DB) *Server {
	t.Helper()

	router := server.New(server.Deps{
		News:           news.NewHandler(news.NewService(news.NewRepo(db))),
		Raids:          raids.NewHandler(raids.NewService(raids.NewRepo(db))),
		Streams:        streams.NewHandler(streams.NewService(streams.NewRepo(db))),
		AllowedOrigins: []string{"http://localhost:5173"},
	})

	httpServer := httptest.NewServer(router)
	t.Cleanup(httpServer.Close)

	return &Server{Server: httpServer}
}

type Response struct {
	StatusCode int
	Body       []byte
}

func (s *Server) Get(t *testing.T, path string) Response {
	t.Helper()

	resp, err := s.Client().Get(s.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body of GET %s: %v", path, err)
	}

	return Response{StatusCode: resp.StatusCode, Body: body}
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
