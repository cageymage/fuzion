package youtube_test

import (
	"context"
	"fmt"
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

func TestLiveStreams_ReturnsLiveChannels_WhenYouTubeResponds(t *testing.T) {
	// given one channel that is live and one whose latest upload is an old video
	fake := testutil.NewFakeYouTube(t)
	fake.AddChannel(testutil.YouTubeChannel{ID: "UCemberfist", Uploads: []testutil.YouTubeUpload{
		{ID: "live-1", Title: "Forever raid night", Live: true, Viewers: 312, ThumbnailURL: "https://i.ytimg.example/live-1/mqdefault_live.jpg"},
	}})
	fake.AddChannel(testutil.YouTubeChannel{ID: "UCmoonveil", Uploads: []testutil.YouTubeUpload{
		{ID: "vod-1", Title: "Last week's highlights"},
	}})

	// when I ask which channels are live
	live, err := newClient(fake, testutil.YouTubeAPIKey).LiveStreams(context.Background(), []string{"UCemberfist", "UCmoonveil"})

	// then I expect only the live channel, with its title, viewers and thumbnail
	if err != nil {
		t.Fatalf("LiveStreams: %v", err)
	}
	want := []youtube.LiveStream{{
		ChannelID:    "UCemberfist",
		VideoID:      "live-1",
		Title:        "Forever raid night",
		ViewerCount:  312,
		ThumbnailURL: "https://i.ytimg.example/live-1/mqdefault_live.jpg",
	}}
	if diff := cmp.Diff(want, live); diff != "" {
		t.Errorf("unexpected live streams (-want +got):\n%s", diff)
	}
}

func TestLiveStreams_FindsLiveStream_WhenNewerUploadsArePostedAboveIt(t *testing.T) {
	// given a channel that posted two Shorts while its stream is running
	fake := testutil.NewFakeYouTube(t)
	fake.AddChannel(testutil.YouTubeChannel{ID: "UCemberfist", Uploads: []testutil.YouTubeUpload{
		{ID: "short-2", Title: "Clip of the pull"},
		{ID: "short-1", Title: "Another clip"},
		{ID: "live-1", Title: "Forever raid night", Live: true, Viewers: 312},
	}})

	// when I ask which channels are live
	live, err := newClient(fake, testutil.YouTubeAPIKey).LiveStreams(context.Background(), []string{"UCemberfist"})

	// then I expect the stream to be found below the Shorts
	if err != nil {
		t.Fatalf("LiveStreams: %v", err)
	}
	if len(live) != 1 || live[0].VideoID != "live-1" {
		t.Errorf("expected the live-1 stream, got %+v", live)
	}
}

func TestLiveStreams_SplitsIdsIntoBatchesOf50_WhenMoreThan50AreRequested(t *testing.T) {
	// given six channels with ten uploads each, 60 videos in all
	fake := testutil.NewFakeYouTube(t)
	channelIDs := []string{}
	for c := range 6 {
		channel := testutil.YouTubeChannel{ID: fmt.Sprintf("UCchannel%d", c)}
		for v := range 10 {
			channel.Uploads = append(channel.Uploads, testutil.YouTubeUpload{ID: fmt.Sprintf("vid-%d-%d", c, v)})
		}
		fake.AddChannel(channel)
		channelIDs = append(channelIDs, channel.ID)
	}

	// when I ask which channels are live
	_, err := newClient(fake, testutil.YouTubeAPIKey).LiveStreams(context.Background(), channelIDs)

	// then I expect the videos looked up in a batch of 50 and a batch of 10
	if err != nil {
		t.Fatalf("LiveStreams: %v", err)
	}
	batches := fake.VideoBatches()
	if len(batches) != 2 || len(batches[0]) != 50 || len(batches[1]) != 10 {
		sizes := []int{}
		for _, b := range batches {
			sizes = append(sizes, len(b))
		}
		t.Errorf("expected video batches of [50 10], got %v", sizes)
	}
}

func TestLiveStreams_ReturnsError_WhenQuotaIsExceeded(t *testing.T) {
	// given a YouTube project whose daily quota is spent
	fake := testutil.NewFakeYouTube(t)
	fake.AddChannel(testutil.YouTubeChannel{ID: "UCemberfist"})
	fake.QuotaExceeded = true

	// when I ask which channels are live
	_, err := newClient(fake, testutil.YouTubeAPIKey).LiveStreams(context.Background(), []string{"UCemberfist"})

	// then I expect an error naming the quota
	if err == nil || !strings.Contains(err.Error(), "quotaExceeded") {
		t.Fatalf("expected a quotaExceeded error, got %v", err)
	}
}

func TestAvatars_ReturnsAvatarsByChannelId_WhenYouTubeResponds(t *testing.T) {
	// given two channels with avatars
	fake := testutil.NewFakeYouTube(t)
	fake.AddChannel(testutil.YouTubeChannel{ID: "UCemberfist", AvatarURL: "https://yt3.example/emberfist.jpg"})
	fake.AddChannel(testutil.YouTubeChannel{ID: "UCmoonveil", AvatarURL: "https://yt3.example/moonveil.jpg"})

	// when I ask for their avatars along with an unknown channel
	avatars, err := newClient(fake, testutil.YouTubeAPIKey).Avatars(context.Background(), []string{"UCemberfist", "UCmoonveil", "UCunknown"})

	// then I expect the two known avatars keyed by channel ID
	if err != nil {
		t.Fatalf("Avatars: %v", err)
	}
	want := map[string]string{
		"UCemberfist": "https://yt3.example/emberfist.jpg",
		"UCmoonveil":  "https://yt3.example/moonveil.jpg",
	}
	if diff := cmp.Diff(want, avatars); diff != "" {
		t.Errorf("unexpected avatars (-want +got):\n%s", diff)
	}
}

func TestAvatars_SplitsIdsIntoBatchesOf50_WhenMoreThan50AreRequested(t *testing.T) {
	// given 51 channels, the last of which has an avatar
	fake := testutil.NewFakeYouTube(t)
	channelIDs := []string{}
	for c := range 51 {
		channelIDs = append(channelIDs, fmt.Sprintf("UCchannel%d", c))
	}
	fake.AddChannel(testutil.YouTubeChannel{ID: "UCchannel50", AvatarURL: "https://yt3.example/fifty-first.jpg"})

	// when I ask for all 51 avatars
	avatars, err := newClient(fake, testutil.YouTubeAPIKey).Avatars(context.Background(), channelIDs)

	// then I expect no error from YouTube's 50-ID limit and the 51st channel's avatar
	if err != nil {
		t.Fatalf("Avatars: %v", err)
	}
	if avatars["UCchannel50"] != "https://yt3.example/fifty-first.jpg" {
		t.Errorf("expected the 51st channel's avatar, got %v", avatars)
	}
}

func TestLiveStreams_SkipsChannel_WhenItHasNoUploadsPlaylist(t *testing.T) {
	// given one live channel and one channel ID YouTube has no uploads playlist for
	fake := testutil.NewFakeYouTube(t)
	fake.AddChannel(testutil.YouTubeChannel{ID: "UCemberfist", Uploads: []testutil.YouTubeUpload{
		{ID: "live-1", Title: "Forever raid night", Live: true, Viewers: 312},
	}})

	// when I ask which channels are live
	live, err := newClient(fake, testutil.YouTubeAPIKey).LiveStreams(context.Background(), []string{"UCnouploads", "UCemberfist"})

	// then I expect the live channel to be returned without an error
	if err != nil {
		t.Fatalf("LiveStreams: %v", err)
	}
	if len(live) != 1 || live[0].ChannelID != "UCemberfist" {
		t.Errorf("expected only UCemberfist live, got %+v", live)
	}
}
