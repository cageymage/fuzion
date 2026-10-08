package roster_test

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jmoiron/sqlx"

	"github.com/cageymage/fuzion/backend/internal/testutil"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.Run(m))
}

type characterJSON struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	SecondaryName string  `json:"secondaryName"`
	Realm         string  `json:"realm"`
	Class         string  `json:"class"`
	Spec          *string `json:"spec"`
	Role          string  `json:"role"`
	Spec2         *string `json:"spec2"`
	Role2         *string `json:"role2"`
	IsMain        bool    `json:"isMain"`
	RaidTeam      *string `json:"raidTeam"`
	Race          *string `json:"race"`
	Level         *int    `json:"level"`
	Faction       *string `json:"faction"`
	CreatedAt     string  `json:"createdAt"`
}

func TestListRoster_ReturnsMainsBeforeAlts(t *testing.T) {
	// given an alt whose name sorts alphabetically before the guild's only main
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, spec, role, is_main, raid_team, created_at)
		VALUES
			('11111111-1111-1111-1111-111111111111', 'Aeliana', 'Dawnsong', 'Emberreach', 'Priest', 'Holy', 'healer', false, NULL, '2026-09-01T12:00:00Z'),
			('22222222-2222-2222-2222-222222222222', 'Zephyrion', 'Ironhide', 'Emberreach', 'Warrior', 'Protection', 'tank', true, 'Team 1', '2026-09-01T12:00:00Z')`)

	// when I ask for the roster
	resp := srv.Get(t, "/api/roster")

	// then I expect the main first despite its name sorting after the alt's
	resp.RequireStatus(t, 200)
	var characters []characterJSON
	resp.DecodeJSON(t, &characters)

	holy := "Holy"
	protection := "Protection"
	team1 := "Team 1"
	want := []characterJSON{
		{
			ID:            "22222222-2222-2222-2222-222222222222",
			Name:          "Zephyrion",
			SecondaryName: "Ironhide",
			Realm:         "Emberreach",
			Class:         "Warrior",
			Spec:          &protection,
			Role:          "tank",
			IsMain:        true,
			RaidTeam:      &team1,
			CreatedAt:     "2026-09-01T12:00:00Z",
		},
		{
			ID:            "11111111-1111-1111-1111-111111111111",
			Name:          "Aeliana",
			SecondaryName: "Dawnsong",
			Realm:         "Emberreach",
			Class:         "Priest",
			Spec:          &holy,
			Role:          "healer",
			IsMain:        false,
			RaidTeam:      nil,
			CreatedAt:     "2026-09-01T12:00:00Z",
		},
	}
	if diff := cmp.Diff(want, characters); diff != "" {
		t.Errorf("unexpected roster (-want +got):\n%s", diff)
	}
}

func TestListRoster_ReturnsCharactersAlphabeticallyWithinMainsAndAlts(t *testing.T) {
	// given two mains and two alts, each pair inserted out of alphabetical order
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, spec, role, is_main, raid_team, created_at)
		VALUES
			('11111111-1111-1111-1111-111111111111', 'Zenda', 'Frostwind', 'Emberreach', 'Mage', 'Frost', 'dps', true, 'Team 1', '2026-09-01T12:00:00Z'),
			('22222222-2222-2222-2222-222222222222', 'Aeliana', 'Dawnsong', 'Emberreach', 'Priest', 'Holy', 'healer', true, 'Team 1', '2026-09-01T12:00:00Z'),
			('33333333-3333-3333-3333-333333333333', 'Yorick', 'Nightblade', 'Emberreach', 'Rogue', 'Assassination', 'dps', false, NULL, '2026-09-01T12:00:00Z'),
			('44444444-4444-4444-4444-444444444444', 'Brannor', 'Wildmane', 'Emberreach', 'Druid', 'Feral', 'dps', false, NULL, '2026-09-01T12:00:00Z')`)

	// when I ask for the roster
	resp := srv.Get(t, "/api/roster")

	// then I expect mains sorted alphabetically, then alts sorted alphabetically
	resp.RequireStatus(t, 200)
	var characters []characterJSON
	resp.DecodeJSON(t, &characters)

	names := make([]string, len(characters))
	for i, c := range characters {
		names[i] = c.Name
	}
	want := []string{"Aeliana", "Zenda", "Brannor", "Yorick"}
	if diff := cmp.Diff(want, names); diff != "" {
		t.Errorf("unexpected roster order (-want +got):\n%s", diff)
	}
}

