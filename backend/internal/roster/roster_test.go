package roster_test

import (
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
	IsMain        bool    `json:"isMain"`
	RaidTeam      *string `json:"raidTeam"`
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
