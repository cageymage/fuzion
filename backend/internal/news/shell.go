package news

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/cageymage/fuzion/backend/internal/clock"
)

const (
	shellTTL      = 5 * time.Minute
	maxShellBytes = 1 << 20
)

// ShellFetcher loads the static site's built index.html, which carries the hashed asset
// URLs a news page needs, and caches it so a post view does not wait on the static site.
type ShellFetcher struct {
	url    string
	client *http.Client
	clock  clock.Clock

	mu        sync.Mutex
	html      string
	fetchedAt time.Time
}

func NewShellFetcher(url string, client *http.Client, clock clock.Clock) *ShellFetcher {
	return &ShellFetcher{url: url, client: client, clock: clock}
}

func (f *ShellFetcher) Get(ctx context.Context) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.html != "" && f.clock.Now().Sub(f.fetchedAt) < shellTTL {
		return f.html, nil
	}
	html, err := f.fetch(ctx)
	if err != nil {
		return "", err
	}
	f.html = html
	f.fetchedAt = f.clock.Now()
	return html, nil
}

func (f *ShellFetcher) fetch(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.url, nil)
	if err != nil {
		return "", fmt.Errorf("build shell request: %w", err)
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch shell: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch shell: unexpected status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxShellBytes))
	if err != nil {
		return "", fmt.Errorf("read shell: %w", err)
	}
	return string(body), nil
}