func TestListRoster_ReturnsSharedFirstNamesOrderedBySecondaryName(t *testing.T) {
	// given two mains who share a first name, inserted out of order by secondary name
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, role, is_main)
		VALUES
			('11111111-1111-1111-1111-111111111111', 'Aeliana', 'Stormwind', 'Emberreach', 'Priest', 'healer', true),
			('22222222-2222-2222-2222-222222222222', 'Aeliana', 'Dawnsong', 'Emberreach', 'Mage', 'dps', true)`)

	// when I ask for the roster
	resp := srv.Get(t, "/api/roster")

	// then I expect the shared first name ordered by secondary name
	resp.RequireStatus(t, 200)
	var characters []characterJSON
	resp.DecodeJSON(t, &characters)

	secondaryNames := make([]string, len(characters))
	for i, c := range characters {
		secondaryNames[i] = c.SecondaryName
	}
	want := []string{"Dawnsong", "Stormwind"}
	if diff := cmp.Diff(want, secondaryNames); diff != "" {
		t.Errorf("unexpected secondary name order (-want +got):\n%s", diff)
	}
}

func TestListRoster_ReturnsEmptyArray_WhenNoCharactersExist(t *testing.T) {
	// given a guild that has not entered any characters yet
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I ask for the roster
	resp := srv.Get(t, "/api/roster")

	// then I expect a 200 with an empty JSON array rather than null
	resp.RequireStatus(t, 200)
	if got := string(resp.Body); got != "[]\n" {
		t.Errorf("expected an empty JSON array, got %q", got)
	}
}

func countCharacters(t *testing.T, db *sqlx.DB) int {
	t.Helper()

	var count int
	if err := db.Get(&count, `SELECT count(*) FROM characters`); err != nil {
		t.Fatalf("count characters: %v", err)
	}
	return count
}

func requireErrorBody(t *testing.T, resp testutil.Response, want string) {
	t.Helper()

	var body map[string]string
	resp.DecodeJSON(t, &body)
	if diff := cmp.Diff(map[string]string{"error": want}, body); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
}

func TestCreateCharacter_ReturnsCreatedCharacter_WhenOfficerSubmitsValidBody(t *testing.T) {
	// given I am logged in as an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I create a character without a realm
	resp := srv.Post(t, "/api/roster", map[string]any{
		"name":          "  Aeliana ",
		"secondaryName": " Dawnsong  ",
		"class":         "Priest",
		"spec":          "Holy",
		"role":          "healer",
		"isMain":        true,
		"raidTeam":      "Team 1",
	})

	// then I expect a 201 with both names trimmed and the default realm
	resp.RequireStatus(t, http.StatusCreated)
	var created characterJSON
	resp.DecodeJSON(t, &created)

	holy := "Holy"
	team1 := "Team 1"
	want := characterJSON{
		ID:            created.ID,
		Name:          "Aeliana",
		SecondaryName: "Dawnsong",
		Realm:         "Emberreach",
		Class:         "Priest",
		Spec:          &holy,
		Role:          "healer",
		IsMain:        true,
		RaidTeam:      &team1,
		CreatedAt:     created.CreatedAt,
	}
	if diff := cmp.Diff(want, created); diff != "" {
		t.Errorf("unexpected created character (-want +got):\n%s", diff)
	}
	if created.ID == "" || created.CreatedAt == "" {
		t.Errorf("expected the server to generate an id and createdAt, got %+v", created)
	}
	if got := countCharacters(t, db); got != 1 {
		t.Errorf("expected the character to be stored once, found %d", got)
	}
}

func TestCreateCharacter_ReturnsCreatedCharacter_WhenBothNamesAreTwelveCharacters(t *testing.T) {
	// given I am logged in as an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I create a character whose names are each at the 12 character limit
	resp := srv.Post(t, "/api/roster", map[string]any{
		"name":          "Abcdefghijkl",
		"secondaryName": "Mnopqrstuvwx",
		"class":         "Mage",
		"role":          "dps",
	})

	// then I expect a 201
	resp.RequireStatus(t, http.StatusCreated)
}

func TestCreateCharacter_ReturnsCreatedCharacter_WhenFirstNameIsSharedWithAnotherCharacter(t *testing.T) {
	// given a character who already uses the first name Aeliana
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, role)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Aeliana', 'Dawnsong', 'Emberreach', 'Priest', 'healer')`)

	// when I create a character with the same first name but a different secondary name
	resp := srv.Post(t, "/api/roster", map[string]any{
		"name":          "Aeliana",
		"secondaryName": "Stormwind",
		"class":         "Mage",
		"role":          "dps",
	})

	// then I expect a 201
	resp.RequireStatus(t, http.StatusCreated)
}

func TestCreateCharacter_ReturnsUnauthorized_WhenCallerIsAnonymous(t *testing.T) {
	// given nobody is logged in
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I create a character
	resp := srv.Post(t, "/api/roster", map[string]any{"name": "Aeliana", "secondaryName": "Dawnsong", "class": "Priest", "role": "healer"})

	// then I expect a 401
	resp.RequireStatus(t, http.StatusUnauthorized)
}

func TestCreateCharacter_ReturnsForbidden_WhenCallerIsNotAnOfficer(t *testing.T) {
	// given I am logged in as a regular member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "member-1", "Member")

	// when I create a character
	resp := srv.Post(t, "/api/roster", map[string]any{"name": "Aeliana", "secondaryName": "Dawnsong", "class": "Priest", "role": "healer"})

	// then I expect a 403 and nothing stored
	resp.RequireStatus(t, http.StatusForbidden)
	if got := countCharacters(t, db); got != 0 {
		t.Errorf("expected no characters to be stored, found %d", got)
	}
}

func TestCreateCharacter_ReturnsBadRequest_WhenRoleIsInvalid(t *testing.T) {
	// given I am logged in as an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I create a character with an unknown role
	resp := srv.Post(t, "/api/roster", map[string]any{"name": "Aeliana", "secondaryName": "Dawnsong", "class": "Priest", "role": "bard"})

	// then I expect a 400 naming the role field
	resp.RequireStatus(t, http.StatusBadRequest)
	requireErrorBody(t, resp, "role: must be one of tank, healer, dps")
}

func TestCreateCharacter_ReturnsBadRequest_WhenNameIsTooShort(t *testing.T) {
	// given I am logged in as an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I create a character whose trimmed name is one character long
	resp := srv.Post(t, "/api/roster", map[string]any{"name": " A ", "secondaryName": "Dawnsong", "class": "Priest", "role": "healer"})

	// then I expect a 400 naming the name field
	resp.RequireStatus(t, http.StatusBadRequest)
	requireErrorBody(t, resp, "name: must be between 2 and 12 characters")
}

func TestCreateCharacter_ReturnsBadRequest_WhenNameIsTooLong(t *testing.T) {
	// given I am logged in as an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I create a character whose name is 13 characters long
	resp := srv.Post(t, "/api/roster", map[string]any{"name": "Abcdefghijklm", "secondaryName": "Dawnsong", "class": "Priest", "role": "healer"})

	// then I expect a 400 naming the name field
	resp.RequireStatus(t, http.StatusBadRequest)
	requireErrorBody(t, resp, "name: must be between 2 and 12 characters")
}

func TestCreateCharacter_ReturnsBadRequest_WhenSecondaryNameIsMissing(t *testing.T) {
	// given I am logged in as an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I create a character without a secondary name
	resp := srv.Post(t, "/api/roster", map[string]any{"name": "Aeliana", "class": "Priest", "role": "healer"})

	// then I expect a 400 naming the secondaryName field
	resp.RequireStatus(t, http.StatusBadRequest)
	requireErrorBody(t, resp, "secondaryName: must be between 2 and 12 characters")
}

