package professions_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jmoiron/sqlx"

	"github.com/cageymage/fuzion/backend/internal/blizzard"
	"github.com/cageymage/fuzion/backend/internal/professions"
	"github.com/cageymage/fuzion/backend/internal/synclog"
	"github.com/cageymage/fuzion/backend/internal/testutil"
)

type fakeProfession struct {
	name  string
	skill int
}

// newFakeBlizzardProfessions serves /profile/wow/character/{slug}/professions for the
// slugs it was given and answers 404 for every other character.
func newFakeBlizzardProfessions(t *testing.T, bySlug map[string][]fakeProfession) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"access_token": "app-token"})
	})
	mux.HandleFunc("GET /profile/wow/character/{slug}/professions", func(w http.ResponseWriter, r *http.Request) {
		known, ok := bySlug[r.PathValue("slug")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		primaries := []map[string]any{}
		for _, p := range known {
			primaries = append(primaries, map[string]any{
				"profession": map[string]any{"name": p.name},
				"tiers":      []map[string]any{{"skill_points": p.skill}},
			})
		}
		json.NewEncoder(w).Encode(map[string]any{"primaries": primaries, "secondaries": []any{}})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func runProfessionsSync(db *sqlx.DB, srv *httptest.Server) error {
	client := blizzard.NewClient(blizzard.Config{
		ClientID: "id", ClientSecret: "secret", BaseURL: srv.URL, AuthURL: srv.URL, Namespace: "profile-forever-us",
	}, nil)
	return professions.NewSyncBlizzardProfessions(professions.NewRepo(db), client, synclog.NewRepo(db)).Run(context.Background())
}

type syncedProfession struct {
	Name       string `db:"name"`
	Profession string `db:"profession"`
	SkillLevel int    `db:"skill_level"`
	Source     string `db:"source"`
}

func syncedProfessions(t *testing.T, db *sqlx.DB) []syncedProfession {
	t.Helper()
	var rows []syncedProfession
	err := db.Select(&rows, `
		SELECT c.name, p.profession, p.skill_level, p.source
		FROM professions p JOIN characters c ON c.id = p.character_id
		ORDER BY c.name, p.profession`)
	if err != nil {
		t.Fatalf("select professions: %v", err)
	}
	return rows
}

type syncLogEntry struct {
	Source  string `db:"source"`
	Status  string `db:"status"`
	Message string `db:"message"`
}

func syncLogEntries(t *testing.T, db *sqlx.DB) []syncLogEntry {
	t.Helper()
	var rows []syncLogEntry
	if err := db.Select(&rows, `SELECT source, status, message FROM sync_log ORDER BY id`); err != nil {
		t.Fatalf("select sync_log: %v", err)
	}
	return rows
}

const syncCharacters = `
	INSERT INTO characters (id, name, secondary_name, realm, class, role, left_guild_at)
	VALUES
		('11111111-1111-1111-1111-111111111111', 'Terrorskull', 'Gloomstep', 'Emberreach', 'Rogue', 'dps', NULL),
		('22222222-2222-2222-2222-222222222222', 'Swamplörd', 'Atrocitus', 'Emberreach', 'Warrior', 'tank', NULL)`

func TestSyncBlizzardProfessions_UpsertsRows_ForCharactersWithProfessions(t *testing.T) {
	// given two roster characters and a Blizzard that knows both of their professions
	db := testutil.DB(t)
	db.MustExec(syncCharacters)
	srv := newFakeBlizzardProfessions(t, map[string][]fakeProfession{
		"terrorskull-gloomstep": {{"Leatherworking", 55}, {"Skinning", 98}},
		"swamplörd-atrocitus":   {{"Mining", 30}},
	})

	// when the professions sync runs
	err := runProfessionsSync(db, srv)

	// then I expect a row per profession marked as from Blizzard, and an ok sync log entry
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	want := []syncedProfession{
		{Name: "Swamplörd", Profession: "Mining", SkillLevel: 30, Source: "blizzard"},
		{Name: "Terrorskull", Profession: "Leatherworking", SkillLevel: 55, Source: "blizzard"},
		{Name: "Terrorskull", Profession: "Skinning", SkillLevel: 98, Source: "blizzard"},
	}
	if diff := cmp.Diff(want, syncedProfessions(t, db)); diff != "" {
		t.Errorf("unexpected professions (-want +got):\n%s", diff)
	}
	wantLog := []syncLogEntry{{Source: "blizzard-professions", Status: "ok", Message: "2 characters: 3 professions stored, 0 not found"}}
	if diff := cmp.Diff(wantLog, syncLogEntries(t, db)); diff != "" {
		t.Errorf("unexpected sync log (-want +got):\n%s", diff)
	}
}

func TestSyncBlizzardProfessions_OverwritesOfficerSkillLevel_WhenBlizzardReportsTheProfession(t *testing.T) {
	// given an officer-entered Skinning level of 10
	db := testutil.DB(t)
	db.MustExec(syncCharacters)
	db.MustExec(`
		INSERT INTO professions (id, character_id, profession, skill_level)
		VALUES ('a1111111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111111', 'Skinning', 10)`)
	srv := newFakeBlizzardProfessions(t, map[string][]fakeProfession{"terrorskull-gloomstep": {{"Skinning", 98}}})

	// when the professions sync runs
	err := runProfessionsSync(db, srv)

	// then I expect Blizzard's level to replace it
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	want := []syncedProfession{{Name: "Terrorskull", Profession: "Skinning", SkillLevel: 98, Source: "blizzard"}}
	if diff := cmp.Diff(want, syncedProfessions(t, db)); diff != "" {
		t.Errorf("unexpected professions (-want +got):\n%s", diff)
	}
}

func TestSyncBlizzardProfessions_KeepsOfficerRow_WhenBlizzardDoesNotReportTheProfession(t *testing.T) {
	// given an officer-entered Mining row that Blizzard does not list for the character
	db := testutil.DB(t)
	db.MustExec(syncCharacters)
	db.MustExec(`
		INSERT INTO professions (id, character_id, profession, skill_level)
		VALUES ('a1111111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111111', 'Mining', 40)`)
	srv := newFakeBlizzardProfessions(t, map[string][]fakeProfession{"terrorskull-gloomstep": {{"Skinning", 98}}})

	// when the professions sync runs
	err := runProfessionsSync(db, srv)

	// then I expect the officer's Mining row untouched next to the new Skinning row
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	want := []syncedProfession{
		{Name: "Terrorskull", Profession: "Mining", SkillLevel: 40, Source: "officer"},
		{Name: "Terrorskull", Profession: "Skinning", SkillLevel: 98, Source: "blizzard"},
	}
	if diff := cmp.Diff(want, syncedProfessions(t, db)); diff != "" {
		t.Errorf("unexpected professions (-want +got):\n%s", diff)
	}
}

func TestSyncBlizzardProfessions_SkipsCharacter_WhenBlizzardReturnsNotFound(t *testing.T) {
	// given two roster characters of whom Blizzard knows only one
	db := testutil.DB(t)
	db.MustExec(syncCharacters)
	srv := newFakeBlizzardProfessions(t, map[string][]fakeProfession{"swamplörd-atrocitus": {{"Mining", 30}}})

	// when the professions sync runs
	err := runProfessionsSync(db, srv)

	// then I expect no error, the known character stored, and the miss counted in the log
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	want := []syncedProfession{{Name: "Swamplörd", Profession: "Mining", SkillLevel: 30, Source: "blizzard"}}
	if diff := cmp.Diff(want, syncedProfessions(t, db)); diff != "" {
		t.Errorf("unexpected professions (-want +got):\n%s", diff)
	}
	wantLog := []syncLogEntry{{Source: "blizzard-professions", Status: "ok", Message: "2 characters: 1 professions stored, 1 not found"}}
	if diff := cmp.Diff(wantLog, syncLogEntries(t, db)); diff != "" {
		t.Errorf("unexpected sync log (-want +got):\n%s", diff)
	}
}

func TestSyncBlizzardProfessions_SkipsCharacters_WhoLeftTheGuildOrLackASurname(t *testing.T) {
	// given a character who left and one still carrying the Unset placeholder surname
	db := testutil.DB(t)
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, role, left_guild_at)
		VALUES
			('11111111-1111-1111-1111-111111111111', 'Quitter', 'Gone', 'Emberreach', 'Mage', 'dps', now()),
			('22222222-2222-2222-2222-222222222222', 'Nameless', 'Unset', 'Emberreach', 'Mage', 'dps', NULL)`)
	srv := newFakeBlizzardProfessions(t, map[string][]fakeProfession{
		"quitter-gone":   {{"Mining", 30}},
		"nameless-unset": {{"Mining", 30}},
	})

	// when the professions sync runs
	err := runProfessionsSync(db, srv)

	// then I expect nothing stored, and neither character asked about
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if diff := cmp.Diff([]syncedProfession(nil), syncedProfessions(t, db)); diff != "" {
		t.Errorf("unexpected professions (-want +got):\n%s", diff)
	}
	wantLog := []syncLogEntry{{Source: "blizzard-professions", Status: "ok", Message: "0 characters: 0 professions stored, 0 not found"}}
	if diff := cmp.Diff(wantLog, syncLogEntries(t, db)); diff != "" {
		t.Errorf("unexpected sync log (-want +got):\n%s", diff)
	}
}

func TestSyncBlizzardProfessions_LogsError_WhenBlizzardFails(t *testing.T) {
	// given a Blizzard that answers with a server error
	db := testutil.DB(t)
	db.MustExec(syncCharacters)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"access_token": "app-token"})
	})
	mux.HandleFunc("GET /profile/wow/character/{slug}/professions", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusInternalServerError)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	// when the professions sync runs
	err := runProfessionsSync(db, srv)

	// then I expect an error and an error row in the sync log
	if err == nil {
		t.Fatal("sync succeeded, want an error")
	}
	entries := syncLogEntries(t, db)
	if len(entries) != 1 || entries[0].Status != "error" || entries[0].Source != "blizzard-professions" {
		t.Errorf("sync log = %+v, want one blizzard-professions error entry", entries)
	}
}
