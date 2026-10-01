package streams_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/cageymage/fuzion/backend/internal/clock"
	"github.com/cageymage/fuzion/backend/internal/streams"
	"github.com/cageymage/fuzion/backend/internal/testutil"
	"github.com/cageymage/fuzion/backend/internal/youtube"
)

type suggestedVideoJSON struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

func TestGetSuggestedVideo_ReturnsPlaylistVideo_WhenPlaylistHasOneVideo(t *testing.T) {
	// given a playlist with a single video
	srv := testutil.NewServer(t, testutil.DB(t))
	srv.YouTube.SetPlaylist(testutil.YouTubeVideo{ID: "vid-1", Title: "Queen Ansurek kill", Privacy: "public"})

	// when I ask for a suggested video
	resp := srv.Get(t, "/api/streams/suggested-video")

	// then I expect a 200 with that video
	resp.RequireStatus(t, 200)
	var video suggestedVideoJSON
	resp.DecodeJSON(t, &video)
	want := suggestedVideoJSON{ID: "vid-1", Title: "Queen Ansurek kill"}
	if diff := cmp.Diff(want, video); diff != "" {
		t.Errorf("unexpected suggested video (-want +got):\n%s", diff)
	}
}

func TestGetSuggestedVideo_ReturnsAPlaylistVideo_WhenPlaylistHasSeveralVideos(t *testing.T) {
	// given a playlist with two videos
	srv := testutil.NewServer(t, testutil.DB(t))
	srv.YouTube.SetPlaylist(
		testutil.YouTubeVideo{ID: "vid-1", Title: "Queen Ansurek kill", Privacy: "public"},
		testutil.YouTubeVideo{ID: "vid-2", Title: "Week two recap", Privacy: "public"},
	)

	// when I ask for a suggested video
	resp := srv.Get(t, "/api/streams/suggested-video")

	// then I expect a 200 with one of the two
	resp.RequireStatus(t, 200)
	var video suggestedVideoJSON
	resp.DecodeJSON(t, &video)
	playlist := []suggestedVideoJSON{
		{ID: "vid-1", Title: "Queen Ansurek kill"},
		{ID: "vid-2", Title: "Week two recap"},
	}
	if video != playlist[0] && video != playlist[1] {
		t.Errorf("expected a video from the playlist, got %+v", video)
	}
}

func TestGetSuggestedVideo_ReturnsNoContent_WhenPlaylistHasNoPlayableVideos(t *testing.T) {
	// given a playlist holding only a private video
	srv := testutil.NewServer(t, testutil.DB(t))
	srv.YouTube.SetPlaylist(testutil.YouTubeVideo{ID: "vid-1", Title: "Private video", Privacy: "private"})

	// when I ask for a suggested video
	resp := srv.Get(t, "/api/streams/suggested-video")

	// then I expect a 204
	resp.RequireStatus(t, 204)
}

func TestGetSuggestedVideo_ReturnsNoContentWithoutCallingYouTube_WhenNoPlaylistIsConfigured(t *testing.T) {
	// given a server with no playlist configured
	srv := testutil.NewServer(t, testutil.DB(t), testutil.WithoutYouTubePlaylist())

	// when I ask for a suggested video
	resp := srv.Get(t, "/api/streams/suggested-video")

	// then I expect a 204 and no upstream request
	resp.RequireStatus(t, 204)
	if got := srv.YouTube.Requests(); got != 0 {
		t.Errorf("expected no YouTube requests, got %d", got)
	}
}

func TestGetSuggestedVideo_ReturnsBadGateway_WhenYouTubeAPIFails(t *testing.T) {
	// given a YouTube API that is down
	srv := testutil.NewServer(t, testutil.DB(t))
	srv.YouTube.Status = 503

	// when I ask for a suggested video
	resp := srv.Get(t, "/api/streams/suggested-video")

	// then I expect a 502 with the following error
	resp.RequireStatus(t, 502)
	var body map[string]string
	resp.DecodeJSON(t, &body)
	want := map[string]string{"error": "suggested video could not be loaded"}
	if diff := cmp.Diff(want, body); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
}

func TestGetSuggestedVideo_CallsYouTubeOnce_WhenAskedTwiceWithinTheCacheWindow(t *testing.T) {
	// given a playlist with a video
	srv := testutil.NewServer(t, testutil.DB(t))
	srv.YouTube.SetPlaylist(testutil.YouTubeVideo{ID: "vid-1", Title: "Queen Ansurek kill", Privacy: "public"})

	// when I ask for a suggested video twice
	srv.Get(t, "/api/streams/suggested-video").RequireStatus(t, 200)
	srv.Get(t, "/api/streams/suggested-video").RequireStatus(t, 200)

	// then I expect a single upstream request
	if got := srv.YouTube.Requests(); got != 1 {
		t.Errorf("expected 1 YouTube request, got %d", got)
	}
}

type steppingClock struct{ now time.Time }

func (c *steppingClock) Now() time.Time { return c.now }

var _ clock.Clock = (*steppingClock)(nil)

func TestSuggestedVideos_RefetchesPlaylist_WhenCacheHasExpired(t *testing.T) {
	// given a playlist that was fetched and cached
	fake := testutil.NewFakeYouTube(t)
	fake.SetPlaylist(testutil.YouTubeVideo{ID: "vid-1", Title: "Queen Ansurek kill", Privacy: "public"})
	client := youtube.NewClient(youtube.Config{APIKey: testutil.YouTubeAPIKey, BaseURL: fake.URL}, fake.Client())
	clk := &steppingClock{now: testutil.FixedNow}
	videos := streams.NewSuggestedVideos(client, testutil.YouTubePlaylistID, clk)
	if _, _, err := videos.Suggest(context.Background()); err != nil {
		t.Fatalf("first Suggest: %v", err)
	}

	// when the cache window has passed and I ask again
	clk.now = clk.now.Add(streams.SuggestedVideoCacheTTL + time.Minute)
	if _, _, err := videos.Suggest(context.Background()); err != nil {
		t.Fatalf("second Suggest: %v", err)
	}

	// then I expect a second upstream request
	if got := fake.Requests(); got != 2 {
		t.Errorf("expected 2 YouTube requests, got %d", got)
	}
}