func TestCreateCharacter_ReturnsBadRequest_WhenSecondaryNameIsTooLong(t *testing.T) {
	// given I am logged in as an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I create a character whose secondary name is 13 characters long
	resp := srv.Post(t, "/api/roster", map[string]any{"name": "Aeliana", "secondaryName": "Abcdefghijklm", "class": "Priest", "role": "healer"})

	// then I expect a 400 naming the secondaryName field
	resp.RequireStatus(t, http.StatusBadRequest)
	requireErrorBody(t, resp, "secondaryName: must be between 2 and 12 characters")
}

func TestCreateCharacter_ReturnsBadRequest_WhenClassIsNotAKnownClass(t *testing.T) {
	// given I am logged in as an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I create a character with an unknown class
	resp := srv.Post(t, "/api/roster", map[string]any{"name": "Aeliana", "secondaryName": "Dawnsong", "class": "Bard", "role": "healer"})

	// then I expect a 400 naming the class field
	resp.RequireStatus(t, http.StatusBadRequest)
	requireErrorBody(t, resp, "class: must be one of Warrior, Paladin, Hunter, Rogue, Priest, Shaman, Mage, Warlock, Druid")
}

func TestCreateCharacter_ReturnsConflict_WhenFullNameAlreadyExists(t *testing.T) {
	// given a character that already exists
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, role)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Aeliana', 'Dawnsong', 'Emberreach', 'Priest', 'healer')`)

	// when I create another character with the same name and secondary name
	resp := srv.Post(t, "/api/roster", map[string]any{"name": "Aeliana", "secondaryName": "Dawnsong", "class": "Mage", "role": "dps"})

	// then I expect a 409
	resp.RequireStatus(t, http.StatusConflict)
}

func TestUpdateCharacter_ChangesOnlyProvidedFields(t *testing.T) {
	// given an existing character
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, spec, role, is_main, raid_team, created_at)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Aeliana', 'Dawnsong', 'Emberreach', 'Priest', 'Holy', 'healer', false, 'Team 1', '2026-09-01T12:00:00Z')`)

	// when I change only the role and main flag
	resp := srv.Patch(t, "/api/roster/11111111-1111-1111-1111-111111111111", map[string]any{
		"role":   "dps",
		"isMain": true,
	})

	// then I expect a 200 where only those fields changed
	resp.RequireStatus(t, http.StatusOK)
	var updated characterJSON
	resp.DecodeJSON(t, &updated)

	holy := "Holy"
	team1 := "Team 1"
	want := characterJSON{
		ID:            "11111111-1111-1111-1111-111111111111",
		Name:          "Aeliana",
		SecondaryName: "Dawnsong",
		Realm:         "Emberreach",
		Class:         "Priest",
		Spec:          &holy,
		Role:          "dps",
		IsMain:        true,
		RaidTeam:      &team1,
		CreatedAt:     "2026-09-01T12:00:00Z",
	}
	if diff := cmp.Diff(want, updated); diff != "" {
		t.Errorf("unexpected updated character (-want +got):\n%s", diff)
	}
}

func TestUpdateCharacter_ChangesSecondaryName_WhenOnlySecondaryNameIsSent(t *testing.T) {
	// given an existing character
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, role)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Aeliana', 'Dawnsong', 'Emberreach', 'Priest', 'healer')`)

	// when I change only the secondary name
	resp := srv.Patch(t, "/api/roster/11111111-1111-1111-1111-111111111111", map[string]any{"secondaryName": "Stormwind"})

	// then I expect the secondary name changed and the first name kept
	resp.RequireStatus(t, http.StatusOK)
	var updated characterJSON
	resp.DecodeJSON(t, &updated)
	if updated.Name != "Aeliana" || updated.SecondaryName != "Stormwind" {
		t.Errorf("expected Aeliana Stormwind, got %s %s", updated.Name, updated.SecondaryName)
	}
}

func TestUpdateCharacter_ReturnsBadRequest_WhenSecondaryNameIsTooLong(t *testing.T) {
	// given an existing character
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, role)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Aeliana', 'Dawnsong', 'Emberreach', 'Priest', 'healer')`)

	// when I set a 13 character secondary name
	resp := srv.Patch(t, "/api/roster/11111111-1111-1111-1111-111111111111", map[string]any{"secondaryName": "Abcdefghijklm"})

	// then I expect a 400 naming the secondaryName field
	resp.RequireStatus(t, http.StatusBadRequest)
	requireErrorBody(t, resp, "secondaryName: must be between 2 and 12 characters")
}

func TestUpdateCharacter_ReturnsConflict_WhenRenamedOntoAnExistingFullName(t *testing.T) {
	// given two characters
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, role)
		VALUES
			('11111111-1111-1111-1111-111111111111', 'Aeliana', 'Dawnsong', 'Emberreach', 'Priest', 'healer'),
			('22222222-2222-2222-2222-222222222222', 'Aeliana', 'Stormwind', 'Emberreach', 'Warrior', 'tank')`)

	// when I change the second character's secondary name to match the first's
	resp := srv.Patch(t, "/api/roster/22222222-2222-2222-2222-222222222222", map[string]any{"secondaryName": "Dawnsong"})

	// then I expect a 409
	resp.RequireStatus(t, http.StatusConflict)
}

func TestUpdateCharacter_ReturnsNotFound_WhenIDDoesNotExist(t *testing.T) {
	// given I am logged in as an officer and no characters exist
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I update a character that does not exist
	resp := srv.Patch(t, "/api/roster/99999999-9999-9999-9999-999999999999", map[string]any{"role": "dps"})

	// then I expect a 404
	resp.RequireStatus(t, http.StatusNotFound)
}

func TestUpdateCharacter_ReturnsForbidden_WhenCallerIsNotAnOfficer(t *testing.T) {
	// given a character and a logged-in regular member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "member-1", "Member")
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, role)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Aeliana', 'Dawnsong', 'Emberreach', 'Priest', 'healer')`)

	// when I update the character
	resp := srv.Patch(t, "/api/roster/11111111-1111-1111-1111-111111111111", map[string]any{"role": "dps"})

	// then I expect a 403
	resp.RequireStatus(t, http.StatusForbidden)
}

