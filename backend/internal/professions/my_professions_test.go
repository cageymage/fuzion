package professions_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jmoiron/sqlx"

	"github.com/cageymage/fuzion/backend/internal/testutil"
)

type storedProfession struct {
	ID          string `db:"id"`
	CharacterID string `db:"character_id"`
	Profession  string `db:"profession"`
	SkillLevel  int    `db:"skill_level"`
}

func storedProfessions(t *testing.T, db *sqlx.DB) []storedProfession {
	t.Helper()
	var rows []storedProfession
	if err := db.Select(&rows, `SELECT id, character_id, profession, skill_level FROM professions ORDER BY id`); err != nil {
		t.Fatalf("select professions: %v", err)
	}
	return rows
}

var seededProfessions = []storedProfession{
	{ID: "a1111111-1111-1111-1111-111111111111", CharacterID: zephyrion.ID, Profession: "Blacksmithing", SkillLevel: 300},
	{ID: "a2222222-2222-2222-2222-222222222222", CharacterID: aeliana.ID, Profession: "Blacksmithing", SkillLevel: 225},
	{ID: "a3333333-3333-3333-3333-333333333333", CharacterID: aeliana.ID, Profession: "Alchemy", SkillLevel: 300},
	{ID: "a4444444-4444-4444-4444-444444444444", CharacterID: zephyrion.ID, Profession: "Alchemy", SkillLevel: 300},
}

// seedOwnedCharacters logs in two members, gives me Aeliana and the other member
// Zephyrion, and leaves me logged in.
func seedOwnedCharacters(t *testing.T, db *sqlx.DB, srv *testutil.Server) {
	t.Helper()
	db.MustExec(seedCharacters)
	db.MustExec(seedProfessions)
	other := srv.LoginAs(t, "11111111111111111", "someoneelse")
	me := srv.LoginAs(t, "80351110224678912", "thundermane")
	db.MustExec(`UPDATE characters SET user_id = $1 WHERE id = $2`, me.ID, aeliana.ID)
	db.MustExec(`UPDATE characters SET user_id = $1 WHERE id = $2`, other.ID, zephyrion.ID)
}

func requireError(t *testing.T, resp testutil.Response, want string) {
	t.Helper()
	var body map[string]string
	resp.DecodeJSON(t, &body)
	if diff := cmp.Diff(map[string]string{"error": want}, body); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
}

func TestCreateMyProfession_ReturnsCreated_WhenMemberOwnsCharacter(t *testing.T) {
	// given I own Aeliana
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedOwnedCharacters(t, db, srv)

	// when I add Tailoring to Aeliana, with stray whitespace
	resp := srv.Post(t, "/api/me/characters/"+aeliana.ID+"/professions", map[string]any{"profession": " Tailoring ", "skillLevel": 150})

	// then I expect a 201 in the same shape as the professions list, with the name trimmed
	resp.RequireStatus(t, http.StatusCreated)
	var created professionJSON
	resp.DecodeJSON(t, &created)
	want := professionJSON{ID: created.ID, Profession: "Tailoring", SkillLevel: 150, Character: aeliana}
	if diff := cmp.Diff(want, created); diff != "" {
		t.Errorf("unexpected created profession (-want +got):\n%s", diff)
	}
	wantStored := append([]storedProfession{}, seededProfessions...)
	wantStored = append(wantStored, storedProfession{ID: created.ID, CharacterID: aeliana.ID, Profession: "Tailoring", SkillLevel: 150})
	slices.SortFunc(wantStored, func(a, b storedProfession) int { return strings.Compare(a.ID, b.ID) })
	if diff := cmp.Diff(wantStored, storedProfessions(t, db)); diff != "" {
		t.Errorf("unexpected professions (-want +got):\n%s", diff)
	}
}

