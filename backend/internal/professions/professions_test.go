package professions_test

import (
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
