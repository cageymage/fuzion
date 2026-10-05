package professions_test

import (
	"net/http"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/cageymage/fuzion/backend/internal/testutil"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.Run(m))
}

type characterJSON struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	SecondaryName string `json:"secondaryName"`
	Class         string `json:"class"`
}

type professionJSON struct {
	ID         string        `json:"id"`
	Profession string        `json:"profession"`
	SkillLevel int           `json:"skillLevel"`
	Character  characterJSON `json:"character"`
}

const seedCharacters = `
	INSERT INTO characters (id, name, secondary_name, realm, class, role)
	VALUES
		('11111111-1111-1111-1111-111111111111', 'Aeliana', 'Dawnsong', 'Emberreach', 'Priest', 'healer'),
		('22222222-2222-2222-2222-222222222222', 'Zephyrion', 'Ironhide', 'Emberreach', 'Warrior', 'tank')`

const seedProfessions = `
	INSERT INTO professions (id, character_id, profession, skill_level)
	VALUES
		('a1111111-1111-1111-1111-111111111111', '22222222-2222-2222-2222-222222222222', 'Blacksmithing', 300),
		('a2222222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111111', 'Blacksmithing', 225),
		('a3333333-3333-3333-3333-333333333333', '11111111-1111-1111-1111-111111111111', 'Alchemy', 300),
		('a4444444-4444-4444-4444-444444444444', '22222222-2222-2222-2222-222222222222', 'Alchemy', 300)`

var (
	aeliana = characterJSON{
		ID:            "11111111-1111-1111-1111-111111111111",
		Name:          "Aeliana",
		SecondaryName: "Dawnsong",
		Class:         "Priest",
	}
	zephyrion = characterJSON{
		ID:            "22222222-2222-2222-2222-222222222222",
		Name:          "Zephyrion",
		SecondaryName: "Ironhide",
		Class:         "Warrior",
	}
)

func TestListProfessions_ReturnsRowsWithCharacterDetails(t *testing.T) {
	// given characters who each know professions at different skill levels
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(seedCharacters)
	db.MustExec(seedProfessions)

	// when I ask for the professions
	resp := srv.Get(t, "/api/professions")

	// then I expect every row with its character, sorted by profession, then highest skill, then character name
	resp.RequireStatus(t, 200)
	var professions []professionJSON
	resp.DecodeJSON(t, &professions)

	want := []professionJSON{
		{ID: "a3333333-3333-3333-3333-333333333333", Profession: "Alchemy", SkillLevel: 300, Character: aeliana},
		{ID: "a4444444-4444-4444-4444-444444444444", Profession: "Alchemy", SkillLevel: 300, Character: zephyrion},
		{ID: "a1111111-1111-1111-1111-111111111111", Profession: "Blacksmithing", SkillLevel: 300, Character: zephyrion},
		{ID: "a2222222-2222-2222-2222-222222222222", Profession: "Blacksmithing", SkillLevel: 225, Character: aeliana},
	}
	if diff := cmp.Diff(want, professions); diff != "" {
		t.Errorf("unexpected professions (-want +got):\n%s", diff)
	}
}

func TestListProfessions_ReturnsOnlyMatchingProfession_WhenFilterIsGiven(t *testing.T) {
	// given characters who know both Alchemy and Blacksmithing
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(seedCharacters)
	db.MustExec(seedProfessions)

	// when I filter by profession using different casing than is stored
	resp := srv.Get(t, "/api/professions?profession=blacksmithing")

	// then I expect only the Blacksmithing rows
	resp.RequireStatus(t, 200)
	var professions []professionJSON
	resp.DecodeJSON(t, &professions)

	want := []professionJSON{
		{ID: "a1111111-1111-1111-1111-111111111111", Profession: "Blacksmithing", SkillLevel: 300, Character: zephyrion},
		{ID: "a2222222-2222-2222-2222-222222222222", Profession: "Blacksmithing", SkillLevel: 225, Character: aeliana},
	}
	if diff := cmp.Diff(want, professions); diff != "" {
		t.Errorf("unexpected professions (-want +got):\n%s", diff)
	}
}

func TestListProfessions_ReturnsEmptyArray_WhenFilterMatchesNothing(t *testing.T) {
	// given characters who know only Alchemy and Blacksmithing
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(seedCharacters)
	db.MustExec(seedProfessions)

	// when I filter by a profession nobody has
	resp := srv.Get(t, "/api/professions?profession=Tailoring")

	// then I expect a 200 with an empty JSON array
	resp.RequireStatus(t, 200)
	var professions []professionJSON
	resp.DecodeJSON(t, &professions)

	if diff := cmp.Diff([]professionJSON{}, professions); diff != "" {
		t.Errorf("unexpected professions (-want +got):\n%s", diff)
	}
}