func TestDeleteCharacter_RemovesCharacter_WhenOfficerDeletes(t *testing.T) {
	// given an existing character
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, role)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Aeliana', 'Dawnsong', 'Emberreach', 'Priest', 'healer')`)

	// when I delete it
	resp := srv.Delete(t, "/api/roster/11111111-1111-1111-1111-111111111111")

	// then I expect a 204 and the character gone
	resp.RequireStatus(t, http.StatusNoContent)
	if got := countCharacters(t, db); got != 0 {
		t.Errorf("expected the character to be deleted, found %d rows", got)
	}
}

func TestDeleteCharacter_ReturnsNotFound_WhenIDDoesNotExist(t *testing.T) {
	// given I am logged in as an officer and no characters exist
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I delete a character that does not exist
	resp := srv.Delete(t, "/api/roster/99999999-9999-9999-9999-999999999999")

	// then I expect a 404
	resp.RequireStatus(t, http.StatusNotFound)
}

func TestDeleteCharacter_ReturnsUnauthorized_WhenCallerIsAnonymous(t *testing.T) {
	// given a character and nobody logged in
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, role)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Aeliana', 'Dawnsong', 'Emberreach', 'Priest', 'healer')`)

	// when I delete the character
	resp := srv.Delete(t, "/api/roster/11111111-1111-1111-1111-111111111111")

	// then I expect a 401
	resp.RequireStatus(t, http.StatusUnauthorized)
}

func TestListRoster_ReturnsSecondSpecAndRole_WhenCharacterHasOne(t *testing.T) {
	// given a rogue who runs a second spec in a different role than the first
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, spec, role, spec2, role2, is_main, created_at)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Yorick', 'Nightblade', 'Emberreach', 'Rogue', 'Assassination', 'dps', 'Combat', 'tank', true, '2026-09-01T12:00:00Z')`)

	// when I ask for the roster
	resp := srv.Get(t, "/api/roster")

	// then I expect both specs and both roles
	resp.RequireStatus(t, 200)
	var characters []characterJSON
	resp.DecodeJSON(t, &characters)

	assassination := "Assassination"
	combat := "Combat"
	tank := "tank"
	want := []characterJSON{
		{
			ID:            "11111111-1111-1111-1111-111111111111",
			Name:          "Yorick",
			SecondaryName: "Nightblade",
			Realm:         "Emberreach",
			Class:         "Rogue",
			Spec:          &assassination,
			Role:          "dps",
			Spec2:         &combat,
			Role2:         &tank,
			IsMain:        true,
			CreatedAt:     "2026-09-01T12:00:00Z",
		},
	}
	if diff := cmp.Diff(want, characters); diff != "" {
		t.Errorf("unexpected roster (-want +got):\n%s", diff)
	}
}

func TestListRoster_ReturnsNullSecondSpecAndRole_ForSingleSpecCharacters(t *testing.T) {
	// given a character with only one spec
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, spec, role, is_main, created_at)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Aeliana', 'Dawnsong', 'Emberreach', 'Priest', 'Holy', 'healer', true, '2026-09-01T12:00:00Z')`)

	// when I ask for the roster
	resp := srv.Get(t, "/api/roster")

	// then I expect spec2 and role2 to be explicit nulls
	resp.RequireStatus(t, 200)
	var characters []map[string]any
	resp.DecodeJSON(t, &characters)
	if len(characters) != 1 {
		t.Fatalf("expected one character, got %d", len(characters))
	}
	for _, field := range []string{"spec2", "role2"} {
		value, present := characters[0][field]
		if !present || value != nil {
			t.Errorf("expected %s to be present and null, got present=%v value=%v", field, present, value)
		}
	}
}

func TestCreateCharacter_ReturnsCreatedCharacter_WhenBothSpecsAndRolesAreSent(t *testing.T) {
	// given I am logged in as an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I create a rogue with two specs and a role for each
	resp := srv.Post(t, "/api/roster", map[string]any{
		"name":          "Yorick",
		"secondaryName": "Nightblade",
		"class":         "Rogue",
		"spec":          "Assassination",
		"role":          "dps",
		"spec2":         " Combat ",
		"role2":         "tank",
	})

	// then I expect a 201 with both specs and roles stored
	resp.RequireStatus(t, http.StatusCreated)
	var created characterJSON
	resp.DecodeJSON(t, &created)

	assassination := "Assassination"
	combat := "Combat"
	tank := "tank"
	want := characterJSON{
		ID:            created.ID,
		Name:          "Yorick",
		SecondaryName: "Nightblade",
		Realm:         "Emberreach",
		Class:         "Rogue",
		Spec:          &assassination,
		Role:          "dps",
		Spec2:         &combat,
		Role2:         &tank,
		IsMain:        false,
		CreatedAt:     created.CreatedAt,
	}
	if diff := cmp.Diff(want, created); diff != "" {
		t.Errorf("unexpected created character (-want +got):\n%s", diff)
	}
}

func TestCreateCharacter_ReturnsBadRequest_WhenRole2IsSetWithoutSpec2(t *testing.T) {
	// given I am logged in as an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I create a character with a second role but no second spec
	resp := srv.Post(t, "/api/roster", map[string]any{
		"name": "Aeliana", "secondaryName": "Dawnsong", "class": "Priest",
		"spec": "Holy", "role": "healer", "role2": "dps",
	})

	// then I expect a 400 naming spec2 and nothing stored
	resp.RequireStatus(t, http.StatusBadRequest)
	requireErrorBody(t, resp, "spec2: must be set together with role2")
	if got := countCharacters(t, db); got != 0 {
		t.Errorf("expected no characters to be stored, found %d", got)
	}
}

func TestCreateCharacter_ReturnsBadRequest_WhenSpec2IsSetWithoutRole2(t *testing.T) {
	// given I am logged in as an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I create a character with a second spec but no second role
	resp := srv.Post(t, "/api/roster", map[string]any{
		"name": "Aeliana", "secondaryName": "Dawnsong", "class": "Priest",
		"spec": "Holy", "role": "healer", "spec2": "Shadow",
	})

	// then I expect a 400 naming role2
	resp.RequireStatus(t, http.StatusBadRequest)
	requireErrorBody(t, resp, "role2: must be set together with spec2")
}

