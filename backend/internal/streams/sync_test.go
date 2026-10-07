package streams_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jmoiron/sqlx"

	"github.com/cageymage/fuzion/backend/internal/streams"
	"github.com/cageymage/fuzion/backend/internal/synclog"
	"github.com/cageymage/fuzion/backend/internal/testutil"
	"github.com/cageymage/fuzion/backend/internal/twitch"
)

type streamRow struct {
	Login           string  `db:"twitch_login"`
	IsLive          bool    `db:"is_live"`
	ViewerCount     int     `db:"viewer_count"`
	GameName        string  `db:"game_name"`
	Title           string  `db:"stream_title"`
	ThumbnailURL    *string `db:"thumbnail_url"`
	ProfileImageURL *string `db:"profile_image_url"`
}

type syncLogRow struct {
	Source  string `db:"source"`
	Status  string `db:"status"`
	Message string `db:"message"`
}

func streamRows(t *testing.T, db *sqlx.DB) []streamRow {
	t.Helper()
	var rows []streamRow
	if err := db.Select(&rows, `SELECT twitch_login, is_live, viewer_count, game_name, stream_title, thumbnail_url, profile_image_url FROM streams ORDER BY twitch_login`); err != nil {
		t.Fatalf("select streams: %v", err)
	}
	return rows
}

func syncLogRows(t *testing.T, db *sqlx.DB) []syncLogRow {
	t.Helper()
	var rows []syncLogRow
	if err := db.Select(&rows, `SELECT source, status, message FROM sync_log ORDER BY id`); err != nil {
		t.Fatalf("select sync_log: %v", err)
	}
	return rows
}

func fakeTwitch(t *testing.T, streamsStatus int, live, users []map[string]any) twitch.Config {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"access_token": "app-token"})
	})
	mux.HandleFunc("GET /helix/streams", func(w http.ResponseWriter, r *http.Request) {
		if streamsStatus != http.StatusOK {
			http.Error(w, "twitch is down", streamsStatus)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"data": live})
	})
	mux.HandleFunc("GET /helix/users", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"data": users})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return twitch.Config{ClientID: "id", ClientSecret: "secret", BaseURL: srv.URL, AuthURL: srv.URL}
}

func newSyncTwitch(db *sqlx.DB, cfg twitch.Config) *streams.SyncTwitch {
	return streams.NewSyncTwitch(streams.NewRepo(db), twitch.NewClient(cfg, nil), synclog.NewRepo(db))
}

func TestUpdateLiveStatus_MarksReturnedChannelsLiveAndOthersOffline(t *testing.T) {
	// given one channel that was live and one that was offline
	db := testutil.DB(t)
	db.MustExec(`
		INSERT INTO streams (id, streamer_name, game_name, stream_title, viewer_count, thumbnail_url, channel_url, is_live, twitch_login)
		VALUES
			('11111111-1111-1111-1111-111111111111', 'Emberfist', 'World of Warcraft', 'Old title', 310, 'https://cdn.example/old.jpg', 'https://twitch.tv/emberfist', true, 'emberfist'),
			('22222222-2222-2222-2222-222222222222', 'Thundermane', '', '', 0, NULL, 'https://twitch.tv/thundermane', false, 'thundermane')`)

	// when I apply a Twitch result in which only Thundermane is live
	err := streams.NewRepo(db).UpdateLiveStatus(context.Background(), map[string]twitch.LiveStream{
		"thundermane": {Login: "thundermane", GameName: "World of Warcraft", Title: "New title", ViewerCount: 1240, ThumbnailURL: "https://cdn.example/new.jpg"},
	})

	// then I expect Thundermane live with fresh details and Emberfist offline with no viewers or title
	if err != nil {
		t.Fatalf("UpdateLiveStatus: %v", err)
	}
	newThumbnail := "https://cdn.example/new.jpg"
	oldThumbnail := "https://cdn.example/old.jpg"
	want := []streamRow{
		{Login: "emberfist", IsLive: false, ViewerCount: 0, GameName: "World of Warcraft", Title: "", ThumbnailURL: &oldThumbnail},
		{Login: "thundermane", IsLive: true, ViewerCount: 1240, GameName: "World of Warcraft", Title: "New title", ThumbnailURL: &newThumbnail},
	}
	if diff := cmp.Diff(want, streamRows(t, db)); diff != "" {
		t.Errorf("unexpected streams (-want +got):\n%s", diff)
	}
}