func TestListProfessions_OmitsProfessions_WhenTheirCharacterIsDeleted(t *testing.T) {
	// given professions for two characters, after one of them is deleted
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(seedCharacters)
	db.MustExec(seedProfessions)
	db.MustExec(`DELETE FROM characters WHERE id = '22222222-2222-2222-2222-222222222222'`)

	// when I ask for the professions
	resp := srv.Get(t, "/api/professions")

	// then I expect only the surviving character's professions
	resp.RequireStatus(t, 200)
	var professions []professionJSON
	resp.DecodeJSON(t, &professions)

	want := []professionJSON{
		{ID: "a3333333-3333-3333-3333-333333333333", Profession: "Alchemy", SkillLevel: 300, Character: aeliana},
		{ID: "a2222222-2222-2222-2222-222222222222", Profession: "Blacksmithing", SkillLevel: 225, Character: aeliana},
	}
	if diff := cmp.Diff(want, professions); diff != "" {
		t.Errorf("unexpected professions (-want +got):\n%s", diff)
	}
}

func TestCreateProfession_ReturnsCreated_WhenOfficerSubmitsValidProfession(t *testing.T) {
	// given I am an officer and a character exists
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	db.MustExec(seedCharacters)

	// when I add a profession to the character
	resp := srv.Post(t, "/api/professions", map[string]any{
		"characterId": aeliana.ID,
		"profession":  "  Alchemy ",
		"skillLevel":  225,
	})

	// then I expect a 201 with the row in the same shape the list returns
	resp.RequireStatus(t, http.StatusCreated)
	var created professionJSON
	resp.DecodeJSON(t, &created)

	if created.ID == "" {
		t.Error("expected the created profession to have an id")
	}
	want := professionJSON{ID: created.ID, Profession: "Alchemy", SkillLevel: 225, Character: aeliana}
	if diff := cmp.Diff(want, created); diff != "" {
		t.Errorf("unexpected created profession (-want +got):\n%s", diff)
	}
}

func TestCreateProfession_ReturnsBadRequest_WhenCharacterDoesNotExist(t *testing.T) {
	// given I am an officer and no characters exist
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I add a profession to a character id that does not exist
	resp := srv.Post(t, "/api/professions", map[string]any{
		"characterId": "99999999-9999-9999-9999-999999999999",
		"profession":  "Alchemy",
		"skillLevel":  225,
	})

	// then I expect a 400 naming the characterId field
	resp.RequireStatus(t, http.StatusBadRequest)
	var body map[string]string
	resp.DecodeJSON(t, &body)
	if diff := cmp.Diff(map[string]string{"error": "characterId: must be an existing character"}, body); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
}

func TestCreateProfession_ReturnsBadRequest_WhenSkillLevelIsNegative(t *testing.T) {
	// given I am an officer and a character exists
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	db.MustExec(seedCharacters)

	// when I add a profession with a negative skill level
	resp := srv.Post(t, "/api/professions", map[string]any{
		"characterId": aeliana.ID,
		"profession":  "Alchemy",
		"skillLevel":  -1,
	})

	// then I expect a 400 naming the skillLevel field
	resp.RequireStatus(t, http.StatusBadRequest)
	var body map[string]string
	resp.DecodeJSON(t, &body)
	if diff := cmp.Diff(map[string]string{"error": "skillLevel: must be 0 or more"}, body); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
}

func TestCreateProfession_ReturnsBadRequest_WhenProfessionIsBlank(t *testing.T) {
	// given I am an officer and a character exists
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	db.MustExec(seedCharacters)

	// when I add a profession whose name is only whitespace
	resp := srv.Post(t, "/api/professions", map[string]any{
		"characterId": aeliana.ID,
		"profession":  "   ",
		"skillLevel":  225,
	})

	// then I expect a 400 naming the profession field
	resp.RequireStatus(t, http.StatusBadRequest)
	var body map[string]string
	resp.DecodeJSON(t, &body)
	if diff := cmp.Diff(map[string]string{"error": "profession: must not be empty"}, body); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
}

func TestCreateProfession_ReturnsConflict_WhenCharacterAlreadyHasThatProfession(t *testing.T) {
	// given I am an officer and a character who already has Alchemy
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	db.MustExec(seedCharacters)
	db.MustExec(seedProfessions)

	// when I add Alchemy to that character again
	resp := srv.Post(t, "/api/professions", map[string]any{
		"characterId": aeliana.ID,
		"profession":  "Alchemy",
		"skillLevel":  150,
	})

	// then I expect a 409
	resp.RequireStatus(t, http.StatusConflict)
}