func TestCreateCharacter_ReturnsBadRequest_WhenRole2IsInvalid(t *testing.T) {
	// given I am logged in as an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I create a character with an unknown second role
	resp := srv.Post(t, "/api/roster", map[string]any{
		"name": "Aeliana", "secondaryName": "Dawnsong", "class": "Priest",
		"spec": "Holy", "role": "healer", "spec2": "Shadow", "role2": "bard",
	})

	// then I expect a 400 naming role2
	resp.RequireStatus(t, http.StatusBadRequest)
	requireErrorBody(t, resp, "role2: must be one of tank, healer, dps")
}

func TestCreateCharacter_ReturnsBadRequest_WhenSpec2EqualsSpec(t *testing.T) {
	// given I am logged in as an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I create a character whose second spec repeats the first
	resp := srv.Post(t, "/api/roster", map[string]any{
		"name": "Aeliana", "secondaryName": "Dawnsong", "class": "Priest",
		"spec": "Holy", "role": "healer", "spec2": "Holy", "role2": "healer",
	})

	// then I expect a 400 naming spec2
	resp.RequireStatus(t, http.StatusBadRequest)
	requireErrorBody(t, resp, "spec2: must differ from spec")
}

func TestCreateCharacter_ReturnsBadRequest_WhenSpecIsNotInTheClassSpecList(t *testing.T) {
	// given I am logged in as an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I create a priest with a warrior spec
	resp := srv.Post(t, "/api/roster", map[string]any{
		"name": "Aeliana", "secondaryName": "Dawnsong", "class": "Priest",
		"spec": "Fury", "role": "healer",
	})

	// then I expect a 400 listing the priest specs
	resp.RequireStatus(t, http.StatusBadRequest)
	requireErrorBody(t, resp, "spec: must be one of Discipline, Holy, Shadow")
}

func TestCreateCharacter_ReturnsBadRequest_WhenSpec2IsNotInTheClassSpecList(t *testing.T) {
	// given I am logged in as an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I create a priest whose second spec belongs to another class
	resp := srv.Post(t, "/api/roster", map[string]any{
		"name": "Aeliana", "secondaryName": "Dawnsong", "class": "Priest",
		"spec": "Holy", "role": "healer", "spec2": "Fury", "role2": "dps",
	})

	// then I expect a 400 listing the priest specs
	resp.RequireStatus(t, http.StatusBadRequest)
	requireErrorBody(t, resp, "spec2: must be one of Discipline, Holy, Shadow")
}

func TestUpdateCharacter_SetsSecondSpecAndRole_WhenBothAreSent(t *testing.T) {
	// given a single-spec character
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, spec, role, is_main, created_at)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Aeliana', 'Dawnsong', 'Emberreach', 'Priest', 'Holy', 'healer', true, '2026-09-01T12:00:00Z')`)

	// when I add a second spec and role
	resp := srv.Patch(t, "/api/roster/11111111-1111-1111-1111-111111111111", map[string]any{
		"spec2": "Shadow",
		"role2": "dps",
	})

	// then I expect a 200 where only the second spec and role were added
	resp.RequireStatus(t, http.StatusOK)
	var updated characterJSON
	resp.DecodeJSON(t, &updated)

	holy := "Holy"
	shadow := "Shadow"
	dps := "dps"
	want := characterJSON{
		ID:            "11111111-1111-1111-1111-111111111111",
		Name:          "Aeliana",
		SecondaryName: "Dawnsong",
		Realm:         "Emberreach",
		Class:         "Priest",
		Spec:          &holy,
		Role:          "healer",
		Spec2:         &shadow,
		Role2:         &dps,
		IsMain:        true,
		CreatedAt:     "2026-09-01T12:00:00Z",
	}
	if diff := cmp.Diff(want, updated); diff != "" {
		t.Errorf("unexpected updated character (-want +got):\n%s", diff)
	}
}

func TestUpdateCharacter_ClearsSecondSpecAndRole_WhenBothAreEmptyStrings(t *testing.T) {
	// given a dual-spec character
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, spec, role, spec2, role2, is_main, created_at)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Aeliana', 'Dawnsong', 'Emberreach', 'Priest', 'Holy', 'healer', 'Shadow', 'dps', true, '2026-09-01T12:00:00Z')`)

	// when I send empty strings for the second spec and role
	resp := srv.Patch(t, "/api/roster/11111111-1111-1111-1111-111111111111", map[string]any{
		"spec2": "",
		"role2": "",
	})

	// then I expect a 200 where the second spec and role are null
	resp.RequireStatus(t, http.StatusOK)
	var updated characterJSON
	resp.DecodeJSON(t, &updated)

	holy := "Holy"
	want := characterJSON{
		ID:            "11111111-1111-1111-1111-111111111111",
		Name:          "Aeliana",
		SecondaryName: "Dawnsong",
		Realm:         "Emberreach",
		Class:         "Priest",
		Spec:          &holy,
		Role:          "healer",
		IsMain:        true,
		CreatedAt:     "2026-09-01T12:00:00Z",
	}
	if diff := cmp.Diff(want, updated); diff != "" {
		t.Errorf("unexpected updated character (-want +got):\n%s", diff)
	}
}

func TestUpdateCharacter_ReturnsBadRequest_WhenOnlyRole2IsCleared(t *testing.T) {
	// given a dual-spec character
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, spec, role, spec2, role2)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Aeliana', 'Dawnsong', 'Emberreach', 'Priest', 'Holy', 'healer', 'Shadow', 'dps')`)

	// when I clear only the second role
	resp := srv.Patch(t, "/api/roster/11111111-1111-1111-1111-111111111111", map[string]any{"role2": ""})

	// then I expect a 400 and the stored pair left intact
	resp.RequireStatus(t, http.StatusBadRequest)
	requireErrorBody(t, resp, "role2: must be set together with spec2")
	var stored string
	if err := db.Get(&stored, `SELECT role2 FROM characters WHERE id = '11111111-1111-1111-1111-111111111111'`); err != nil || stored != "dps" {
		t.Errorf("expected role2 to remain dps, got %q (err %v)", stored, err)
	}
}

