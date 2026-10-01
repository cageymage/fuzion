package testutil

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

const webhookPath = "/api/webhooks/123456789/test-webhook-token"

// FakeWebhook stands in for a Discord channel webhook: it records every
// execute-webhook body and answers 204 like Discord does.
type FakeWebhook struct {
	*httptest.Server

	// Status, when non-zero, makes every request fail with that status.
	Status int

	mu     sync.Mutex
	bodies [][]byte
}

func NewFakeWebhook(t *testing.T) *FakeWebhook {
	t.Helper()

	fake := &FakeWebhook{}
	mux := http.NewServeMux()
	mux.HandleFunc("POST "+webhookPath, fake.execute)
	fake.Server = httptest.NewServer(mux)
	t.Cleanup(fake.Close)
	return fake
}

func (f *FakeWebhook) WebhookURL() string {
	return f.URL + webhookPath
}

// Bodies returns the raw JSON of every message posted so far.
func (f *FakeWebhook) Bodies() [][]byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][]byte(nil), f.bodies...)
}

func (f *FakeWebhook) execute(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)

	f.mu.Lock()
	defer f.mu.Unlock()
	f.bodies = append(f.bodies, body)

	if f.Status != 0 {
		writeJSON(w, f.Status, map[string]any{"message": "webhook is down"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
