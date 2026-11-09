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
	Name   string `db:"name"`
	IsMain bool   `db:"is_main"`
}

func mainRows(t *testing.T, db *sqlx.DB) []mainRow {
	t.Helper()
	var rows []mainRow
	if err := db.Select(&rows, `SELECT name, is_main FROM characters ORDER BY name`); err != nil {
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

func TestUpdateMyCharacter_ReturnsNotFound_WhenAMemberTriesToChangeMainOrAlt(t *testing.T) {
	// given my alt; main or alt now follows the guild rank in game
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	me := srv.LoginAs(t, "80351110224678912", "thundermane")
	insertOwnedCharacter(t, db, healbotID, "Healbot", false, &me.ID)

	// when I try to make it my main
	resp := srv.Patch(t, "/api/me/characters/"+healbotID, map[string]any{"isMain": true})

	// then I expect a 404, since members can no longer change it, and the character still an alt
	resp.RequireStatus(t, http.StatusNotFound)
	want := []mainRow{{Name: "Healbot", IsMain: false}}
	if diff := cmp.Diff(want, mainRows(t, db)); diff != "" {
		t.Errorf("unexpected main/alt (-want +got):\n%s", diff)
	}
}