func TestCreateProfession_ReturnsUnauthorized_WhenCallerIsAnonymous(t *testing.T) {
	// given nobody is logged in
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(seedCharacters)

	// when I add a profession
	resp := srv.Post(t, "/api/professions", map[string]any{
		"characterId": aeliana.ID,
		"profession":  "Alchemy",
		"skillLevel":  225,
	})

	// then I expect a 401
	resp.RequireStatus(t, http.StatusUnauthorized)
}

func TestCreateProfession_ReturnsForbidden_WhenCallerIsNotAnOfficer(t *testing.T) {
	// given I am logged in as a regular member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "member-1", "Member")
	db.MustExec(seedCharacters)

	// when I add a profession
	resp := srv.Post(t, "/api/professions", map[string]any{
		"characterId": aeliana.ID,
		"profession":  "Alchemy",
		"skillLevel":  225,
	})

	// then I expect a 403
	resp.RequireStatus(t, http.StatusForbidden)
}

func TestUpdateProfession_ChangesOnlyProvidedFields(t *testing.T) {
	// given I am an officer and a character with a Blacksmithing profession at 225
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	db.MustExec(seedCharacters)
	db.MustExec(seedProfessions)

	// when I update only the skill level
	resp := srv.Patch(t, "/api/professions/a2222222-2222-2222-2222-222222222222", map[string]any{"skillLevel": 300})

	// then I expect a 200 with the new skill level and the profession name unchanged
	resp.RequireStatus(t, http.StatusOK)
	var updated professionJSON
	resp.DecodeJSON(t, &updated)

	want := professionJSON{ID: "a2222222-2222-2222-2222-222222222222", Profession: "Blacksmithing", SkillLevel: 300, Character: aeliana}
	if diff := cmp.Diff(want, updated); diff != "" {
		t.Errorf("unexpected updated profession (-want +got):\n%s", diff)
	}
}

func TestUpdateProfession_ReturnsNotFound_WhenProfessionDoesNotExist(t *testing.T) {
	// given I am an officer and no professions exist
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I update a profession id that does not exist
	resp := srv.Patch(t, "/api/professions/99999999-9999-9999-9999-999999999999", map[string]any{"skillLevel": 300})

	// then I expect a 404
	resp.RequireStatus(t, http.StatusNotFound)
}

func TestUpdateProfession_ReturnsNotFound_WhenIdIsMalformed(t *testing.T) {
	// given I am an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I update a profession using an id that is not a UUID
	resp := srv.Patch(t, "/api/professions/not-a-uuid", map[string]any{"skillLevel": 300})

	// then I expect a 404
	resp.RequireStatus(t, http.StatusNotFound)
}

func TestDeleteProfession_ReturnsNoContent_WhenOfficerDeletes(t *testing.T) {
	// given I am an officer and professions exist
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	db.MustExec(seedCharacters)
	db.MustExec(seedProfessions)

	// when I delete one profession
	resp := srv.Delete(t, "/api/professions/a2222222-2222-2222-2222-222222222222")

	// then I expect a 204 and that row gone from the list
	resp.RequireStatus(t, http.StatusNoContent)
	list := srv.Get(t, "/api/professions")
	var remaining []professionJSON
	list.DecodeJSON(t, &remaining)
	want := []professionJSON{
		{ID: "a3333333-3333-3333-3333-333333333333", Profession: "Alchemy", SkillLevel: 300, Character: aeliana},
		{ID: "a4444444-4444-4444-4444-444444444444", Profession: "Alchemy", SkillLevel: 300, Character: zephyrion},
		{ID: "a1111111-1111-1111-1111-111111111111", Profession: "Blacksmithing", SkillLevel: 300, Character: zephyrion},
	}
	if diff := cmp.Diff(want, remaining); diff != "" {
		t.Errorf("unexpected remaining professions (-want +got):\n%s", diff)
	}
}

func TestDeleteProfession_ReturnsForbidden_WhenCallerIsNotAnOfficer(t *testing.T) {
	// given I am logged in as a regular member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "member-1", "Member")
	db.MustExec(seedCharacters)
	db.MustExec(seedProfessions)

	// when I delete a profession
	resp := srv.Delete(t, "/api/professions/a2222222-2222-2222-2222-222222222222")

	// then I expect a 403
	resp.RequireStatus(t, http.StatusForbidden)
}
