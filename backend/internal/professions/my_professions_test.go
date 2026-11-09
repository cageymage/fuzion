package professions_test

import (
	"net/http"
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

// seedOwnedCharacter gives me Aeliana and leaves me logged in.
func seedOwnedCharacter(t *testing.T, db *sqlx.DB, srv *testutil.Server) {
	t.Helper()
	db.MustExec(seedCharacters)
	db.MustExec(seedProfessions)
	me := srv.LoginAs(t, "80351110224678912", "thundermane")
	db.MustExec(`UPDATE characters SET user_id = $1 WHERE id = $2`, me.ID, aeliana.ID)
}

// The professions sync owns skill levels, so members have no way to write them.

func TestCreateMyProfession_ReturnsNotFound_WhenAMemberAddsAProfession(t *testing.T) {
	// given I own Aeliana
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedOwnedCharacter(t, db, srv)

	// when I try to add Tailoring to Aeliana
	resp := srv.Post(t, "/api/me/characters/"+aeliana.ID+"/professions", map[string]any{"profession": "Tailoring", "skillLevel": 150})

	// then I expect a 404 and the professions unchanged
	resp.RequireStatus(t, http.StatusNotFound)
	if diff := cmp.Diff(seededProfessions, storedProfessions(t, db)); diff != "" {
		t.Errorf("unexpected professions (-want +got):\n%s", diff)
	}
}

func TestUpdateMyProfession_ReturnsNotFound_WhenAMemberChangesASkillLevel(t *testing.T) {
	// given I own Aeliana and her Alchemy
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedOwnedCharacter(t, db, srv)

	// when I try to change her Alchemy skill
	resp := srv.Patch(t, "/api/me/professions/a3333333-3333-3333-3333-333333333333", map[string]any{"skillLevel": 1})

	// then I expect a 404 and the professions unchanged
	resp.RequireStatus(t, http.StatusNotFound)
	if diff := cmp.Diff(seededProfessions, storedProfessions(t, db)); diff != "" {
		t.Errorf("unexpected professions (-want +got):\n%s", diff)
	}
}

func TestDeleteMyProfession_ReturnsNotFound_WhenAMemberRemovesAProfession(t *testing.T) {
	// given I own Aeliana and her Alchemy
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	seedOwnedCharacter(t, db, srv)

	// when I try to remove her Alchemy
	resp := srv.Delete(t, "/api/me/professions/a3333333-3333-3333-3333-333333333333")

	// then I expect a 404 and the professions unchanged
	resp.RequireStatus(t, http.StatusNotFound)
	if diff := cmp.Diff(seededProfessions, storedProfessions(t, db)); diff != "" {
		t.Errorf("unexpected professions (-want +got):\n%s", diff)
	}
}