func TestUpdateCharacter_ReturnsBadRequest_WhenSpec2IsChangedToTheStoredSpec(t *testing.T) {
	// given a dual-spec character
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, spec, role, spec2, role2)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Aeliana', 'Dawnsong', 'Emberreach', 'Priest', 'Holy', 'healer', 'Shadow', 'dps')`)

	// when I change the second spec to match the first
	resp := srv.Patch(t, "/api/roster/11111111-1111-1111-1111-111111111111", map[string]any{"spec2": "Holy"})

	// then I expect a 400 naming spec2
	resp.RequireStatus(t, http.StatusBadRequest)
	requireErrorBody(t, resp, "spec2: must differ from spec")
}

func TestUpdateCharacter_ReturnsBadRequest_WhenClassChangeInvalidatesTheStoredSpec(t *testing.T) {
	// given a holy priest
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, spec, role)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Aeliana', 'Dawnsong', 'Emberreach', 'Priest', 'Holy', 'healer')`)

	// when I change only the class to Mage
	resp := srv.Patch(t, "/api/roster/11111111-1111-1111-1111-111111111111", map[string]any{"class": "Mage"})

	// then I expect a 400 because Holy is not a mage spec
	resp.RequireStatus(t, http.StatusBadRequest)
	requireErrorBody(t, resp, "spec: must be one of Arcane, Fire, Frost")
}

func TestUpdateCharacter_ChangesRaidTeam_WhenStoredSpecIsOutsideTheClassSpecList(t *testing.T) {
	// given a character whose free-text spec predates the spec list
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, spec, role)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Aeliana', 'Dawnsong', 'Emberreach', 'Priest', 'Holy/Disc', 'healer')`)

	// when I change only the raid team
	resp := srv.Patch(t, "/api/roster/11111111-1111-1111-1111-111111111111", map[string]any{"raidTeam": "Team 2"})

	// then I expect a 200 because the untouched spec is not re-validated
	resp.RequireStatus(t, http.StatusOK)
}

type rosterProfessionsJSON struct {
	Name        string `json:"name"`
	Professions []struct {
		Profession string `json:"profession"`
		SkillLevel int    `json:"skillLevel"`
	} `json:"professions"`
}

func TestListRoster_IncludesProfessions_ForCharacterWithPrimaryProfessions(t *testing.T) {
	// given a character who knows two professions at different skill levels
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, role, is_main)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Aeliana', 'Dawnsong', 'Emberreach', 'Priest', 'healer', true)`)
	db.MustExec(`
		INSERT INTO professions (id, character_id, profession, skill_level)
		VALUES
			('a1111111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111111', 'Alchemy', 225),
			('a2222222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111111', 'Herbalism', 300)`)

	// when I ask for the roster
	resp := srv.Get(t, "/api/roster")

	// then I expect the character's professions embedded, highest skill first
	resp.RequireStatus(t, http.StatusOK)
	var characters []rosterProfessionsJSON
	resp.DecodeJSON(t, &characters)

	if len(characters) != 1 {
		t.Fatalf("expected 1 character, got %d", len(characters))
	}
	want := []struct {
		Profession string `json:"profession"`
		SkillLevel int    `json:"skillLevel"`
	}{
		{Profession: "Herbalism", SkillLevel: 300},
		{Profession: "Alchemy", SkillLevel: 225},
	}
	if diff := cmp.Diff(want, characters[0].Professions); diff != "" {
		t.Errorf("unexpected professions (-want +got):\n%s", diff)
	}
}

func TestListRoster_ReturnsEmptyProfessionsArray_ForCharacterWithNoProfessions(t *testing.T) {
	// given a character with no professions
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, role, is_main)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Aeliana', 'Dawnsong', 'Emberreach', 'Priest', 'healer', true)`)

	// when I ask for the roster
	resp := srv.Get(t, "/api/roster")

	// then I expect an empty professions array, not null
	resp.RequireStatus(t, http.StatusOK)
	var raw []map[string]json.RawMessage
	resp.DecodeJSON(t, &raw)

	if len(raw) != 1 {
		t.Fatalf("expected 1 character, got %d", len(raw))
	}
	if got := string(raw[0]["professions"]); got != "[]" {
		t.Errorf("expected professions to be [], got %s", got)
	}
}

func TestUpdateCharacter_ReturnsProfessions_WhenCharacterHasProfessions(t *testing.T) {
	// given I am an officer and a character who knows Mining
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, role, is_main)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Aeliana', 'Dawnsong', 'Emberreach', 'Priest', 'healer', true)`)
	db.MustExec(`
		INSERT INTO professions (id, character_id, profession, skill_level)
		VALUES ('a1111111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111111', 'Mining', 150)`)

	// when I change her raid team
	resp := srv.Patch(t, "/api/roster/11111111-1111-1111-1111-111111111111", map[string]any{"raidTeam": "Team 1"})

	// then I expect the updated character to carry her professions
	resp.RequireStatus(t, http.StatusOK)
	var updated rosterProfessionsJSON
	resp.DecodeJSON(t, &updated)

	want := []struct {
		Profession string `json:"profession"`
		SkillLevel int    `json:"skillLevel"`
	}{{Profession: "Mining", SkillLevel: 150}}
	if diff := cmp.Diff(want, updated.Professions); diff != "" {
		t.Errorf("unexpected professions (-want +got):\n%s", diff)
	}
}

type professionJSON struct {
	Profession string `json:"profession"`
	SkillLevel int    `json:"skillLevel"`
}

const seedAeliana = `
	INSERT INTO characters (id, name, secondary_name, realm, class, role, is_main)
	VALUES ('11111111-1111-1111-1111-111111111111', 'Aeliana', 'Dawnsong', 'Emberreach', 'Priest', 'healer', true)`

func createBody(professions any) map[string]any {
	return map[string]any{
		"name":          "Aeliana",
		"secondaryName": "Dawnsong",
		"class":         "Priest",
		"role":          "healer",
		"professions":   professions,
	}
}

func TestCreateCharacter_StoresProfessions_WhenProfessionsAreGiven(t *testing.T) {
	// given I am logged in as an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I create a character with two primary and one secondary profession
	resp := srv.Post(t, "/api/roster", createBody([]string{"Mining", "Herbalism", "Cooking"}))

	// then I expect the character back with those professions at skill level 0
	resp.RequireStatus(t, http.StatusCreated)
	var created struct {
		Professions []professionJSON `json:"professions"`
	}
	resp.DecodeJSON(t, &created)

	want := []professionJSON{
		{Profession: "Cooking", SkillLevel: 0},
		{Profession: "Herbalism", SkillLevel: 0},
		{Profession: "Mining", SkillLevel: 0},
	}
	if diff := cmp.Diff(want, created.Professions); diff != "" {
		t.Errorf("unexpected professions (-want +got):\n%s", diff)
	}
}

