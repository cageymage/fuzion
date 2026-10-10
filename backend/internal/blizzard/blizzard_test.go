package blizzard_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/cageymage/fuzion/backend/internal/blizzard"
)

type fakeBlizzard struct {
	*httptest.Server

	mu            sync.Mutex
	tokenRequests int
	tokenAuth     string
	tokenGrant    string
	rosterPath    string
	rosterQuery   string
	rosterAuth    string
	rosterStatus  int
}

func newFakeBlizzard(t *testing.T) *fakeBlizzard {
	t.Helper()

	roster, err := os.ReadFile("testdata/roster.json")
	if err != nil {
		t.Fatalf("read roster fixture: %v", err)
	}

	f := &fakeBlizzard{rosterStatus: http.StatusOK}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.tokenRequests++
		f.tokenAuth = r.Header.Get("Authorization")
		r.ParseForm()
		f.tokenGrant = r.PostForm.Get("grant_type")
		json.NewEncoder(w).Encode(map[string]any{"access_token": "app-token", "token_type": "bearer", "expires_in": 86399})
	})
	mux.HandleFunc("GET /data/wow/guild/{realm}/{guild}/roster", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.rosterPath = r.URL.EscapedPath()
		f.rosterQuery = r.URL.RawQuery
		f.rosterAuth = r.Header.Get("Authorization")
		if f.rosterStatus != http.StatusOK {
			http.Error(w, "blizzard says no", f.rosterStatus)
			return
		}
		w.Write(roster)
	})
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

func (f *fakeBlizzard) config() blizzard.Config {
	return blizzard.Config{
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		BaseURL:      f.URL,
		AuthURL:      f.URL,
		Namespace:    "profile-classic1x-us",
	}
}

func TestGuildRoster_ParsesMembers_WhenBlizzardResponds(t *testing.T) {
	// given a Blizzard that returns a Classic-shaped guild roster
	fake := newFakeBlizzard(t)
	client := blizzard.NewClient(fake.config(), nil)

	// when I ask for the guild roster
	members, err := client.GuildRoster(context.Background(), "defias-pillager", "test-guild")

	// then I expect each member's name, realm, level, rank and class
	if err != nil {
		t.Fatalf("GuildRoster: %v", err)
	}
	want := []blizzard.RosterMember{
		{Name: "Testwarr", RealmSlug: "defias-pillager", Level: 60, Rank: 0, ClassID: 1, Class: "Warrior", RaceID: 1, Race: "Human"},
		{Name: "Testmage", RealmSlug: "defias-pillager", Level: 45, Rank: 5, ClassID: 8, Class: "Mage", RaceID: 7, Race: "Gnome"},
		{Name: "Testdruid", RealmSlug: "defias-pillager", Level: 60, Rank: 6, ClassID: 11, Class: "Druid", RaceID: 4, Race: "Night Elf"},
		{Name: "Thráin", RealmSlug: "area-52", Level: 58, Rank: 3, ClassID: 2, Class: "Paladin", RaceID: 3, Race: "Dwarf"},
		{Name: "Testdk", RealmSlug: "area-52", Level: 90, Rank: 5, ClassID: 6, Class: "", RaceID: 1, Race: "Human"},
	}
	if diff := cmp.Diff(want, members); diff != "" {
		t.Errorf("unexpected members (-want +got):\n%s", diff)
	}
}

func TestGuildRoster_SendsBasicAuthTokenRequestAndBearerRosterRequest(t *testing.T) {
	// given a Blizzard that records how it is called
	fake := newFakeBlizzard(t)
	client := blizzard.NewClient(fake.config(), nil)

	// when I ask for the guild roster
	if _, err := client.GuildRoster(context.Background(), "defias-pillager", "test-guild"); err != nil {
		t.Fatalf("GuildRoster: %v", err)
	}

	// then I expect client credentials with basic auth, then the roster with the bearer token
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if fake.tokenGrant != "client_credentials" {
		t.Errorf("grant_type = %q, want client_credentials", fake.tokenGrant)
	}
	if !strings.HasPrefix(fake.tokenAuth, "Basic ") {
		t.Errorf("token Authorization = %q, want a Basic header", fake.tokenAuth)
	}
	if fake.rosterAuth != "Bearer app-token" {
		t.Errorf("roster Authorization = %q, want Bearer app-token", fake.rosterAuth)
	}
	if fake.rosterPath != "/data/wow/guild/defias-pillager/test-guild/roster" {
		t.Errorf("roster path = %q", fake.rosterPath)
	}
	if fake.rosterQuery != "locale=en_US&namespace=profile-classic1x-us" {
		t.Errorf("roster query = %q", fake.rosterQuery)
	}
}

func TestGuildRoster_EscapesSlugs_WhenTheyContainReservedCharacters(t *testing.T) {
	// given a guild slug with a space and a slash in it
	fake := newFakeBlizzard(t)
	client := blizzard.NewClient(fake.config(), nil)

	// when I ask for that guild's roster
	if _, err := client.GuildRoster(context.Background(), "area-52", "my guild/x"); err != nil {
		t.Fatalf("GuildRoster: %v", err)
	}

	// then I expect the request path to keep the slug in one escaped segment
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if fake.rosterPath != "/data/wow/guild/area-52/my%20guild%2Fx/roster" {
		t.Errorf("roster path = %q", fake.rosterPath)
	}
}

func TestGuildRoster_FetchesOneToken_WhenCalledTwice(t *testing.T) {
	// given a client that already fetched a roster
	fake := newFakeBlizzard(t)
	client := blizzard.NewClient(fake.config(), nil)
	if _, err := client.GuildRoster(context.Background(), "defias-pillager", "test-guild"); err != nil {
		t.Fatalf("first GuildRoster: %v", err)
	}

	// when I ask again
	if _, err := client.GuildRoster(context.Background(), "defias-pillager", "test-guild"); err != nil {
		t.Fatalf("second GuildRoster: %v", err)
	}

	// then I expect the token to have been requested once
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if fake.tokenRequests != 1 {
		t.Errorf("token requests = %d, want 1", fake.tokenRequests)
	}
}

func TestGuildRoster_ReturnsError_WhenBlizzardRespondsWithNotFound(t *testing.T) {
	// given a Blizzard that does not know the guild
	fake := newFakeBlizzard(t)
	fake.rosterStatus = http.StatusNotFound
	client := blizzard.NewClient(fake.config(), nil)

	// when I ask for the guild roster
	_, err := client.GuildRoster(context.Background(), "defias-pillager", "test-guild")

	// then I expect an error naming the status
	if err == nil || !strings.Contains(err.Error(), "404") {
		t.Errorf("err = %v, want an error mentioning 404", err)
	}
}

func TestGuildRoster_ReturnsError_WhenTokenRequestFails(t *testing.T) {
	// given a token endpoint that rejects the credentials
	mux := http.NewServeMux()
	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"invalid_client"}`, http.StatusUnauthorized)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	client := blizzard.NewClient(blizzard.Config{ClientID: "x", ClientSecret: "y", BaseURL: srv.URL, AuthURL: srv.URL}, nil)

	// when I ask for the guild roster
	_, err := client.GuildRoster(context.Background(), "defias-pillager", "test-guild")

	// then I expect the token failure to be reported
	if err == nil || !strings.Contains(err.Error(), "app token") {
		t.Errorf("err = %v, want an app token error", err)
	}
}
