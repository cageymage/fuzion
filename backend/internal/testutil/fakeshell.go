package testutil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// ShellPrefix and ShellSuffix surround the og block of the SPA shell served by
// FakeShell, so a test can assemble the page it expects as ShellPrefix + block + ShellSuffix.
const (
	ShellPrefix = `<!doctype html><html><head><meta charset="utf-8" /><!--og:start-->`
	ShellSuffix = `<!--og:end--><script type="module" src="/assets/index-abc123.js"></script></head><body><div id="root"></div></body></html>`

	defaultShellHead = "\n    <title>Fuzion — PvE · US</title>\n    "
)

// ShellHTML is the page FakeShell serves: the built index.html with its default tags.
const ShellHTML = ShellPrefix + defaultShellHead + ShellSuffix

// FakeShell stands in for the deployed static site, which serves the built
// index.html the API wraps per-post tags around.
type FakeShell struct {
	*httptest.Server

	// Status, when non-zero, makes every request fail with that status.
	Status int
}

func NewFakeShell(t *testing.T) *FakeShell {
	t.Helper()

	fake := &FakeShell{}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /index.html", fake.indexHTML)
	fake.Server = httptest.NewServer(mux)
	t.Cleanup(fake.Close)
	return fake
}

func (f *FakeShell) indexHTML(w http.ResponseWriter, _ *http.Request) {
	if f.Status != 0 {
		http.Error(w, "shell unavailable", f.Status)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(ShellHTML))
}