func TestCreateCharacter_ReturnsBadRequest_WhenProfessionIsUnknown(t *testing.T) {
	// given I am logged in as an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I create a character with a profession that does not exist in Forever
	resp := srv.Post(t, "/api/roster", createBody([]string{"Jewelcrafting"}))

	// then I expect a 400 naming the profession and no character stored
	resp.RequireStatus(t, http.StatusBadRequest)
	var body map[string]string
	resp.DecodeJSON(t, &body)
	if diff := cmp.Diff(map[string]string{"error": `professions: "Jewelcrafting" is not a known profession`}, body); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
	var count int
	if err := db.Get(&count, `SELECT count(*) FROM characters`); err != nil {
		t.Fatalf("count characters: %v", err)
	}
	if count != 0 {
		t.Errorf("expected no character to be stored, got %d", count)
	}
}

func TestCreateCharacter_ReturnsBadRequest_WhenMoreThanTwoPrimaryProfessionsAreGiven(t *testing.T) {
	// given I am logged in as an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I create a character with three primary professions
	resp := srv.Post(t, "/api/roster", createBody([]string{"Mining", "Herbalism", "Skinning"}))

	// then I expect a 400
	resp.RequireStatus(t, http.StatusBadRequest)
	var body map[string]string
	resp.DecodeJSON(t, &body)
	if diff := cmp.Diff(map[string]string{"error": "professions: at most 2 primary professions are allowed"}, body); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
}

func TestCreateCharacter_ReturnsBadRequest_WhenProfessionIsRepeated(t *testing.T) {
	// given I am logged in as an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I create a character with the same profession twice
	resp := srv.Post(t, "/api/roster", createBody([]string{"Mining", "Mining"}))

	// then I expect a 400
	resp.RequireStatus(t, http.StatusBadRequest)
	var body map[string]string
	resp.DecodeJSON(t, &body)
	if diff := cmp.Diff(map[string]string{"error": `professions: "Mining" is listed more than once`}, body); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
}

func TestUpdateCharacter_ReplacesProfessionsAndKeepsSkill_WhenProfessionsAreGiven(t *testing.T) {
	// given I am an officer and a character who knows Mining at 300 and Herbalism at 150
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	db.MustExec(seedAeliana)
	db.MustExec(`
		INSERT INTO professions (id, character_id, profession, skill_level)
		VALUES
			('a1111111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111111', 'Mining', 300),
			('a2222222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111111', 'Herbalism', 150)`)

	// when I set her professions to Mining and Cooking
	resp := srv.Patch(t, "/api/roster/11111111-1111-1111-1111-111111111111", map[string]any{"professions": []string{"Mining", "Cooking"}})

	// then I expect Mining to keep its skill, Herbalism gone, and Cooking added at 0
	resp.RequireStatus(t, http.StatusOK)
	var updated struct {
		Professions []professionJSON `json:"professions"`
	}
	resp.DecodeJSON(t, &updated)

	want := []professionJSON{
		{Profession: "Mining", SkillLevel: 300},
		{Profession: "Cooking", SkillLevel: 0},
	}
	if diff := cmp.Diff(want, updated.Professions); diff != "" {
		t.Errorf("unexpected professions (-want +got):\n%s", diff)
	}
}

func TestUpdateCharacter_LeavesProfessionsAlone_WhenProfessionsAreOmitted(t *testing.T) {
	// given I am an officer and a character who knows Mining
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	db.MustExec(seedAeliana)
	db.MustExec(`
		INSERT INTO professions (id, character_id, profession, skill_level)
		VALUES ('a1111111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111111', 'Mining', 300)`)

	// when I change only her raid team
	resp := srv.Patch(t, "/api/roster/11111111-1111-1111-1111-111111111111", map[string]any{"raidTeam": "Team 1"})

	// then I expect her professions unchanged
	resp.RequireStatus(t, http.StatusOK)
	var updated struct {
		Professions []professionJSON `json:"professions"`
	}
	resp.DecodeJSON(t, &updated)

	if diff := cmp.Diff([]professionJSON{{Profession: "Mining", SkillLevel: 300}}, updated.Professions); diff != "" {
		t.Errorf("unexpected professions (-want +got):\n%s", diff)
	}
}

func TestUpdateCharacter_ClearsProfessions_WhenProfessionsIsEmpty(t *testing.T) {
	// given I am an officer and a character who knows Mining
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	db.MustExec(seedAeliana)
	db.MustExec(`
		INSERT INTO professions (id, character_id, profession, skill_level)
		VALUES ('a1111111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111111', 'Mining', 300)`)

	// when I set her professions to an empty list
	resp := srv.Patch(t, "/api/roster/11111111-1111-1111-1111-111111111111", map[string]any{"professions": []string{}})

	// then I expect no professions left
	resp.RequireStatus(t, http.StatusOK)
	var updated struct {
		Professions []professionJSON `json:"professions"`
	}
	resp.DecodeJSON(t, &updated)

	if diff := cmp.Diff([]professionJSON{}, updated.Professions); diff != "" {
		t.Errorf("unexpected professions (-want +got):\n%s", diff)
	}
}

const wantRaceError = "race: must be one of Dwarf, Gnome, Human, Night Elf, Orc, Skyborne (High Order), Skyborne (Windshaper), Tauren, Troll, Undead"

