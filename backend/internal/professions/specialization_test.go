package professions_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jmoiron/sqlx"

	"github.com/cageymage/fuzion/backend/internal/testutil"
)

const (
	zephyrionBlacksmithing = "a1111111-1111-1111-1111-111111111111"
	aelianaBlacksmithing   = "a2222222-2222-2222-2222-222222222222"
	aelianaAlchemy         = "a3333333-3333-3333-3333-333333333333"
	terrorskullEngineering = "b1111111-1111-1111-1111-111111111111"
)

type specializedProfessionJSON struct {
	ID                   string        `json:"id"`
	Profession           string        `json:"profession"`
	SkillLevel           int           `json:"skillLevel"`
	Specialization       *string       `json:"specialization"`
	SpecializationSource *string       `json:"specializationSource"`
	Character            characterJSON `json:"character"`
}

type storedSpecialization struct {
	Specialization *string `db:"specialization"`
	Source         *string `db:"specialization_source"`
}

func ptr(s string) *string { return &s }

func specializationOf(t *testing.T, db *sqlx.DB, professionID string) storedSpecialization {
	t.Helper()
	var row storedSpecialization
	if err := db.Get(&row, `SELECT specialization, specialization_source FROM professions WHERE id = $1`, professionID); err != nil {
		t.Fatalf("select specialization: %v", err)
	}
	return row
}

func setSpecialization(db *sqlx.DB, professionID, specialization, source string) {
	db.MustExec(`UPDATE professions SET specialization = $2, specialization_source = $3 WHERE id = $1`, professionID, specialization, source)
}

func requireError(t *testing.T, resp testutil.Response, status int, want string) {
	t.Helper()
	resp.RequireStatus(t, status)
	var body map[string]string
	resp.DecodeJSON(t, &body)
	if diff := cmp.Diff(map[string]string{"error": want}, body); diff != "" {
		t.Errorf("unexpected body (-want +got):\n%s", diff)
	}
}

