package testutil

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
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

// YouTubeChannel is a channel the fake knows, with its uploads newest first.
type YouTubeChannel struct {
	ID        string
	AvatarURL string
	Uploads   []YouTubeUpload
}

type YouTubeUpload struct {
	ID           string
	Title        string
	Live         bool
	Viewers      int
	ThumbnailURL string
}

// FakeYouTube stands in for the YouTube Data API's playlistItems.list, videos.list
// and channels.list. It knows the suggested-video playlist, YouTubePlaylistID, plus
// the uploads playlist of every channel added with AddChannel, and rejects any other
// key or playlist the way YouTube does.
type FakeYouTube struct {
	*httptest.Server

	// Status, when non-zero, makes every request fail with that status.
	Status int
	// QuotaExceeded makes every request fail the way YouTube does once the
	// project's daily quota is spent.
	QuotaExceeded bool
	// PageSize is how many items one response holds; it defaults to 50.
	PageSize int

	mu           sync.Mutex
	videos       []YouTubeVideo
	channels     []YouTubeChannel
	videoBatches [][]string
	requests     int
}

func NewFakeYouTube(t *testing.T) *FakeYouTube {
	t.Helper()

	fake := &FakeYouTube{PageSize: 50}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /youtube/v3/playlistItems", fake.playlistItems)
	mux.HandleFunc("GET /youtube/v3/videos", fake.videosList)
	mux.HandleFunc("GET /youtube/v3/channels", fake.channelsList)
	fake.Server = httptest.NewServer(mux)
	t.Cleanup(fake.Close)
	return fake
}

func (f *FakeYouTube) SetPlaylist(videos ...YouTubeVideo) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.videos = videos
}

func (f *FakeYouTube) AddChannel(channel YouTubeChannel) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.channels = append(f.channels, channel)
}

// VideoBatches returns the video IDs of each videos.list request, in order.
func (f *FakeYouTube) VideoBatches() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.videoBatches
}

func (f *FakeYouTube) Requests() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.requests
}

// rejected writes the error response YouTube would send and reports whether it did.
// The caller must hold f.mu.
func (f *FakeYouTube) rejected(w http.ResponseWriter, r *http.Request) bool {
	f.requests++

	if f.Status != 0 {
		writeJSON(w, f.Status, map[string]any{"error": map[string]any{"code": f.Status, "message": "youtube is down"}})
		return true
	}
	if r.URL.Query().Get("key") != YouTubeAPIKey {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": map[string]any{"code": 403, "message": "API key not valid"}})
		return true
	}
	if f.QuotaExceeded {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": map[string]any{
			"code":    403,
			"message": "The request cannot be completed because you have exceeded your quota.",
			"errors":  []map[string]any{{"reason": "quotaExceeded", "domain": "youtube.quota"}},
		}})
		return true
	}
	return false
}

func (f *FakeYouTube) playlistItems(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.rejected(w, r) {
		return
	}
	q := r.URL.Query()
	for _, channel := range f.channels {
		if q.Get("playlistId") == "UU"+strings.TrimPrefix(channel.ID, "UC") {
			f.writeUploads(w, channel, q.Get("maxResults"))
			return
		}
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

func (f *FakeYouTube) writeUploads(w http.ResponseWriter, channel YouTubeChannel, maxResults string) {
	limit, err := strconv.Atoi(maxResults)
	if err != nil {
		limit = 5
	}
	uploads := channel.Uploads[:min(limit, len(channel.Uploads))]
	items := make([]map[string]any, 0, len(uploads))
	for _, u := range uploads {
		items = append(items, map[string]any{"contentDetails": map[string]any{"videoId": u.ID}})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (f *FakeYouTube) videosList(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.rejected(w, r) {
		return
	}
	ids := strings.Split(r.URL.Query().Get("id"), ",")
	if len(ids) > 50 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]any{"code": 400, "message": "too many ids"}})
		return
	}
	f.videoBatches = append(f.videoBatches, ids)

	items := []map[string]any{}
	for _, id := range ids {
		for _, channel := range f.channels {
			for _, u := range channel.Uploads {
				if u.ID != id {
					continue
				}
				broadcast := "none"
				if u.Live {
					broadcast = "live"
				}
				item := map[string]any{
					"id": u.ID,
					"snippet": map[string]any{
						"channelId":            channel.ID,
						"title":                u.Title,
						"liveBroadcastContent": broadcast,
						"thumbnails":           map[string]any{"medium": map[string]any{"url": u.ThumbnailURL}},
					},
				}
				if u.Live {
					item["liveStreamingDetails"] = map[string]any{"concurrentViewers": strconv.Itoa(u.Viewers)}
				}
				items = append(items, item)
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (f *FakeYouTube) channelsList(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.rejected(w, r) {
		return
	}
	ids := strings.Split(r.URL.Query().Get("id"), ",")
	if len(ids) > 50 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]any{"code": 400, "message": "too many ids"}})
		return
	}
	items := []map[string]any{}
	for _, id := range ids {
		for _, channel := range f.channels {
			if channel.ID == id {
				items = append(items, map[string]any{
					"id":      channel.ID,
					"snippet": map[string]any{"thumbnails": map[string]any{"medium": map[string]any{"url": channel.AvatarURL}}},
				})
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