func TestCreateMyProfession_ReturnsForbidden_WhenCharacterBelongsToSomeoneElse(t *testing.T) {
	// given another member owns Zephyrion
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedOwnedCharacters(t, db, srv)

	// when I add Tailoring to Zephyrion
	resp := srv.Post(t, "/api/me/characters/"+zephyrion.ID+"/professions", map[string]any{"profession": "Tailoring", "skillLevel": 150})

	// then I expect a 403 and nothing added
	resp.RequireStatus(t, http.StatusForbidden)
	requireError(t, resp, "character belongs to someone else")
	if diff := cmp.Diff(seededProfessions, storedProfessions(t, db)); diff != "" {
		t.Errorf("professions changed (-want +got):\n%s", diff)
	}
}

func TestCreateMyProfession_ReturnsForbidden_WhenCharacterHasNoOwner(t *testing.T) {
	// given nobody owns Aeliana
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedOwnedCharacters(t, db, srv)
	db.MustExec(`UPDATE characters SET user_id = NULL WHERE id = $1`, aeliana.ID)

	// when I add Tailoring to Aeliana
	resp := srv.Post(t, "/api/me/characters/"+aeliana.ID+"/professions", map[string]any{"profession": "Tailoring", "skillLevel": 150})

	// then I expect a 403 and nothing added
	resp.RequireStatus(t, http.StatusForbidden)
	if diff := cmp.Diff(seededProfessions, storedProfessions(t, db)); diff != "" {
		t.Errorf("professions changed (-want +got):\n%s", diff)
	}
}

func TestCreateMyProfession_ReturnsNotFound_WhenCharacterDoesNotExist(t *testing.T) {
	// given I own Aeliana
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedOwnedCharacters(t, db, srv)

	// when I add a profession to a character id that does not exist
	resp := srv.Post(t, "/api/me/characters/99999999-9999-9999-9999-999999999999/professions", map[string]any{"profession": "Tailoring", "skillLevel": 150})

	// then I expect a 404
	resp.RequireStatus(t, http.StatusNotFound)
	requireError(t, resp, "character not found")
}

func TestCreateMyProfession_ReturnsBadRequest_WhenSkillLevelIsNegative(t *testing.T) {
	// given I own Aeliana
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedOwnedCharacters(t, db, srv)

	// when I add a profession with a negative skill level
	resp := srv.Post(t, "/api/me/characters/"+aeliana.ID+"/professions", map[string]any{"profession": "Tailoring", "skillLevel": -1})

	// then I expect the same 400 officers get
	resp.RequireStatus(t, http.StatusBadRequest)
	requireError(t, resp, "skillLevel: must be 0 or more")
}

func TestCreateMyProfession_ReturnsConflict_WhenCharacterAlreadyHasThatProfession(t *testing.T) {
	// given I own Aeliana, who already knows Alchemy
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedOwnedCharacters(t, db, srv)

	// when I add Alchemy again
	resp := srv.Post(t, "/api/me/characters/"+aeliana.ID+"/professions", map[string]any{"profession": "Alchemy", "skillLevel": 1})

	// then I expect the same 409 officers get
	resp.RequireStatus(t, http.StatusConflict)
	requireError(t, resp, "character already has this profession")
}

func TestCreateMyProfession_ReturnsUnauthorized_WhenCallerIsAnonymous(t *testing.T) {
	// given nobody is logged in
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(seedCharacters)

	// when I add a profession to Aeliana
	resp := srv.Post(t, "/api/me/characters/"+aeliana.ID+"/professions", map[string]any{"profession": "Tailoring", "skillLevel": 150})

	// then I expect a 401
	resp.RequireStatus(t, http.StatusUnauthorized)
}

func TestUpdateMyProfession_ChangesSkillLevel_WhenMemberOwnsCharacter(t *testing.T) {
	// given I own Aeliana, who knows Alchemy at 300
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedOwnedCharacters(t, db, srv)

	// when I raise her Alchemy to 310
	resp := srv.Patch(t, "/api/me/professions/a3333333-3333-3333-3333-333333333333", map[string]any{"skillLevel": 310})

	// then I expect the updated row back
	resp.RequireStatus(t, http.StatusOK)
	var updated professionJSON
	resp.DecodeJSON(t, &updated)
	want := professionJSON{ID: "a3333333-3333-3333-3333-333333333333", Profession: "Alchemy", SkillLevel: 310, Character: aeliana}
	if diff := cmp.Diff(want, updated); diff != "" {
		t.Errorf("unexpected updated profession (-want +got):\n%s", diff)
	}
}