func TestSetSpecialization_StoresAManualValue_WhenTheOwnerPicksOne(t *testing.T) {
	// given I own Aeliana, a blacksmith
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedOwnedCharacter(t, db, srv)

	// when I set her Blacksmithing specialization
	resp := srv.Patch(t, "/api/professions/"+aelianaBlacksmithing+"/specialization", map[string]any{"specialization": "Weaponsmith"})

	// then I expect the profession back with it, stored as a manual entry
	resp.RequireStatus(t, http.StatusOK)
	var got specializedProfessionJSON
	resp.DecodeJSON(t, &got)
	want := specializedProfessionJSON{ID: aelianaBlacksmithing, Profession: "Blacksmithing", SkillLevel: 225, Specialization: ptr("Weaponsmith"), SpecializationSource: ptr("manual"), Character: aeliana}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected profession (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(storedSpecialization{ptr("Weaponsmith"), ptr("manual")}, specializationOf(t, db, aelianaBlacksmithing)); diff != "" {
		t.Errorf("unexpected stored specialization (-want +got):\n%s", diff)
	}
}

func TestSetSpecialization_ClearsTheManualValue_WhenTheOwnerSendsNull(t *testing.T) {
	// given I own Aeliana, whose Weaponsmith specialization was set by hand
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedOwnedCharacter(t, db, srv)
	setSpecialization(db, aelianaBlacksmithing, "Weaponsmith", "manual")

	// when I clear it
	resp := srv.Patch(t, "/api/professions/"+aelianaBlacksmithing+"/specialization", map[string]any{"specialization": nil})

	// then I expect no specialization in the response or the database
	resp.RequireStatus(t, http.StatusOK)
	var got specializedProfessionJSON
	resp.DecodeJSON(t, &got)
	want := specializedProfessionJSON{ID: aelianaBlacksmithing, Profession: "Blacksmithing", SkillLevel: 225, Character: aeliana}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected profession (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(storedSpecialization{}, specializationOf(t, db, aelianaBlacksmithing)); diff != "" {
		t.Errorf("unexpected stored specialization (-want +got):\n%s", diff)
	}
}

func TestSetSpecialization_StoresTheValue_WhenAnOfficerSetsAnotherUsersCharacter(t *testing.T) {
	// given an officer who does not own Zephyrion
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(seedCharacters)
	db.MustExec(seedProfessions)
	srv.LoginAs(t, "80351110224678912", "thundermane", testutil.AsOfficer())

	// when the officer sets Zephyrion's Blacksmithing specialization
	resp := srv.Patch(t, "/api/professions/"+zephyrionBlacksmithing+"/specialization", map[string]any{"specialization": "Armorsmith"})

	// then I expect it stored as a manual entry
	resp.RequireStatus(t, http.StatusOK)
	if diff := cmp.Diff(storedSpecialization{ptr("Armorsmith"), ptr("manual")}, specializationOf(t, db, zephyrionBlacksmithing)); diff != "" {
		t.Errorf("unexpected stored specialization (-want +got):\n%s", diff)
	}
}

func TestSetSpecialization_ReturnsForbidden_WhenAMemberSetsAnotherUsersCharacter(t *testing.T) {
	// given I own Aeliana but not Zephyrion
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedOwnedCharacter(t, db, srv)

	// when I set Zephyrion's Blacksmithing specialization
	resp := srv.Patch(t, "/api/professions/"+zephyrionBlacksmithing+"/specialization", map[string]any{"specialization": "Armorsmith"})

	// then I expect a 403 and nothing stored
	requireError(t, resp, http.StatusForbidden, "character is not yours")
	if diff := cmp.Diff(storedSpecialization{}, specializationOf(t, db, zephyrionBlacksmithing)); diff != "" {
		t.Errorf("unexpected stored specialization (-want +got):\n%s", diff)
	}
}

func TestSetSpecialization_ReturnsUnauthorized_WhenLoggedOut(t *testing.T) {
	// given a blacksmith and nobody logged in
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(seedCharacters)
	db.MustExec(seedProfessions)

	// when I set a specialization
	resp := srv.Patch(t, "/api/professions/"+zephyrionBlacksmithing+"/specialization", map[string]any{"specialization": "Armorsmith"})

	// then I expect a 401
	requireError(t, resp, http.StatusUnauthorized, "login required")
}

func TestSetSpecialization_ReturnsNotFound_WhenTheProfessionDoesNotExist(t *testing.T) {
	// given I own Aeliana
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedOwnedCharacter(t, db, srv)

	// when I set the specialization of an unknown profession id
	resp := srv.Patch(t, "/api/professions/a9999999-9999-9999-9999-999999999999/specialization", map[string]any{"specialization": "Armorsmith"})

	// then I expect a 404
	requireError(t, resp, http.StatusNotFound, "profession not found")
}

func TestSetSpecialization_ReturnsBadRequest_WhenItBelongsToAnotherProfession(t *testing.T) {
	// given I own Aeliana, a blacksmith
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedOwnedCharacter(t, db, srv)

	// when I set a Leatherworking specialization on Blacksmithing
	resp := srv.Patch(t, "/api/professions/"+aelianaBlacksmithing+"/specialization", map[string]any{"specialization": "Tribal"})

	// then I expect a 400 and nothing stored
	requireError(t, resp, http.StatusBadRequest, "specialization: must be one of Armorsmith, Weaponsmith")
	if diff := cmp.Diff(storedSpecialization{}, specializationOf(t, db, aelianaBlacksmithing)); diff != "" {
		t.Errorf("unexpected stored specialization (-want +got):\n%s", diff)
	}
}

func TestSetSpecialization_ReturnsBadRequest_WhenTheProfessionHasNoSpecializations(t *testing.T) {
	// given I own Aeliana, an alchemist
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedOwnedCharacter(t, db, srv)

	// when I set a specialization on Alchemy
	resp := srv.Patch(t, "/api/professions/"+aelianaAlchemy+"/specialization", map[string]any{"specialization": "Weaponsmith"})

	// then I expect a 400
	requireError(t, resp, http.StatusBadRequest, "specialization: Alchemy has none to choose from")
}

func TestSetSpecialization_ReturnsBadRequest_WhenTheFieldIsMissing(t *testing.T) {
	// given I own Aeliana, whose Weaponsmith specialization was set by hand
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedOwnedCharacter(t, db, srv)
	setSpecialization(db, aelianaBlacksmithing, "Weaponsmith", "manual")

	// when I send a body without a specialization
	resp := srv.Patch(t, "/api/professions/"+aelianaBlacksmithing+"/specialization", map[string]any{})

	// then I expect a 400 and the stored value kept
	requireError(t, resp, http.StatusBadRequest, "specialization: must be a string or null")
	if diff := cmp.Diff(storedSpecialization{ptr("Weaponsmith"), ptr("manual")}, specializationOf(t, db, aelianaBlacksmithing)); diff != "" {
		t.Errorf("unexpected stored specialization (-want +got):\n%s", diff)
	}
}

func TestSetSpecialization_ReturnsConflict_WhenTheValueComesFromBlizzard(t *testing.T) {
	// given I own Aeliana, whose specialization Blizzard reported
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedOwnedCharacter(t, db, srv)
	setSpecialization(db, aelianaBlacksmithing, "Weaponsmith", "blizzard")

	// when I set a different one
	resp := srv.Patch(t, "/api/professions/"+aelianaBlacksmithing+"/specialization", map[string]any{"specialization": "Armorsmith"})

	// then I expect a 409 and Blizzard's value kept
	requireError(t, resp, http.StatusConflict, "specialization comes from Battle.net")
	if diff := cmp.Diff(storedSpecialization{ptr("Weaponsmith"), ptr("blizzard")}, specializationOf(t, db, aelianaBlacksmithing)); diff != "" {
		t.Errorf("unexpected stored specialization (-want +got):\n%s", diff)
	}
}

func TestListProfessions_ReturnsSpecializations_WhenSomeAreSet(t *testing.T) {
	// given two blacksmiths of whom only Zephyrion has a specialization
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(seedCharacters)
	db.MustExec(seedProfessions)
	setSpecialization(db, zephyrionBlacksmithing, "Weaponsmith", "manual")

	// when I ask for the blacksmiths
	resp := srv.Get(t, "/api/professions?profession=Blacksmithing")

	// then I expect Zephyrion's specialization and none for Aeliana
	resp.RequireStatus(t, http.StatusOK)
	var got []specializedProfessionJSON
	resp.DecodeJSON(t, &got)
	want := []specializedProfessionJSON{
		{ID: zephyrionBlacksmithing, Profession: "Blacksmithing", SkillLevel: 300, Specialization: ptr("Weaponsmith"), SpecializationSource: ptr("manual"), Character: zephyrion},
		{ID: aelianaBlacksmithing, Profession: "Blacksmithing", SkillLevel: 225, Character: aeliana},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected professions (-want +got):\n%s", diff)
	}
}

func TestUpdateProfession_ClearsTheSpecialization_WhenTheProfessionChanges(t *testing.T) {
	// given an officer and a Weaponsmith
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(seedCharacters)
	db.MustExec(seedProfessions)
	setSpecialization(db, zephyrionBlacksmithing, "Weaponsmith", "manual")
	srv.LoginAs(t, "80351110224678912", "thundermane", testutil.AsOfficer())

	// when the officer changes Blacksmithing to Mining
	resp := srv.Patch(t, "/api/professions/"+zephyrionBlacksmithing, map[string]any{"profession": "Mining"})

	// then I expect no specialization left
	resp.RequireStatus(t, http.StatusOK)
	if diff := cmp.Diff(storedSpecialization{}, specializationOf(t, db, zephyrionBlacksmithing)); diff != "" {
		t.Errorf("unexpected stored specialization (-want +got):\n%s", diff)
	}
}

func TestUpdateProfession_KeepsTheSpecialization_WhenOnlyTheSkillLevelChanges(t *testing.T) {
	// given an officer and a Weaponsmith
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(seedCharacters)
	db.MustExec(seedProfessions)
	setSpecialization(db, zephyrionBlacksmithing, "Weaponsmith", "manual")
	srv.LoginAs(t, "80351110224678912", "thundermane", testutil.AsOfficer())

	// when the officer changes the skill level
	resp := srv.Patch(t, "/api/professions/"+zephyrionBlacksmithing, map[string]any{"skillLevel": 290})

	// then I expect the specialization kept
	resp.RequireStatus(t, http.StatusOK)
	if diff := cmp.Diff(storedSpecialization{ptr("Weaponsmith"), ptr("manual")}, specializationOf(t, db, zephyrionBlacksmithing)); diff != "" {
		t.Errorf("unexpected stored specialization (-want +got):\n%s", diff)
	}
}

// newFakeBlizzardEngineer serves Terrorskull as an engineer; an empty specialization is
// left out of the response, as Blizzard does.
func newFakeBlizzardEngineer(t *testing.T, specialization string) *httptest.Server {
	t.Helper()

	engineering := map[string]any{
		"profession": map[string]any{"name": "Engineering"},
		"tiers":      []map[string]any{{"skill_points": 268, "known_recipes": []any{}}},
	}
	if specialization != "" {
		engineering["specialization"] = map[string]any{"name": specialization}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"access_token": "app-token"})
	})
	mux.HandleFunc("GET /profile/wow/character/terrorskull-gloomstep/professions", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"primaries": []any{engineering}, "secondaries": []any{}})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func seedEngineer(db *sqlx.DB) {
	db.MustExec(syncCharacters)
	db.MustExec(`
		INSERT INTO professions (id, character_id, profession, skill_level)
		VALUES ($1, '11111111-1111-1111-1111-111111111111', 'Engineering', 250)`, terrorskullEngineering)
}

