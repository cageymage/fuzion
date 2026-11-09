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
		{ID: tankardID, Name: "Tankard", SecondaryName: "Unset", Realm: "Emberreach", Class: "Warrior", Role: strPtr("tank"), IsMain: true, CreatedAt: "2026-10-01T12:00:00Z"},
		{ID: healbotID, Name: "Healbot", SecondaryName: "Unset", Realm: "Emberreach", Class: "Warrior", Role: strPtr("tank"), IsMain: false, CreatedAt: "2026-10-01T12:00:00Z"},
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

func TestUpdateMyCharacter_SetsSpecAndRole_WhenOwner(t *testing.T) {
	// given my warrior with no spec chosen
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	me := srv.LoginAs(t, "80351110224678912", "thundermane")
	insertOwnedCharacter(t, db, tankardID, "Tankard", true, &me.ID)

	// when I set a primary and a secondary spec and role
	resp := srv.Patch(t, "/api/me/characters/"+tankardID, map[string]any{
		"spec": "Protection", "role": "tank", "spec2": "Fury", "role2": "dps",
	})

	// then I expect the character back with both specs and roles
	resp.RequireStatus(t, http.StatusOK)
	var got characterJSON
	resp.DecodeJSON(t, &got)
	want := characterJSON{
		ID: tankardID, Name: "Tankard", SecondaryName: "Unset", Realm: "Emberreach", Class: "Warrior",
		Spec: strPtr("Protection"), Role: strPtr("tank"), Spec2: strPtr("Fury"), Role2: strPtr("dps"),
		IsMain: true, CreatedAt: "2026-10-01T12:00:00Z",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected character (-want +got):\n%s", diff)
	}
}

func TestUpdateMyCharacter_ClearsTheRole_WhenTheRoleIsAnEmptyString(t *testing.T) {
	// given my warrior set to tank
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	me := srv.LoginAs(t, "80351110224678912", "thundermane")
	insertOwnedCharacter(t, db, tankardID, "Tankard", true, &me.ID)

	// when I set its role back to none
	resp := srv.Patch(t, "/api/me/characters/"+tankardID, map[string]any{"role": ""})

	// then I expect the character back with no role
	resp.RequireStatus(t, http.StatusOK)
	var got characterJSON
	resp.DecodeJSON(t, &got)
	want := characterJSON{
		ID: tankardID, Name: "Tankard", SecondaryName: "Unset", Realm: "Emberreach", Class: "Warrior",
		IsMain: true, CreatedAt: "2026-10-01T12:00:00Z",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected character (-want +got):\n%s", diff)
	}
}

func TestUpdateMyCharacter_ReturnsForbidden_WhenNotOwner(t *testing.T) {
	// given another member's tank
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	other := srv.LoginAs(t, "11111111111111111", "someoneelse")
	srv.LoginAs(t, "80351110224678912", "thundermane")
	insertOwnedCharacter(t, db, strangerID, "Stranger", true, &other.ID)

	// when I try to set its role
	resp := srv.Patch(t, "/api/me/characters/"+strangerID, map[string]any{"role": "healer"})

	// then I expect a 403 and the role unchanged
	resp.RequireStatus(t, http.StatusForbidden)
	if diff := cmp.Diff(`{"error":"character is not yours"}`+"\n", string(resp.Body)); diff != "" {
		t.Errorf("unexpected body (-want +got):\n%s", diff)
	}
	var role string
	if err := db.Get(&role, `SELECT role FROM characters WHERE id = $1`, strangerID); err != nil {
		t.Fatalf("select role: %v", err)
	}
	if role != "tank" {
		t.Errorf("expected role tank, got %q", role)
	}
}

func TestUpdateMyCharacter_ReturnsNotFound_WhenTheCharacterDoesNotExist(t *testing.T) {
	// given a logged-in member and no such character
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "80351110224678912", "thundermane")

	// when I try to set its role
	resp := srv.Patch(t, "/api/me/characters/"+tankardID, map[string]any{"role": "healer"})

	// then I expect a 404
	resp.RequireStatus(t, http.StatusNotFound)
	if diff := cmp.Diff(`{"error":"character not found"}`+"\n", string(resp.Body)); diff != "" {
		t.Errorf("unexpected body (-want +got):\n%s", diff)
	}
}

func TestUpdateMyCharacter_ReturnsBadRequest_WhenSecondSpecWithoutSecondRole(t *testing.T) {
	// given my warrior
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	me := srv.LoginAs(t, "80351110224678912", "thundermane")
	insertOwnedCharacter(t, db, tankardID, "Tankard", true, &me.ID)

	// when I set a second spec without a second role
	resp := srv.Patch(t, "/api/me/characters/"+tankardID, map[string]any{"spec2": "Fury"})

	// then I expect a 400 naming the missing field
	resp.RequireStatus(t, http.StatusBadRequest)
	if diff := cmp.Diff(`{"error":"role2: must be set together with spec2"}`+"\n", string(resp.Body)); diff != "" {
		t.Errorf("unexpected body (-want +got):\n%s", diff)
	}
}

func TestUpdateMyCharacter_ReturnsBadRequest_WhenTheSpecIsNotOneOfTheClasses(t *testing.T) {
	// given my warrior
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	me := srv.LoginAs(t, "80351110224678912", "thundermane")
	insertOwnedCharacter(t, db, tankardID, "Tankard", true, &me.ID)

	// when I set a priest spec
	resp := srv.Patch(t, "/api/me/characters/"+tankardID, map[string]any{"spec": "Shadow"})

	// then I expect a 400 listing the warrior specs
	resp.RequireStatus(t, http.StatusBadRequest)
	if diff := cmp.Diff(`{"error":"spec: must be one of Arms, Fury, Protection"}`+"\n", string(resp.Body)); diff != "" {
		t.Errorf("unexpected body (-want +got):\n%s", diff)
	}
}

func TestUpdateMyCharacter_IgnoresOfficerFields_WhenTheBodyIncludesThem(t *testing.T) {
	// given my alt; main or alt follows the guild rank, and raid teams belong to officers
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	me := srv.LoginAs(t, "80351110224678912", "thundermane")
	insertOwnedCharacter(t, db, healbotID, "Healbot", false, &me.ID)

	// when I send a role along with main, raid team and name changes
	resp := srv.Patch(t, "/api/me/characters/"+healbotID, map[string]any{
		"role": "healer", "isMain": true, "raidTeam": "Team 1", "name": "Renamed",
	})

	// then I expect only the role changed
	resp.RequireStatus(t, http.StatusOK)
	var got characterJSON
	resp.DecodeJSON(t, &got)
	want := characterJSON{
		ID: healbotID, Name: "Healbot", SecondaryName: "Unset", Realm: "Emberreach", Class: "Warrior",
		Role: strPtr("healer"), IsMain: false, CreatedAt: "2026-10-01T12:00:00Z",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected character (-want +got):\n%s", diff)
	}
}

func TestUpdateMyCharacter_ReturnsUnauthorized_WhenNotLoggedIn(t *testing.T) {
	// given nobody is logged in
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I try to set a character's role
	resp := srv.Patch(t, "/api/me/characters/"+tankardID, map[string]any{"role": "healer"})

	// then I expect a 401
	resp.RequireStatus(t, http.StatusUnauthorized)
}