func TestListRoster_ReturnsFactionDerivedFromRace_WhenCharacterHasRace(t *testing.T) {
	// given an Orc, a Dwarf and one Skyborne of each faction
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, role, is_main, race, created_at)
		VALUES
			('11111111-1111-1111-1111-111111111111', 'Aeliana', 'Dawnsong', 'Emberreach', 'Warrior', 'tank', true, 'Orc', '2026-09-01T12:00:00Z'),
			('22222222-2222-2222-2222-222222222222', 'Bronzebeard', 'Ironhide', 'Emberreach', 'Warrior', 'tank', true, 'Dwarf', '2026-09-01T12:00:00Z'),
			('33333333-3333-3333-3333-333333333333', 'Cirrus', 'Highwind', 'Emberreach', 'Mage', 'dps', true, 'Skyborne (High Order)', '2026-09-01T12:00:00Z'),
			('44444444-4444-4444-4444-444444444444', 'Dusk', 'Galeborn', 'Emberreach', 'Shaman', 'healer', true, 'Skyborne (Windshaper)', '2026-09-01T12:00:00Z')`)

	// when I ask for the roster
	resp := srv.Get(t, "/api/roster")

	// then I expect each character's faction to follow from its race
	resp.RequireStatus(t, http.StatusOK)
	var characters []characterJSON
	resp.DecodeJSON(t, &characters)

	got := map[string][2]string{}
	for _, c := range characters {
		got[c.Name] = [2]string{*c.Race, *c.Faction}
	}
	want := map[string][2]string{
		"Aeliana":     {"Orc", "Horde"},
		"Bronzebeard": {"Dwarf", "Alliance"},
		"Cirrus":      {"Skyborne (High Order)", "Alliance"},
		"Dusk":        {"Skyborne (Windshaper)", "Horde"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected races and factions (-want +got):\n%s", diff)
	}
}

func TestListRoster_ReturnsNullRaceLevelAndFaction_WhenCharacterHasNone(t *testing.T) {
	// given a character that predates race and level
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, role, is_main, created_at)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Aeliana', 'Dawnsong', 'Emberreach', 'Priest', 'healer', true, '2026-09-01T12:00:00Z')`)

	// when I ask for the roster
	resp := srv.Get(t, "/api/roster")

	// then I expect race, level and faction to be null
	resp.RequireStatus(t, http.StatusOK)
	var characters []characterJSON
	resp.DecodeJSON(t, &characters)
	if len(characters) != 1 {
		t.Fatalf("expected one character, got %d", len(characters))
	}
	if got := characters[0]; got.Race != nil || got.Level != nil || got.Faction != nil {
		t.Errorf("expected null race, level and faction, got race=%v level=%v faction=%v", got.Race, got.Level, got.Faction)
	}
}

func TestCreateCharacter_StoresRaceLevelAndFaction_WhenValuesAreValid(t *testing.T) {
	// given I am logged in as an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I create a Troll at level 60
	body := createBody([]string{})
	body["race"] = "Troll"
	body["level"] = 60
	resp := srv.Post(t, "/api/roster", body)

	// then I expect a 201 carrying the race, level and derived faction
	resp.RequireStatus(t, http.StatusCreated)
	var created characterJSON
	resp.DecodeJSON(t, &created)
	if created.Race == nil || *created.Race != "Troll" || created.Level == nil || *created.Level != 60 || created.Faction == nil || *created.Faction != "Horde" {
		t.Errorf("expected Troll, 60, Horde, got race=%v level=%v faction=%v", created.Race, created.Level, created.Faction)
	}
}

func TestCreateCharacter_ReturnsBadRequest_WhenRaceIsNotPlayable(t *testing.T) {
	// given I am logged in as an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I create a character with a race that is not playable
	body := createBody([]string{})
	body["race"] = "Murloc"
	resp := srv.Post(t, "/api/roster", body)

	// then I expect a 400 naming the race field and nothing stored
	resp.RequireStatus(t, http.StatusBadRequest)
	requireErrorBody(t, resp, wantRaceError)
	if got := countCharacters(t, db); got != 0 {
		t.Errorf("expected no character to be stored, found %d", got)
	}
}

func TestCreateCharacter_ReturnsBadRequest_WhenLevelIsBelowOne(t *testing.T) {
	// given I am logged in as an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I create a character at level 0
	body := createBody([]string{})
	body["level"] = 0
	resp := srv.Post(t, "/api/roster", body)

	// then I expect a 400 naming the level field
	resp.RequireStatus(t, http.StatusBadRequest)
	requireErrorBody(t, resp, "level: must be between 1 and 60")
}

func TestCreateCharacter_ReturnsBadRequest_WhenLevelIsAboveTheCap(t *testing.T) {
	// given I am logged in as an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I create a character at level 61
	body := createBody([]string{})
	body["level"] = 61
	resp := srv.Post(t, "/api/roster", body)

	// then I expect a 400 naming the level field
	resp.RequireStatus(t, http.StatusBadRequest)
	requireErrorBody(t, resp, "level: must be between 1 and 60")
}

func TestUpdateCharacter_SetsRaceAndLevel_WhenValuesAreValid(t *testing.T) {
	// given a character with no race or level
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, role, is_main, created_at)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Aeliana', 'Dawnsong', 'Emberreach', 'Priest', 'healer', true, '2026-09-01T12:00:00Z')`)

	// when I set a Night Elf at level 58
	resp := srv.Patch(t, "/api/roster/11111111-1111-1111-1111-111111111111", map[string]any{"race": "Night Elf", "level": 58})

	// then I expect a 200 carrying the race, level and derived faction
	resp.RequireStatus(t, http.StatusOK)
	var updated characterJSON
	resp.DecodeJSON(t, &updated)
	if updated.Race == nil || *updated.Race != "Night Elf" || updated.Level == nil || *updated.Level != 58 || updated.Faction == nil || *updated.Faction != "Alliance" {
		t.Errorf("expected Night Elf, 58, Alliance, got race=%v level=%v faction=%v", updated.Race, updated.Level, updated.Faction)
	}
}

func TestUpdateCharacter_ReturnsBadRequest_WhenRaceIsNotPlayable(t *testing.T) {
	// given an existing character
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, role, is_main, created_at)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Aeliana', 'Dawnsong', 'Emberreach', 'Priest', 'healer', true, '2026-09-01T12:00:00Z')`)

	// when I set a race that is not playable
	resp := srv.Patch(t, "/api/roster/11111111-1111-1111-1111-111111111111", map[string]any{"race": "Worgen"})

	// then I expect a 400 naming the race field
	resp.RequireStatus(t, http.StatusBadRequest)
	requireErrorBody(t, resp, wantRaceError)
}