func TestSyncBlizzardProfessions_StoresTheSpecialization_WhenBlizzardReportsOne(t *testing.T) {
	// given an engineer without a specialization, whom Blizzard reports as a Gnomish Engineer
	db := testutil.DB(t)
	seedEngineer(db)
	srv := newFakeBlizzardEngineer(t, "Gnomish Engineer")

	// when the professions sync runs
	err := runProfessionsSync(db, srv)

	// then I expect Gnomish stored as a Blizzard value
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if diff := cmp.Diff(storedSpecialization{ptr("Gnomish"), ptr("blizzard")}, specializationOf(t, db, terrorskullEngineering)); diff != "" {
		t.Errorf("unexpected stored specialization (-want +got):\n%s", diff)
	}
}

func TestSyncBlizzardProfessions_StoresTheSpecialization_WhenTheProfessionIsNew(t *testing.T) {
	// given a character with no professions, whom Blizzard reports as a Goblin Engineer
	db := testutil.DB(t)
	db.MustExec(syncCharacters)
	srv := newFakeBlizzardEngineer(t, "Goblin Engineer")

	// when the professions sync runs
	err := runProfessionsSync(db, srv)

	// then I expect Goblin stored as a Blizzard value on the new row
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	var rows []storedSpecialization
	if err := db.Select(&rows, `SELECT specialization, specialization_source FROM professions`); err != nil {
		t.Fatalf("select specializations: %v", err)
	}
	if diff := cmp.Diff([]storedSpecialization{{ptr("Goblin"), ptr("blizzard")}}, rows); diff != "" {
		t.Errorf("unexpected stored specializations (-want +got):\n%s", diff)
	}
}

