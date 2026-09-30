package youtube_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/cageymage/fuzion/backend/internal/testutil"
	"github.com/cageymage/fuzion/backend/internal/youtube"
)

func newClient(fake *testutil.FakeYouTube, apiKey string) *youtube.Client {
	return youtube.NewClient(youtube.Config{APIKey: apiKey, BaseURL: fake.URL}, fake.Client())
}

func TestPlaylistVideos_ReturnsVideos_WhenPlaylistHasItems(t *testing.T) {
	// given a playlist with two public videos
	fake := testutil.NewFakeYouTube(t)
	fake.SetPlaylist(
		testutil.YouTubeVideo{ID: "vid-1", Title: "Queen Ansurek kill", Privacy: "public"},
		testutil.YouTubeVideo{ID: "vid-2", Title: "Week two recap", Privacy: "public"},
	)

	// when I fetch the playlist
	videos, err := newClient(fake, testutil.YouTubeAPIKey).PlaylistVideos(context.Background(), testutil.YouTubePlaylistID)

	// then I expect both videos in playlist order
	if err != nil {
		t.Fatalf("PlaylistVideos: %v", err)
	}
	want := []youtube.Video{
		{ID: "vid-1", Title: "Queen Ansurek kill"},
		{ID: "vid-2", Title: "Week two recap"},
	}
	if diff := cmp.Diff(want, videos); diff != "" {
		t.Errorf("unexpected videos (-want +got):\n%s", diff)
	}
}

func TestPlaylistVideos_ReturnsEveryVideo_WhenPlaylistSpansMultiplePages(t *testing.T) {
	// given a playlist of three videos served two per page
	fake := testutil.NewFakeYouTube(t)
	fake.PageSize = 2
	fake.SetPlaylist(
		testutil.YouTubeVideo{ID: "vid-1", Title: "One", Privacy: "public"},
		testutil.YouTubeVideo{ID: "vid-2", Title: "Two", Privacy: "public"},
		testutil.YouTubeVideo{ID: "vid-3", Title: "Three", Privacy: "public"},
	)

	// when I fetch the playlist
	videos, err := newClient(fake, testutil.YouTubeAPIKey).PlaylistVideos(context.Background(), testutil.YouTubePlaylistID)

	// then I expect all three videos from two upstream requests
	if err != nil {
		t.Fatalf("PlaylistVideos: %v", err)
	}
	want := []youtube.Video{
		{ID: "vid-1", Title: "One"},
		{ID: "vid-2", Title: "Two"},
		{ID: "vid-3", Title: "Three"},
	}
	if diff := cmp.Diff(want, videos); diff != "" {
		t.Errorf("unexpected videos (-want +got):\n%s", diff)
	}
	if got := fake.Requests(); got != 2 {
		t.Errorf("expected 2 upstream requests, got %d", got)
	}
}

func TestPlaylistVideos_SkipsVideos_WhenTheyAreNotPlayable(t *testing.T) {
	// given a playlist with a private video, an unlisted video and a deleted one
	fake := testutil.NewFakeYouTube(t)
	fake.SetPlaylist(
		testutil.YouTubeVideo{ID: "vid-private", Title: "Private video", Privacy: "private"},
		testutil.YouTubeVideo{ID: "vid-unlisted", Title: "Officer meeting", Privacy: "unlisted"},
		testutil.YouTubeVideo{ID: "vid-deleted", Title: "Deleted video", Privacy: ""},
	)

	// when I fetch the playlist
	videos, err := newClient(fake, testutil.YouTubeAPIKey).PlaylistVideos(context.Background(), testutil.YouTubePlaylistID)

	// then I expect only the unlisted video, which still embeds
	if err != nil {
		t.Fatalf("PlaylistVideos: %v", err)
	}
	want := []youtube.Video{{ID: "vid-unlisted", Title: "Officer meeting"}}
	if diff := cmp.Diff(want, videos); diff != "" {
		t.Errorf("unexpected videos (-want +got):\n%s", diff)
	}
}

func TestPlaylistVideos_ReturnsEmptyList_WhenPlaylistHasNoVideos(t *testing.T) {
	// given an empty playlist
	fake := testutil.NewFakeYouTube(t)

	// when I fetch the playlist
	videos, err := newClient(fake, testutil.YouTubeAPIKey).PlaylistVideos(context.Background(), testutil.YouTubePlaylistID)

	// then I expect no videos and no error
	if err != nil {
		t.Fatalf("PlaylistVideos: %v", err)
	}
	if len(videos) != 0 {
		t.Errorf("expected no videos, got %v", videos)
	}
}

func TestPlaylistVideos_ReturnsError_WhenYouTubeAPIFails(t *testing.T) {
	// given a YouTube API that is down
	fake := testutil.NewFakeYouTube(t)
	fake.Status = 503

	// when I fetch the playlist
	_, err := newClient(fake, testutil.YouTubeAPIKey).PlaylistVideos(context.Background(), testutil.YouTubePlaylistID)

	// then I expect an error naming the status
	if err == nil || !strings.Contains(err.Error(), "unexpected status 503") {
		t.Fatalf("expected an unexpected-status-503 error, got %v", err)
	}
}

func TestPlaylistVideos_ReturnsErrorWithoutLeakingKey_WhenKeyIsRejected(t *testing.T) {
	// given a client configured with a bad API key
	fake := testutil.NewFakeYouTube(t)

	// when I fetch the playlist
	_, err := newClient(fake, "wrong-key").PlaylistVideos(context.Background(), testutil.YouTubePlaylistID)

	// then I expect a 403 error that does not contain the key
	if err == nil || !strings.Contains(err.Error(), "unexpected status 403") {
		t.Fatalf("expected an unexpected-status-403 error, got %v", err)
	}
	if strings.Contains(err.Error(), "wrong-key") {
		t.Errorf("error leaks the API key: %v", err)
	}
}