func TestUpdateMyProfession_ReturnsForbidden_WhenCharacterBelongsToSomeoneElse(t *testing.T) {
	// given another member owns Zephyrion, who knows Blacksmithing at 300
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedOwnedCharacters(t, db, srv)

	// when I change his Blacksmithing
	resp := srv.Patch(t, "/api/me/professions/a1111111-1111-1111-1111-111111111111", map[string]any{"skillLevel": 1})

	// then I expect a 403 and nothing changed
	resp.RequireStatus(t, http.StatusForbidden)
	if diff := cmp.Diff(seededProfessions, storedProfessions(t, db)); diff != "" {
		t.Errorf("professions changed (-want +got):\n%s", diff)
	}
}

func TestUpdateMyProfession_ReturnsNotFound_WhenProfessionDoesNotExist(t *testing.T) {
	// given I own Aeliana
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedOwnedCharacters(t, db, srv)

	// when I update a profession id that does not exist
	resp := srv.Patch(t, "/api/me/professions/99999999-9999-9999-9999-999999999999", map[string]any{"skillLevel": 1})

	// then I expect a 404
	resp.RequireStatus(t, http.StatusNotFound)
	requireError(t, resp, "profession not found")
}

func TestDeleteMyProfession_ReturnsNoContent_WhenMemberOwnsCharacter(t *testing.T) {
	// given I own Aeliana, who knows Alchemy
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedOwnedCharacters(t, db, srv)

	// when I remove her Alchemy
	resp := srv.Delete(t, "/api/me/professions/a3333333-3333-3333-3333-333333333333")

	// then I expect a 204 and only that row gone
	resp.RequireStatus(t, http.StatusNoContent)
	want := []storedProfession{seededProfessions[0], seededProfessions[1], seededProfessions[3]}
	if diff := cmp.Diff(want, storedProfessions(t, db)); diff != "" {
		t.Errorf("unexpected professions (-want +got):\n%s", diff)
	}
}

func TestDeleteMyProfession_ReturnsForbidden_WhenCharacterBelongsToSomeoneElse(t *testing.T) {
	// given another member owns Zephyrion, who knows Alchemy
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedOwnedCharacters(t, db, srv)

	// when I remove his Alchemy
	resp := srv.Delete(t, "/api/me/professions/a4444444-4444-4444-4444-444444444444")

	// then I expect a 403 and nothing removed
	resp.RequireStatus(t, http.StatusForbidden)
	if diff := cmp.Diff(seededProfessions, storedProfessions(t, db)); diff != "" {
		t.Errorf("professions changed (-want +got):\n%s", diff)
	}
}

func TestUpdateProfession_LetsOfficerEditAMembersProfession(t *testing.T) {
	// given a member owns Aeliana, who knows Alchemy at 300, and I am an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedOwnedCharacters(t, db, srv)
	srv.LoginAs(t, "33333333333333333", "officer", testutil.AsOfficer())

	// when I change her Alchemy through the officer endpoint
	resp := srv.Patch(t, "/api/professions/a3333333-3333-3333-3333-333333333333", map[string]any{"skillLevel": 250})

	// then I expect the change applied, since officers can edit any character
	resp.RequireStatus(t, http.StatusOK)
	var updated professionJSON
	resp.DecodeJSON(t, &updated)
	want := professionJSON{ID: "a3333333-3333-3333-3333-333333333333", Profession: "Alchemy", SkillLevel: 250, Character: aeliana}
	if diff := cmp.Diff(want, updated); diff != "" {
		t.Errorf("unexpected updated profession (-want +got):\n%s", diff)
	}
}
