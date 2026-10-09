package testutil

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

const webhookPath = "/api/webhooks/123456789/test-webhook-token"

// FakeWebhook stands in for a Discord channel webhook: it records every
// execute-webhook body and answers like Discord does, 204 normally and the
// created message when the caller passes wait=true.
type FakeWebhook struct {
	*httptest.Server

	// Status, when non-zero, makes every request fail with that status.
	Status int

	// EditStatus, when non-zero, makes only message edits fail with that status.
	EditStatus int

	mu     sync.Mutex
	bodies [][]byte
	edits  []WebhookEdit
}

// WebhookEdit is one edit-message request: which message and the raw JSON body.
type WebhookEdit struct {
	MessageID string
	Body      []byte
}

func NewFakeWebhook(t *testing.T) *FakeWebhook {
	t.Helper()

	fake := &FakeWebhook{}
	mux := http.NewServeMux()
	mux.HandleFunc("POST "+webhookPath, fake.execute)
	mux.HandleFunc("PATCH "+webhookPath+"/messages/{id}", fake.edit)
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

// MessageID is the id the fake assigns to the nth posted message (1-based).
func MessageID(n int) string {
	return fmt.Sprintf("msg-%d", n)
}

// Edits returns every edit-message request received so far.
func (f *FakeWebhook) Edits() []WebhookEdit {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]WebhookEdit(nil), f.edits...)
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
	if r.URL.Query().Get("wait") == "true" {
		writeJSON(w, http.StatusOK, map[string]any{"id": MessageID(len(f.bodies))})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (f *FakeWebhook) edit(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)

	f.mu.Lock()
	defer f.mu.Unlock()
	f.edits = append(f.edits, WebhookEdit{MessageID: r.PathValue("id"), Body: body})

	if status := max(f.Status, f.EditStatus); status != 0 {
		writeJSON(w, status, map[string]any{"message": "webhook is down"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": r.PathValue("id")})
}