func TestUpdateLiveStatus_ClearsTitle_WhenChannelGoesOffline(t *testing.T) {
	// given a live channel with a title
	db := testutil.DB(t)
	db.MustExec(`
		INSERT INTO streams (id, streamer_name, game_name, stream_title, viewer_count, channel_url, is_live, twitch_login)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Emberfist', 'World of Warcraft', 'Mythic progress', 310, 'https://twitch.tv/emberfist', true, 'emberfist')`)

	// when I apply a Twitch result with nobody live
	err := streams.NewRepo(db).UpdateLiveStatus(context.Background(), map[string]twitch.LiveStream{})

	// then I expect the title to be empty
	if err != nil {
		t.Fatalf("UpdateLiveStatus: %v", err)
	}
	var title string
	if err := db.Get(&title, `SELECT stream_title FROM streams`); err != nil {
		t.Fatalf("select stream_title: %v", err)
	}
	if title != "" {
		t.Errorf("stream_title = %q, want it cleared", title)
	}
}

func TestUpdateLiveStatus_LeavesChannelsWithoutTwitchLoginUntouched(t *testing.T) {
	// given a live channel that has no twitch login recorded
	db := testutil.DB(t)
	db.MustExec(`
		INSERT INTO streams (id, streamer_name, game_name, viewer_count, thumbnail_url, channel_url, is_live, twitch_login)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Manual', 'World of Warcraft', 50, NULL, 'https://example.com/manual', true, NULL)`)

	// when I apply a Twitch result with nobody live
	err := streams.NewRepo(db).UpdateLiveStatus(context.Background(), map[string]twitch.LiveStream{})

	// then I expect the manually managed channel to stay live
	if err != nil {
		t.Fatalf("UpdateLiveStatus: %v", err)
	}
	var isLive bool
	if err := db.Get(&isLive, `SELECT is_live FROM streams`); err != nil {
		t.Fatalf("select is_live: %v", err)
	}
	if !isLive {
		t.Errorf("expected the channel without a twitch login to stay live")
	}
}

func TestSyncTwitch_FlipsIsLiveAndWritesOkSyncLog_WhenTwitchReturnsLiveChannel(t *testing.T) {
	// given an offline channel and a Twitch that reports it live
	db := testutil.DB(t)
	db.MustExec(`
		INSERT INTO streams (id, streamer_name, game_name, viewer_count, thumbnail_url, channel_url, is_live, twitch_login)
		VALUES
			('11111111-1111-1111-1111-111111111111', 'Thundermane', '', 0, NULL, 'https://twitch.tv/thundermane', false, 'thundermane'),
			('22222222-2222-2222-2222-222222222222', 'Moonveil', '', 0, NULL, 'https://twitch.tv/moonveil', false, 'moonveil')`)
	cfg := fakeTwitch(t, http.StatusOK, []map[string]any{{
		"user_login":    "Thundermane",
		"game_name":     "World of Warcraft",
		"title":         "Mythic Queen Ansurek progress",
		"viewer_count":  1240,
		"thumbnail_url": "https://cdn.example/thumb-{width}x{height}.jpg",
	}}, []map[string]any{
		{"login": "thundermane", "profile_image_url": "https://cdn.example/thundermane-avatar.png"},
		{"login": "moonveil", "profile_image_url": "https://cdn.example/moonveil-avatar.png"},
	})

	// when the Twitch sync runs
	err := newSyncTwitch(db, cfg).Run(context.Background())

	// then I expect Thundermane live, Moonveil offline, both with avatars, and an ok sync_log row
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	thumbnail := "https://cdn.example/thumb-440x248.jpg"
	thundermaneAvatar := "https://cdn.example/thundermane-avatar.png"
	moonveilAvatar := "https://cdn.example/moonveil-avatar.png"
	wantStreams := []streamRow{
		{Login: "moonveil", IsLive: false, ViewerCount: 0, GameName: "", ThumbnailURL: nil, ProfileImageURL: &moonveilAvatar},
		{Login: "thundermane", IsLive: true, ViewerCount: 1240, GameName: "World of Warcraft", Title: "Mythic Queen Ansurek progress", ThumbnailURL: &thumbnail, ProfileImageURL: &thundermaneAvatar},
	}
	if diff := cmp.Diff(wantStreams, streamRows(t, db)); diff != "" {
		t.Errorf("unexpected streams (-want +got):\n%s", diff)
	}
	wantLog := []syncLogRow{{Source: "twitch", Status: "ok", Message: "1 of 2 channels live"}}
	if diff := cmp.Diff(wantLog, syncLogRows(t, db)); diff != "" {
		t.Errorf("unexpected sync_log (-want +got):\n%s", diff)
	}
}

