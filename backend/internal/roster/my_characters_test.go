package roster_test

import (
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/cageymage/fuzion/backend/internal/testutil"
)

const (
	tankardID  = "11111111-1111-1111-1111-111111111111"
	healbotID  = "22222222-2222-2222-2222-222222222222"
	strangerID = "33333333-3333-3333-3333-333333333333"
	driftID    = "44444444-4444-4444-4444-444444444444"
	quitterID  = "55555555-5555-5555-5555-555555555555"
)

func insertOwnedCharacter(t *testing.T, db *sqlx.DB, id, name string, isMain bool, owner *uuid.UUID) {
	t.Helper()
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, role, is_main, user_id, created_at)
		VALUES ($1, $2, 'Unset', 'Emberreach', 'Warrior', 'tank', $3, $4, '2026-10-01T12:00:00Z')`,
		id, name, isMain, owner)
}

type mainRow struct {
	Name            string `db:"name"`
	IsMain          bool   `db:"is_main"`
	IsMainSetByHand bool   `db:"is_main_set_by_hand"`
}

func mainRows(t *testing.T, db *sqlx.DB) []mainRow {
	t.Helper()
	var rows []mainRow
	if err := db.Select(&rows, `SELECT name, is_main, is_main_set_by_hand FROM characters ORDER BY name`); err != nil {
		t.Fatalf("select characters: %v", err)
	}
	return rows
}

func TestListMyCharacters_ReturnsOnlyCharactersTheCallerOwns(t *testing.T) {
	// given my main and alt, another member's character, an unowned one, and one of mine that left the guild
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	other := srv.LoginAs(t, "11111111111111111", "someoneelse")
	me := srv.LoginAs(t, "80351110224678912", "thundermane")
	insertOwnedCharacter(t, db, tankardID, "Tankard", true, &me.ID)
	insertOwnedCharacter(t, db, healbotID, "Healbot", false, &me.ID)
	insertOwnedCharacter(t, db, strangerID, "Stranger", true, &other.ID)
	insertOwnedCharacter(t, db, driftID, "Drifter", true, nil)
	insertOwnedCharacter(t, db, quitterID, "Quitter", false, &me.ID)
	db.MustExec(`UPDATE characters SET left_guild_at = now() WHERE id = $1`, quitterID)

	// when I list my characters
	resp := srv.Get(t, "/api/me/characters")

	// then I expect only my two current characters, main first
	resp.RequireStatus(t, http.StatusOK)
	var got []characterJSON
	resp.DecodeJSON(t, &got)
	want := []characterJSON{
		{ID: tankardID, Name: "Tankard", SecondaryName: "Unset", Realm: "Emberreach", Class: "Warrior", Role: "tank", IsMain: true, CreatedAt: "2026-10-01T12:00:00Z"},
		{ID: healbotID, Name: "Healbot", SecondaryName: "Unset", Realm: "Emberreach", Class: "Warrior", Role: "tank", IsMain: false, CreatedAt: "2026-10-01T12:00:00Z"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected characters (-want +got):\n%s", diff)
	}
}

func TestListMyCharacters_ReturnsEmptyList_WhenTheCallerOwnsNoCharacters(t *testing.T) {
	// given a logged-in member who owns nothing, and an unowned character
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "80351110224678912", "thundermane")
	insertOwnedCharacter(t, db, driftID, "Drifter", true, nil)

	// when I list my characters
	resp := srv.Get(t, "/api/me/characters")

	// then I expect a 200 with an empty JSON array
	resp.RequireStatus(t, http.StatusOK)
	if got := string(resp.Body); got != "[]\n" {
		t.Errorf("expected an empty array, got %q", got)
	}
}

func TestListMyCharacters_ReturnsUnauthorized_WhenNotLoggedIn(t *testing.T) {
	// given nobody is logged in
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I list my characters
	resp := srv.Get(t, "/api/me/characters")

	// then I expect a 401
	resp.RequireStatus(t, http.StatusUnauthorized)
}

func TestUpdateMyCharacter_ClearsPreviousMain_WhenNewMainIsSet(t *testing.T) {
	// given my main and alt, and another member's main
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	other := srv.LoginAs(t, "11111111111111111", "someoneelse")
	me := srv.LoginAs(t, "80351110224678912", "thundermane")
	insertOwnedCharacter(t, db, tankardID, "Tankard", true, &me.ID)
	insertOwnedCharacter(t, db, healbotID, "Healbot", false, &me.ID)
	insertOwnedCharacter(t, db, strangerID, "Stranger", true, &other.ID)

	// when I make my alt my main
	resp := srv.Patch(t, "/api/me/characters/"+healbotID, map[string]any{"isMain": true})

	// then I expect it returned as a main, my old main now an alt, both set by hand, and the other member's main untouched
	resp.RequireStatus(t, http.StatusOK)
	var got characterJSON
	resp.DecodeJSON(t, &got)
	want := characterJSON{ID: healbotID, Name: "Healbot", SecondaryName: "Unset", Realm: "Emberreach", Class: "Warrior", Role: "tank", IsMain: true, CreatedAt: "2026-10-01T12:00:00Z"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected response (-want +got):\n%s", diff)
	}
	wantRows := []mainRow{
		{Name: "Healbot", IsMain: true, IsMainSetByHand: true},
		{Name: "Stranger", IsMain: true, IsMainSetByHand: false},
		{Name: "Tankard", IsMain: false, IsMainSetByHand: true},
	}
	if diff := cmp.Diff(wantRows, mainRows(t, db)); diff != "" {
		t.Errorf("unexpected mains (-want +got):\n%s", diff)
	}
}

func TestUpdateMyCharacter_MakesCharacterAnAlt_WhenIsMainIsFalse(t *testing.T) {
	// given my only character, a main
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	me := srv.LoginAs(t, "80351110224678912", "thundermane")
	insertOwnedCharacter(t, db, tankardID, "Tankard", true, &me.ID)

	// when I mark it as an alt
	resp := srv.Patch(t, "/api/me/characters/"+tankardID, map[string]any{"isMain": false})

	// then I expect it stored as an alt set by hand
	resp.RequireStatus(t, http.StatusOK)
	want := []mainRow{{Name: "Tankard", IsMain: false, IsMainSetByHand: true}}
	if diff := cmp.Diff(want, mainRows(t, db)); diff != "" {
		t.Errorf("unexpected mains (-want +got):\n%s", diff)
	}
}

func TestUpdateMyCharacter_LeavesRaidTeamUnchanged_WhenBodyIncludesIt(t *testing.T) {
	// given my character on an officer-assigned raid team
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	me := srv.LoginAs(t, "80351110224678912", "thundermane")
	insertOwnedCharacter(t, db, tankardID, "Tankard", false, &me.ID)
	db.MustExec(`UPDATE characters SET raid_team = 'Team 1' WHERE id = $1`, tankardID)

	// when I send a raid team along with main/alt
	resp := srv.Patch(t, "/api/me/characters/"+tankardID, map[string]any{"isMain": true, "raidTeam": "Team 2"})

	// then I expect main/alt applied and the raid team still the officer's
	resp.RequireStatus(t, http.StatusOK)
	var got characterJSON
	resp.DecodeJSON(t, &got)
	team1 := "Team 1"
	want := characterJSON{ID: tankardID, Name: "Tankard", SecondaryName: "Unset", Realm: "Emberreach", Class: "Warrior", Role: "tank", IsMain: true, RaidTeam: &team1, CreatedAt: "2026-10-01T12:00:00Z"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected response (-want +got):\n%s", diff)
	}
}

func TestUpdateMyCharacter_ReturnsForbidden_WhenCharacterBelongsToSomeoneElse(t *testing.T) {
	// given another member's alt
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	other := srv.LoginAs(t, "11111111111111111", "someoneelse")
	srv.LoginAs(t, "80351110224678912", "thundermane")
	insertOwnedCharacter(t, db, strangerID, "Stranger", false, &other.ID)

	// when I try to make it a main
	resp := srv.Patch(t, "/api/me/characters/"+strangerID, map[string]any{"isMain": true})

	// then I expect a 403 and the character unchanged
	resp.RequireStatus(t, http.StatusForbidden)
	requireErrorBody(t, resp, "character belongs to someone else")
	want := []mainRow{{Name: "Stranger", IsMain: false, IsMainSetByHand: false}}
	if diff := cmp.Diff(want, mainRows(t, db)); diff != "" {
		t.Errorf("unexpected mains (-want +got):\n%s", diff)
	}
}

func TestUpdateMyCharacter_ReturnsForbidden_WhenCharacterHasNoOwner(t *testing.T) {
	// given an unowned alt
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "80351110224678912", "thundermane")
	insertOwnedCharacter(t, db, driftID, "Drifter", false, nil)

	// when I try to make it a main
	resp := srv.Patch(t, "/api/me/characters/"+driftID, map[string]any{"isMain": true})

	// then I expect a 403 and the character unchanged
	resp.RequireStatus(t, http.StatusForbidden)
	want := []mainRow{{Name: "Drifter", IsMain: false, IsMainSetByHand: false}}
	if diff := cmp.Diff(want, mainRows(t, db)); diff != "" {
		t.Errorf("unexpected mains (-want +got):\n%s", diff)
	}
}

func TestUpdateMyCharacter_ReturnsNotFound_WhenCharacterDoesNotExist(t *testing.T) {
	// given a logged-in member and no characters
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "80351110224678912", "thundermane")

	// when I update a character id that does not exist
	resp := srv.Patch(t, "/api/me/characters/"+tankardID, map[string]any{"isMain": true})

	// then I expect a 404
	resp.RequireStatus(t, http.StatusNotFound)
	requireErrorBody(t, resp, "character not found")
}

func TestUpdateMyCharacter_ReturnsBadRequest_WhenIsMainIsMissing(t *testing.T) {
	// given my alt
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	me := srv.LoginAs(t, "80351110224678912", "thundermane")
	insertOwnedCharacter(t, db, tankardID, "Tankard", false, &me.ID)

	// when I send a body without isMain
	resp := srv.Patch(t, "/api/me/characters/"+tankardID, map[string]any{})

	// then I expect a 400 naming the field
	resp.RequireStatus(t, http.StatusBadRequest)
	requireErrorBody(t, resp, "isMain: is required")
}

func TestUpdateMyCharacter_ReturnsUnauthorized_WhenNotLoggedIn(t *testing.T) {
	// given nobody is logged in and an unowned character
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertOwnedCharacter(t, db, driftID, "Drifter", false, nil)

	// when I try to make it a main
	resp := srv.Patch(t, "/api/me/characters/"+driftID, map[string]any{"isMain": true})

	// then I expect a 401
	resp.RequireStatus(t, http.StatusUnauthorized)
}
