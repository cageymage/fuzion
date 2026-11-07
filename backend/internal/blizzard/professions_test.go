package blizzard_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/cageymage/fuzion/backend/internal/blizzard"
)

func newFakeProfessions(t *testing.T, status int) (*httptest.Server, *string) {
	t.Helper()

	body, err := os.ReadFile("testdata/professions.json")
	if err != nil {
		t.Fatalf("read professions fixture: %v", err)
	}
	var gotPath string
	mux := http.NewServeMux()
	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"access_token":"app-token"}`))
	})
	mux.HandleFunc("GET /profile/wow/character/{slug}/professions", func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath() + "?" + r.URL.RawQuery
		if status != http.StatusOK {
			http.Error(w, "blizzard says no", status)
			return
		}
		w.Write(body)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &gotPath
}

func professionsClient(srv *httptest.Server) *blizzard.Client {
	return blizzard.NewClient(blizzard.Config{
		ClientID: "id", ClientSecret: "secret", BaseURL: srv.URL, AuthURL: srv.URL, Namespace: "profile-forever-us",
	}, nil)
}

func TestCharacterProfessions_ReturnsPrimariesAndSecondaries_WhenBlizzardResponds(t *testing.T) {
	// given a Blizzard that returns a character's professions
	srv, _ := newFakeProfessions(t, http.StatusOK)
	client := professionsClient(srv)

	// when I ask for the professions of terrorskull-gloomstep
	got, err := client.CharacterProfessions(context.Background(), "terrorskull-gloomstep")

	// then I expect each profession with its skill points, primaries first
	if err != nil {
		t.Fatalf("CharacterProfessions: %v", err)
	}
	want := []blizzard.CharacterProfession{
		{Name: "Leatherworking", SkillPoints: 55},
		{Name: "Skinning", SkillPoints: 98},
		{Name: "Cooking", SkillPoints: 24},
		{Name: "First Aid", SkillPoints: 71},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected professions (-want +got):\n%s", diff)
	}
}

func TestCharacterProfessions_RequestsTheCharacterSlugInTheConfiguredNamespace(t *testing.T) {
	// given a Blizzard that records the request path
	srv, gotPath := newFakeProfessions(t, http.StatusOK)
	client := professionsClient(srv)

	// when I ask for the professions of a character whose slug has a non-ASCII letter
	if _, err := client.CharacterProfessions(context.Background(), "swamplörd-atrocitus"); err != nil {
		t.Fatalf("CharacterProfessions: %v", err)
	}

	// then I expect the slug escaped in the path and the namespace and locale in the query
	want := "/profile/wow/character/swampl%C3%B6rd-atrocitus/professions?locale=en_US&namespace=profile-forever-us"
	if *gotPath != want {
		t.Errorf("request = %q, want %q", *gotPath, want)
	}
}

func TestCharacterProfessions_ReturnsErrNotFound_WhenBlizzardRespondsWithNotFound(t *testing.T) {
	// given a Blizzard that does not know the character
	srv, _ := newFakeProfessions(t, http.StatusNotFound)
	client := professionsClient(srv)

	// when I ask for the character's professions
	_, err := client.CharacterProfessions(context.Background(), "nobody-here")

	// then I expect an error that is blizzard.ErrNotFound
	if !errors.Is(err, blizzard.ErrNotFound) {
		t.Errorf("err = %v, want blizzard.ErrNotFound", err)
	}
}

func TestCharacterProfessions_ReturnsPlainError_WhenBlizzardFails(t *testing.T) {
	// given a Blizzard that answers with a server error
	srv, _ := newFakeProfessions(t, http.StatusInternalServerError)
	client := professionsClient(srv)

	// when I ask for a character's professions
	_, err := client.CharacterProfessions(context.Background(), "terrorskull-gloomstep")

	// then I expect an error that is not ErrNotFound, so a caller does not treat it as a missing character
	if err == nil || errors.Is(err, blizzard.ErrNotFound) {
		t.Errorf("err = %v, want a non-ErrNotFound error", err)
	}
}