func TestSyncTwitch_WritesErrorSyncLog_WhenTwitchIsDown(t *testing.T) {
	// given a live channel and a Twitch that answers 500
	db := testutil.DB(t)
	db.MustExec(`
		INSERT INTO streams (id, streamer_name, game_name, stream_title, viewer_count, thumbnail_url, channel_url, is_live, twitch_login)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Thundermane', 'World of Warcraft', 'Still here', 1240, NULL, 'https://twitch.tv/thundermane', true, 'thundermane')`)
	cfg := fakeTwitch(t, http.StatusInternalServerError, nil, nil)

	// when the Twitch sync runs
	err := newSyncTwitch(db, cfg).Run(context.Background())

	// then I expect an error, an error sync_log row, and the streams table untouched
	if err == nil {
		t.Fatal("expected Run to return an error when Twitch is down")
	}
	rows := syncLogRows(t, db)
	if len(rows) != 1 || rows[0].Source != "twitch" || rows[0].Status != "error" || rows[0].Message == "" {
		t.Errorf("expected one twitch error sync_log row with a message, got %+v", rows)
	}
	wantStreams := []streamRow{{Login: "thundermane", IsLive: true, ViewerCount: 1240, GameName: "World of Warcraft", Title: "Still here", ThumbnailURL: nil}}
	if diff := cmp.Diff(wantStreams, streamRows(t, db)); diff != "" {
		t.Errorf("expected streams to be untouched (-want +got):\n%s", diff)
	}
}

func TestUpdateProfileImages_StoresAvatarsForKnownLoginsOnly(t *testing.T) {
	// given two channels, one of which already has an avatar
	db := testutil.DB(t)
	db.MustExec(`
		INSERT INTO streams (id, streamer_name, game_name, viewer_count, thumbnail_url, channel_url, is_live, twitch_login, profile_image_url)
		VALUES
			('11111111-1111-1111-1111-111111111111', 'Emberfist', '', 0, NULL, 'https://twitch.tv/emberfist', false, 'emberfist', 'https://cdn.example/emberfist-old.png'),
			('22222222-2222-2222-2222-222222222222', 'Thundermane', '', 0, NULL, 'https://twitch.tv/thundermane', false, 'thundermane', NULL)`)

	// when I store an avatar for Thundermane only
	err := streams.NewRepo(db).UpdateProfileImages(context.Background(), map[string]string{
		"thundermane": "https://cdn.example/thundermane.png",
	})

	// then I expect Thundermane's avatar stored and Emberfist's kept
	if err != nil {
		t.Fatalf("UpdateProfileImages: %v", err)
	}
	emberfistAvatar := "https://cdn.example/emberfist-old.png"
	thundermaneAvatar := "https://cdn.example/thundermane.png"
	want := []streamRow{
		{Login: "emberfist", ProfileImageURL: &emberfistAvatar},
		{Login: "thundermane", ProfileImageURL: &thundermaneAvatar},
	}
	if diff := cmp.Diff(want, streamRows(t, db)); diff != "" {
		t.Errorf("unexpected streams (-want +got):\n%s", diff)
	}
}
