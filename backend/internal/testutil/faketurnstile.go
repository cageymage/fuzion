package testutil

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

const (
	TurnstileSecretKey  = "test-turnstile-secret"
	ValidTurnstileToken = "valid-turnstile-token"
)

type TurnstileRequest struct {
	Secret   string
	Response string
	RemoteIP string
}

// FakeTurnstile stands in for Cloudflare's siteverify endpoint: it accepts
// ValidTurnstileToken with TurnstileSecretKey and rejects everything else the
// way Cloudflare does, with a 200 and success=false.
type FakeTurnstile struct {
	*httptest.Server

	// Status, when non-zero, makes every request fail with that status.
	Status int

	mu       sync.Mutex
	requests []TurnstileRequest
}

func NewFakeTurnstile(t *testing.T) *FakeTurnstile {
	t.Helper()

	fake := &FakeTurnstile{}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /turnstile/v0/siteverify", fake.siteverify)
	fake.Server = httptest.NewServer(mux)
	t.Cleanup(fake.Close)
	return fake
}

func (f *FakeTurnstile) Requests() []TurnstileRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]TurnstileRequest(nil), f.requests...)
}

func (f *FakeTurnstile) siteverify(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error-codes": []string{"bad-request"}})
		return
	}
	req := TurnstileRequest{
		Secret:   r.PostForm.Get("secret"),
		Response: r.PostForm.Get("response"),
		RemoteIP: r.PostForm.Get("remoteip"),
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, req)

	if f.Status != 0 {
		writeJSON(w, f.Status, map[string]any{"success": false, "error-codes": []string{"internal-error"}})
		return
	}
	if req.Secret != TurnstileSecretKey {
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error-codes": []string{"invalid-input-secret"}})
		return
	}
	if req.Response != ValidTurnstileToken {
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error-codes": []string{"invalid-input-response"}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "error-codes": []string{}})
}
