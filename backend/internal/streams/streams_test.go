package streams_test

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/cageymage/fuzion/backend/internal/testutil"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.Run(m))
}

type streamJSON struct {
	ID           string  `json:"id"`
	StreamerName string  `json:"streamerName"`
	GameName     string  `json:"gameName"`
	ViewerCount  int     `json:"viewerCount"`
	ThumbnailURL *string `json:"thumbnailUrl"`
	ChannelURL   string  `json:"channelUrl"`
	IsLive       bool    `json:"isLive"`
}

func TestListLiveStreams_ReturnsLiveStreamersOrderedByViewerCount(t *testing.T) {
	// given two live streamers with different audiences
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO streams (id, streamer_name, game_name, viewer_count, thumbnail_url, channel_url, is_live)
		VALUES
			('11111111-1111-1111-1111-111111111111', 'Emberfist', 'World of Warcraft: Forever', 310, NULL, 'https://twitch.tv/emberfist', true),
			('22222222-2222-2222-2222-222222222222', 'Thundermane', 'World of Warcraft: Forever', 1240, 'https://cdn.example/thumb.jpg', 'https://twitch.tv/thundermane', true)`)

	// when I ask who is live
	resp := srv.Get(t, "/api/streams/live")

	// then I expect a 200 with the biggest stream first
	resp.RequireStatus(t, 200)
	var live []streamJSON
	resp.DecodeJSON(t, &live)

	thumbnailURL := "https://cdn.example/thumb.jpg"
	want := []streamJSON{
		{
			ID:           "22222222-2222-2222-2222-222222222222",
			StreamerName: "Thundermane",
			GameName:     "World of Warcraft: Forever",
			ViewerCount:  1240,
			ThumbnailURL: &thumbnailURL,
			ChannelURL:   "https://twitch.tv/thundermane",
			IsLive:       true,
		},
		{
			ID:           "11111111-1111-1111-1111-111111111111",
			StreamerName: "Emberfist",
			GameName:     "World of Warcraft: Forever",
			ViewerCount:  310,
			ThumbnailURL: nil,
			ChannelURL:   "https://twitch.tv/emberfist",
			IsLive:       true,
		},
	}
	if diff := cmp.Diff(want, live); diff != "" {
		t.Errorf("unexpected live streams (-want +got):\n%s", diff)
	}
}

func TestListLiveStreams_ExcludesStreamersWhoAreOffline(t *testing.T) {
	// given one live streamer and one offline streamer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO streams (id, streamer_name, game_name, viewer_count, thumbnail_url, channel_url, is_live)
		VALUES
			('11111111-1111-1111-1111-111111111111', 'Thundermane', 'World of Warcraft: Forever', 1240, NULL, 'https://twitch.tv/thundermane', true),
			('22222222-2222-2222-2222-222222222222', 'Emberfist', 'World of Warcraft: Forever', 0, NULL, 'https://twitch.tv/emberfist', false)`)

	// when I ask who is live
	resp := srv.Get(t, "/api/streams/live")

	// then I expect only the live streamer back
	resp.RequireStatus(t, 200)
	var live []streamJSON
	resp.DecodeJSON(t, &live)

	if len(live) != 1 {
		t.Fatalf("expected exactly one live stream, got %d: %q", len(live), resp.Body)
	}
	if live[0].StreamerName != "Thundermane" {
		t.Errorf("expected Thundermane to be the only live streamer, got %q", live[0].StreamerName)
	}
	if !live[0].IsLive {
		t.Errorf("expected Thundermane to be marked live")
	}
}

func TestListLiveStreams_ReturnsEmptyArray_WhenNobodyIsLive(t *testing.T) {
	// given a guild whose streamers are all offline
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO streams (id, streamer_name, game_name, viewer_count, thumbnail_url, channel_url, is_live)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Thundermane', 'World of Warcraft: Forever', 0, NULL, 'https://twitch.tv/thundermane', false)`)

	// when I ask who is live
	resp := srv.Get(t, "/api/streams/live")

	// then I expect a 200 with an empty JSON array rather than null
	resp.RequireStatus(t, 200)
	if got := string(resp.Body); got != "[]\n" {
		t.Errorf("expected an empty JSON array, got %q", got)
	}
}

func TestListStreams_ReturnsLiveChannelsBeforeOfflineOnes(t *testing.T) {
	// given one offline streamer and one live streamer, inserted out of alphabetical order
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO streams (id, streamer_name, game_name, viewer_count, thumbnail_url, channel_url, is_live)
		VALUES
			('11111111-1111-1111-1111-111111111111', 'Aelith', '', 0, NULL, 'https://twitch.tv/aelith', false),
			('22222222-2222-2222-2222-222222222222', 'Thundermane', 'World of Warcraft: Forever', 1240, NULL, 'https://twitch.tv/thundermane', true)`)

	// when I ask for every stream channel
	resp := srv.Get(t, "/api/streams")

	// then I expect the live channel first, then the offline one
	resp.RequireStatus(t, 200)
	var all []streamJSON
	resp.DecodeJSON(t, &all)

	want := []streamJSON{
		{
			ID:           "22222222-2222-2222-2222-222222222222",
			StreamerName: "Thundermane",
			GameName:     "World of Warcraft: Forever",
			ViewerCount:  1240,
			ThumbnailURL: nil,
			ChannelURL:   "https://twitch.tv/thundermane",
			IsLive:       true,
		},
		{
			ID:           "11111111-1111-1111-1111-111111111111",
			StreamerName: "Aelith",
			GameName:     "",
			ViewerCount:  0,
			ThumbnailURL: nil,
			ChannelURL:   "https://twitch.tv/aelith",
			IsLive:       false,
		},
	}
	if diff := cmp.Diff(want, all); diff != "" {
		t.Errorf("unexpected streams (-want +got):\n%s", diff)
	}
}

func TestListStreams_ReturnsEmptyArray_WhenNoChannelsExist(t *testing.T) {
	// given a guild with no stream channels registered
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I ask for every stream channel
	resp := srv.Get(t, "/api/streams")

	// then I expect a 200 with an empty JSON array rather than null
	resp.RequireStatus(t, 200)
	if got := string(resp.Body); got != "[]\n" {
		t.Errorf("expected an empty JSON array, got %q", got)
	}
}
