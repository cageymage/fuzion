package twitch_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/cageymage/fuzion/backend/internal/twitch"
)

type fakeTwitch struct {
	*httptest.Server

	mu         sync.Mutex
	requests   []string
	streamsURL []string
	helixAuth  string
	helixID    string
	status     int
	live       []map[string]any
	users      []map[string]any
	usersURL   []string
}

func newFakeTwitch(t *testing.T) *fakeTwitch {
	t.Helper()

	f := &fakeTwitch{status: http.StatusOK}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.requests = append(f.requests, "token")
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if r.PostForm.Get("grant_type") != "client_credentials" ||
			r.PostForm.Get("client_id") != "test-id" ||
			r.PostForm.Get("client_secret") != "test-secret" {
			http.Error(w, "bad credentials", http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"access_token": "app-token", "expires_in": 5000000, "token_type": "bearer"})
	})
	mux.HandleFunc("GET /helix/streams", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.requests = append(f.requests, "streams")
		f.streamsURL = append(f.streamsURL, r.URL.RawQuery)
		f.helixAuth = r.Header.Get("Authorization")
		f.helixID = r.Header.Get("Client-Id")
		if f.status != http.StatusOK {
			http.Error(w, "twitch is down", f.status)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"data": f.live})
	})

	mux.HandleFunc("GET /helix/users", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.requests = append(f.requests, "users")
		f.usersURL = append(f.usersURL, r.URL.RawQuery)
		if f.status != http.StatusOK {
			http.Error(w, "twitch is down", f.status)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"data": f.users})
	})

	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

func (f *fakeTwitch) client() *twitch.Client {
	return twitch.NewClient(twitch.Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		BaseURL:      f.URL,
		AuthURL:      f.URL,
	}, f.Client())
}

func TestLiveStreams_ReturnsLiveChannels_WhenTwitchResponds(t *testing.T) {
	// given Twitch reports one of two requested channels as live
	fake := newFakeTwitch(t)
	fake.live = []map[string]any{{
		"user_login":    "Thundermane",
		"game_name":     "World of Warcraft",
		"title":         "Mythic Queen Ansurek progress",
		"viewer_count":  1240,
		"thumbnail_url": "https://static-cdn.jtvnw.net/previews-ttv/live_user_thundermane-{width}x{height}.jpg",
	}}

	// when I ask which channels are live
	got, err := fake.client().LiveStreams(context.Background(), []string{"thundermane", "moonveil"})

	// then I expect only the live channel, with a usable thumbnail
	if err != nil {
		t.Fatalf("LiveStreams: %v", err)
	}
	want := []twitch.LiveStream{{
		Login:        "thundermane",
		GameName:     "World of Warcraft",
		Title:        "Mythic Queen Ansurek progress",
		ViewerCount:  1240,
		ThumbnailURL: "https://static-cdn.jtvnw.net/previews-ttv/live_user_thundermane-440x248.jpg",
	}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected live streams (-want +got):\n%s", diff)
	}
}

func TestLiveStreams_ReplacesThumbnailSizePlaceholders_WhenTwitchReturnsTemplateUrl(t *testing.T) {
	// given Twitch returns a thumbnail URL with size placeholders
	fake := newFakeTwitch(t)
	fake.live = []map[string]any{{
		"user_login":    "Thundermane",
		"thumbnail_url": "https://static-cdn.jtvnw.net/previews-ttv/live_user_thundermane-{width}x{height}.jpg",
	}}

	// when I ask which channels are live
	got, err := fake.client().LiveStreams(context.Background(), []string{"thundermane"})

	// then I expect the thumbnail URL to hold a fixed size and no placeholders
	if err != nil {
		t.Fatalf("LiveStreams: %v", err)
	}
	want := "https://static-cdn.jtvnw.net/previews-ttv/live_user_thundermane-440x248.jpg"
	if len(got) != 1 || got[0].ThumbnailURL != want {
		t.Errorf("thumbnail URL = %+v, want %q", got, want)
	}
}

func TestLiveStreams_ReturnsTitle_WhenTwitchResponds(t *testing.T) {
	// given Twitch reports a live channel with a title
	fake := newFakeTwitch(t)
	fake.live = []map[string]any{{
		"user_login": "Thundermane",
		"title":      "Mythic Queen Ansurek progress",
	}}

	// when I ask which channels are live
	got, err := fake.client().LiveStreams(context.Background(), []string{"thundermane"})

	// then I expect the stream title to come back
	if err != nil {
		t.Fatalf("LiveStreams: %v", err)
	}
	if len(got) != 1 || got[0].Title != "Mythic Queen Ansurek progress" {
		t.Errorf("title = %+v, want %q", got, "Mythic Queen Ansurek progress")
	}
}

func TestLiveStreams_FetchesAppToken_BeforeFirstCall(t *testing.T) {
	// given a fake Twitch that has not yet handed out a token
	fake := newFakeTwitch(t)

	// when I ask which channels are live
	if _, err := fake.client().LiveStreams(context.Background(), []string{"thundermane"}); err != nil {
		t.Fatalf("LiveStreams: %v", err)
	}

	// then I expect the token request to come first and the streams call to carry it
	if diff := cmp.Diff([]string{"token", "streams"}, fake.requests); diff != "" {
		t.Errorf("unexpected request order (-want +got):\n%s", diff)
	}
	if fake.helixAuth != "Bearer app-token" {
		t.Errorf("expected the streams call to send the app token, got Authorization %q", fake.helixAuth)
	}
	if fake.helixID != "test-id" {
		t.Errorf("expected the streams call to send the client id, got Client-Id %q", fake.helixID)
	}
}

