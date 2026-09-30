package testutil

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
)

const (
	YouTubeAPIKey     = "test-youtube-key"
	YouTubePlaylistID = "PLtestplaylist"
)

type YouTubeVideo struct {
	ID      string
	Title   string
	Privacy string
}

// FakeYouTube stands in for the YouTube Data API's playlistItems.list. It only
// knows one playlist, YouTubePlaylistID, and rejects any other key or playlist
// the way YouTube does.
type FakeYouTube struct {
	*httptest.Server

	// Status, when non-zero, makes every request fail with that status.
	Status int
	// PageSize is how many items one response holds; it defaults to 50.
	PageSize int

	mu       sync.Mutex
	videos   []YouTubeVideo
	requests int
}

func NewFakeYouTube(t *testing.T) *FakeYouTube {
	t.Helper()

	fake := &FakeYouTube{PageSize: 50}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /youtube/v3/playlistItems", fake.playlistItems)
	fake.Server = httptest.NewServer(mux)
	t.Cleanup(fake.Close)
	return fake
}

func (f *FakeYouTube) SetPlaylist(videos ...YouTubeVideo) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.videos = videos
}

func (f *FakeYouTube) Requests() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.requests
}

func (f *FakeYouTube) playlistItems(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests++

	if f.Status != 0 {
		writeJSON(w, f.Status, map[string]any{"error": map[string]any{"code": f.Status, "message": "youtube is down"}})
		return
	}
	q := r.URL.Query()
	if q.Get("key") != YouTubeAPIKey {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": map[string]any{"code": 403, "message": "API key not valid"}})
		return
	}
	if q.Get("playlistId") != YouTubePlaylistID {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": map[string]any{"code": 404, "message": "playlistNotFound"}})
		return
	}

	start, _ := strconv.Atoi(q.Get("pageToken"))
	end := min(start+f.PageSize, len(f.videos))
	items := make([]map[string]any, 0, end-start)
	for _, v := range f.videos[start:end] {
		items = append(items, map[string]any{
			"snippet": map[string]any{
				"title":      v.Title,
				"resourceId": map[string]any{"videoId": v.ID},
			},
			"status": map[string]any{"privacyStatus": v.Privacy},
		})
	}
	body := map[string]any{"items": items}
	if end < len(f.videos) {
		body["nextPageToken"] = strconv.Itoa(end)
	}
	writeJSON(w, http.StatusOK, body)
}