func TestSyncBlizzardProfessions_OverwritesTheManualSpecialization_WhenBlizzardReportsOne(t *testing.T) {
	// given a Goblin specialization set by hand on a character Blizzard reports as a Gnomish Engineer
	db := testutil.DB(t)
	seedEngineer(db)
	setSpecialization(db, terrorskullEngineering, "Goblin", "manual")
	srv := newFakeBlizzardEngineer(t, "Gnomish Engineer")

	// when the professions sync runs
	err := runProfessionsSync(db, srv)

	// then I expect Blizzard's value to replace it
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if diff := cmp.Diff(storedSpecialization{ptr("Gnomish"), ptr("blizzard")}, specializationOf(t, db, terrorskullEngineering)); diff != "" {
		t.Errorf("unexpected stored specialization (-want +got):\n%s", diff)
	}
}

func TestSyncBlizzardProfessions_KeepsTheManualSpecialization_WhenBlizzardReportsNone(t *testing.T) {
	// given a Goblin specialization set by hand on a character Blizzard reports without one
	db := testutil.DB(t)
	seedEngineer(db)
	setSpecialization(db, terrorskullEngineering, "Goblin", "manual")
	srv := newFakeBlizzardEngineer(t, "")

	// when the professions sync runs
	err := runProfessionsSync(db, srv)

	// then I expect the manual value untouched
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if diff := cmp.Diff(storedSpecialization{ptr("Goblin"), ptr("manual")}, specializationOf(t, db, terrorskullEngineering)); diff != "" {
		t.Errorf("unexpected stored specialization (-want +got):\n%s", diff)
	}
}

func TestSyncBlizzardProfessions_ClearsTheBlizzardSpecialization_WhenBlizzardStopsReportingIt(t *testing.T) {
	// given a specialization from an earlier sync that Blizzard does not report any more
	db := testutil.DB(t)
	seedEngineer(db)
	setSpecialization(db, terrorskullEngineering, "Goblin", "blizzard")
	srv := newFakeBlizzardEngineer(t, "")

	// when the professions sync runs
	err := runProfessionsSync(db, srv)

	// then I expect it cleared
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if diff := cmp.Diff(storedSpecialization{}, specializationOf(t, db, terrorskullEngineering)); diff != "" {
		t.Errorf("unexpected stored specialization (-want +got):\n%s", diff)
	}
}

func TestSyncBlizzardProfessions_KeepsTheManualSpecialization_WhenBlizzardReportsAnUnknownOne(t *testing.T) {
	// given a Goblin specialization set by hand and a Blizzard reporting a name the site does not know
	db := testutil.DB(t)
	seedEngineer(db)
	setSpecialization(db, terrorskullEngineering, "Goblin", "manual")
	srv := newFakeBlizzardEngineer(t, "Titan Engineer")

	// when the professions sync runs
	err := runProfessionsSync(db, srv)

	// then I expect no error and the manual value untouched
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if diff := cmp.Diff(storedSpecialization{ptr("Goblin"), ptr("manual")}, specializationOf(t, db, terrorskullEngineering)); diff != "" {
		t.Errorf("unexpected stored specialization (-want +got):\n%s", diff)
	}
}