func TestLiveStreams_FiltersToWorldOfWarcraftCategory_WhenAskingTwitch(t *testing.T) {
	// given a fake Twitch
	fake := newFakeTwitch(t)

	// when I ask which channels are live
	if _, err := fake.client().LiveStreams(context.Background(), []string{"thundermane", "moonveil"}); err != nil {
		t.Fatalf("LiveStreams: %v", err)
	}

	// then I expect the query to name both logins and the WoW category
	if len(fake.streamsURL) != 1 {
		t.Fatalf("expected one streams call, got %d", len(fake.streamsURL))
	}
	query := fake.streamsURL[0]
	for _, part := range []string{"user_login=thundermane", "user_login=moonveil", "game_id=" + twitch.WorldOfWarcraftGameID, "first=100"} {
		if !strings.Contains(query, part) {
			t.Errorf("expected query %q to contain %q", query, part)
		}
	}
}

func TestLiveStreams_SplitsLoginsIntoBatchesOf100_WhenMoreThan100AreRequested(t *testing.T) {
	// given 150 channels to check
	fake := newFakeTwitch(t)
	logins := make([]string, 150)
	for i := range logins {
		logins[i] = "streamer" + string(rune('a'+i%26)) + strings.Repeat("x", i/26)
	}

	// when I ask which are live
	if _, err := fake.client().LiveStreams(context.Background(), logins); err != nil {
		t.Fatalf("LiveStreams: %v", err)
	}

	// then I expect two streams calls of 100 and 50 logins
	if len(fake.streamsURL) != 2 {
		t.Fatalf("expected two streams calls, got %d", len(fake.streamsURL))
	}
	if got := strings.Count(fake.streamsURL[0], "user_login="); got != 100 {
		t.Errorf("expected 100 logins in the first call, got %d", got)
	}
	if got := strings.Count(fake.streamsURL[1], "user_login="); got != 50 {
		t.Errorf("expected 50 logins in the second call, got %d", got)
	}
}

func TestLiveStreams_ReturnsError_WhenTwitchRespondsWithServerError(t *testing.T) {
	// given Twitch is down
	fake := newFakeTwitch(t)
	fake.status = http.StatusInternalServerError

	// when I ask which channels are live
	got, err := fake.client().LiveStreams(context.Background(), []string{"thundermane"})

	// then I expect an error and no streams
	if err == nil {
		t.Fatalf("expected an error, got streams %v", got)
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected the error to mention status 500, got %q", err)
	}
}

func TestLiveStreams_ReturnsError_WhenCredentialsAreRejected(t *testing.T) {
	// given a client with the wrong secret
	fake := newFakeTwitch(t)
	client := twitch.NewClient(twitch.Config{
		ClientID:     "test-id",
		ClientSecret: "wrong",
		BaseURL:      fake.URL,
		AuthURL:      fake.URL,
	}, fake.Client())

	// when I ask which channels are live
	_, err := client.LiveStreams(context.Background(), []string{"thundermane"})

	// then I expect an error and no streams call was made
	if err == nil {
		t.Fatal("expected an error when the token request is rejected")
	}
	if diff := cmp.Diff([]string{"token"}, fake.requests); diff != "" {
		t.Errorf("unexpected requests (-want +got):\n%s", diff)
	}
}

func TestAvatars_ReturnsProfileImagesByLogin_WhenTwitchResponds(t *testing.T) {
	// given Twitch knows two of the three requested users
	fake := newFakeTwitch(t)
	fake.users = []map[string]any{
		{"login": "thundermane", "profile_image_url": "https://static-cdn.jtvnw.net/jtv_user_pictures/thundermane.png"},
		{"login": "Moonveil", "profile_image_url": "https://static-cdn.jtvnw.net/jtv_user_pictures/moonveil.png"},
	}

	// when I ask for their avatars
	got, err := fake.client().Avatars(context.Background(), []string{"thundermane", "moonveil", "ghost"})

	// then I expect a profile image per known login, keyed by lowercase login
	if err != nil {
		t.Fatalf("Avatars: %v", err)
	}
	want := map[string]string{
		"thundermane": "https://static-cdn.jtvnw.net/jtv_user_pictures/thundermane.png",
		"moonveil":    "https://static-cdn.jtvnw.net/jtv_user_pictures/moonveil.png",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected avatars (-want +got):\n%s", diff)
	}
	for _, part := range []string{"login=thundermane", "login=moonveil", "login=ghost"} {
		if !strings.Contains(fake.usersURL[0], part) {
			t.Errorf("expected query %q to contain %q", fake.usersURL[0], part)
		}
	}
}

func TestAvatars_ReusesAppToken_WhenLiveStreamsWasCalledFirst(t *testing.T) {
	// given a client that has already asked which channels are live
	fake := newFakeTwitch(t)
	client := fake.client()
	if _, err := client.LiveStreams(context.Background(), []string{"thundermane"}); err != nil {
		t.Fatalf("LiveStreams: %v", err)
	}

	// when I ask for avatars
	if _, err := client.Avatars(context.Background(), []string{"thundermane"}); err != nil {
		t.Fatalf("Avatars: %v", err)
	}

	// then I expect only one token request in total
	if diff := cmp.Diff([]string{"token", "streams", "users"}, fake.requests); diff != "" {
		t.Errorf("unexpected request order (-want +got):\n%s", diff)
	}
}

func TestAvatars_ReturnsError_WhenTwitchRespondsWithServerError(t *testing.T) {
	// given Twitch is down
	fake := newFakeTwitch(t)
	fake.status = http.StatusInternalServerError

	// when I ask for avatars
	_, err := fake.client().Avatars(context.Background(), []string{"thundermane"})

	// then I expect an error
	if err == nil {
		t.Fatal("expected an error when Twitch is down")
	}
}
