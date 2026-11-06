package roster_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jmoiron/sqlx"

	"github.com/cageymage/fuzion/backend/internal/blizzard"
	"github.com/cageymage/fuzion/backend/internal/roster"
	"github.com/cageymage/fuzion/backend/internal/synclog"
	"github.com/cageymage/fuzion/backend/internal/testutil"
)

type fakeMember struct {
	name    string
	surname string
	class   int
	race    int
	level   int
	rank    int
}

type fakeBlizzardRoster struct {
	*httptest.Server

	members []fakeMember
	status  int
}

func newFakeBlizzardRoster(t *testing.T, members ...fakeMember) *fakeBlizzardRoster {
	t.Helper()

	f := &fakeBlizzardRoster{members: members, status: http.StatusOK}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"access_token": "app-token"})
	})
	mux.HandleFunc("GET /data/wow/guild/{guild}/roster", func(w http.ResponseWriter, r *http.Request) {
		if f.status != http.StatusOK {
			http.Error(w, "blizzard is down", f.status)
			return
		}
		list := []map[string]any{}
		for _, m := range f.members {
			list = append(list, map[string]any{
				"character": map[string]any{
					"name":           m.name,
					"level":          m.level,
					"surname":        m.surname,
					"playable_class": map[string]any{"id": m.class},
					"playable_race":  map[string]any{"id": m.race},
				},
				"rank": m.rank,
			})
		}
		json.NewEncoder(w).Encode(map[string]any{"members": list})
	})
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

func (f *fakeBlizzardRoster) runSync(db *sqlx.DB, altRanks ...int) error {
	client := blizzard.NewClient(blizzard.Config{
		ClientID: "id", ClientSecret: "secret", BaseURL: f.URL, AuthURL: f.URL, Namespace: "profile-classic1x-us",
	}, nil)
	return roster.NewSyncBlizzardRoster(roster.NewRepo(db), client, synclog.NewRepo(db), "fuzion", altRanks).Run(context.Background())
}

type syncedCharacter struct {
	Name          string  `db:"name"`
	SecondaryName string  `db:"secondary_name"`
	Realm         string  `db:"realm"`
	Class         string  `db:"class"`
	Role          string  `db:"role"`
	Spec          *string `db:"spec"`
	IsMain        bool    `db:"is_main"`
	RaidTeam      *string `db:"raid_team"`
	Race          *string `db:"race"`
	Level         *int    `db:"level"`
	GuildRank     *int    `db:"guild_rank"`
	Left          bool    `db:"left"`
}

func syncedCharacters(t *testing.T, db *sqlx.DB) []syncedCharacter {
	t.Helper()
	var rows []syncedCharacter
	err := db.Select(&rows, `
		SELECT name, secondary_name, realm, class, role, spec, is_main, raid_team, race, level, guild_rank,
		       left_guild_at IS NOT NULL AS "left"
		FROM characters ORDER BY name, secondary_name`)
	if err != nil {
		t.Fatalf("select characters: %v", err)
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

func characterID(t *testing.T, db *sqlx.DB, name string) string {
	t.Helper()
	var id string
	if err := db.Get(&id, `SELECT id FROM characters WHERE name = $1`, name); err != nil {
		t.Fatalf("select id of %s: %v", name, err)
	}
	return id
}

func intPtr(n int) *int { return &n }

func strPtr(s string) *string { return &s }

const (
	warrior = 1
	paladin = 2
	mage    = 8
	druid   = 11
	// Death Knight exists on retail only.
	deathKnight = 6
)

func TestSyncBlizzardRoster_InsertsNewCharacters_WhenBlizzardListsMembersTheTableLacks(t *testing.T) {
	// given an empty roster and a Blizzard guild with a warrior and a paladin
	db := testutil.DB(t)
	fake := newFakeBlizzardRoster(t,
		fakeMember{name: "Tankard", surname: "Ironhide", class: warrior, race: 1, level: 60, rank: 0},
		fakeMember{name: "Lightbringer", surname: "Brightwood", class: paladin, race: 3, level: 55, rank: 4},
	)

	// when the roster sync runs
	err := fake.runSync(db)

	// then I expect both characters added with placeholders for the fields Blizzard does not give
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	want := []syncedCharacter{
		{Name: "Lightbringer", SecondaryName: "Brightwood", Realm: "Emberreach", Class: "Paladin", Role: "dps", IsMain: true, Race: strPtr("Dwarf"), Level: intPtr(55), GuildRank: intPtr(4)},
		{Name: "Tankard", SecondaryName: "Ironhide", Realm: "Emberreach", Class: "Warrior", Role: "dps", IsMain: true, Race: strPtr("Human"), Level: intPtr(60), GuildRank: intPtr(0)},
	}
	if diff := cmp.Diff(want, syncedCharacters(t, db)); diff != "" {
		t.Errorf("unexpected characters (-want +got):\n%s", diff)
	}
}

func TestSyncBlizzardRoster_PreservesOfficerOwnedFields_WhenUpdatingExistingCharacter(t *testing.T) {
	// given an officer-managed tank with a main status, raid team and spec
	db := testutil.DB(t)
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, spec, role, is_main, raid_team, level)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Tankard', 'Ironhide', 'Emberreach', 'Warrior', 'Protection', 'tank', true, 'Team 1', 58)`)
	fake := newFakeBlizzardRoster(t, fakeMember{name: "TANKARD", surname: "IRONHIDE", class: warrior, race: 1, level: 60, rank: 2})

	// when the roster sync runs with a different case and a higher level
	err := fake.runSync(db)

	// then I expect level and rank refreshed and everything the officer owns untouched
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	protection := "Protection"
	team1 := "Team 1"
	want := []syncedCharacter{
		{Name: "Tankard", SecondaryName: "Ironhide", Realm: "Emberreach", Class: "Warrior", Role: "tank", Spec: &protection, IsMain: true, RaidTeam: &team1, Race: strPtr("Human"), Level: intPtr(60), GuildRank: intPtr(2)},
	}
	if diff := cmp.Diff(want, syncedCharacters(t, db)); diff != "" {
		t.Errorf("unexpected characters (-want +got):\n%s", diff)
	}
}

func TestSyncBlizzardRoster_MarksMissingCharactersAsLeft_WhenAnEarlierSyncSawThem(t *testing.T) {
	// given two characters imported by a first sync
	db := testutil.DB(t)
	first := newFakeBlizzardRoster(t,
		fakeMember{name: "Stayer", surname: "Stoneback", class: mage, race: 1, level: 60, rank: 1},
		fakeMember{name: "Quitter", surname: "Stoneback", class: druid, race: 1, level: 60, rank: 5},
	)
	if err := first.runSync(db); err != nil {
		t.Fatalf("first sync: %v", err)
	}

	// when a later sync no longer lists one of them
	second := newFakeBlizzardRoster(t, fakeMember{name: "Stayer", surname: "Stoneback", class: mage, race: 1, level: 60, rank: 1})
	err := second.runSync(db)

	// then I expect only the missing character to be marked as having left
	if err != nil {
		t.Fatalf("second sync: %v", err)
	}
	var left []string
	for _, c := range syncedCharacters(t, db) {
		if c.Left {
			left = append(left, c.Name)
		}
	}
	if diff := cmp.Diff([]string{"Quitter"}, left); diff != "" {
		t.Errorf("unexpected characters marked as left (-want +got):\n%s", diff)
	}
}

func TestSyncBlizzardRoster_ClearsLeftMarker_WhenCharacterIsListedAgain(t *testing.T) {
	// given a character a previous sync marked as having left
	db := testutil.DB(t)
	if err := newFakeBlizzardRoster(t,
		fakeMember{name: "Stayer", surname: "Stoneback", class: mage, race: 1, level: 60, rank: 1},
		fakeMember{name: "Boomerang", surname: "Stoneback", class: druid, race: 1, level: 60, rank: 5},
	).runSync(db); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	if err := newFakeBlizzardRoster(t, fakeMember{name: "Stayer", surname: "Stoneback", class: mage, race: 1, level: 60, rank: 1}).runSync(db); err != nil {
		t.Fatalf("second sync: %v", err)
	}

	// when a sync lists the character again
	err := newFakeBlizzardRoster(t,
		fakeMember{name: "Stayer", surname: "Stoneback", class: mage, race: 1, level: 60, rank: 1},
		fakeMember{name: "Boomerang", surname: "Stoneback", class: druid, race: 1, level: 60, rank: 5},
	).runSync(db)

	// then I expect nobody to be marked as having left
	if err != nil {
		t.Fatalf("third sync: %v", err)
	}
	for _, c := range syncedCharacters(t, db) {
		if c.Left {
			t.Errorf("%s is still marked as left", c.Name)
		}
	}
}

func TestSyncBlizzardRoster_LeavesHandEnteredCharactersVisible_WhenBlizzardNeverListedThem(t *testing.T) {
	// given a character an officer entered by hand that Blizzard has never listed
	db := testutil.DB(t)
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, role)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Recruit', 'Trialist', 'Emberreach', 'Rogue', 'dps')`)
	fake := newFakeBlizzardRoster(t, fakeMember{name: "Stayer", surname: "Stoneback", class: mage, race: 1, level: 60, rank: 1})

	// when the roster sync runs
	err := fake.runSync(db)

	// then I expect the hand-entered character not to be marked as having left
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	for _, c := range syncedCharacters(t, db) {
		if c.Name == "Recruit" && c.Left {
			t.Errorf("hand-entered character was marked as left")
		}
	}
}

func TestSyncBlizzardRoster_SkipsMembersAndRecordsIt_WhenClassIsNotSupported(t *testing.T) {
	// given a guild with a retail-only Death Knight and a level beyond the site's cap
	db := testutil.DB(t)
	fake := newFakeBlizzardRoster(t,
		fakeMember{name: "Grimblade", surname: "Stoneback", class: deathKnight, race: 1, level: 90, rank: 3},
		fakeMember{name: "Overleveled", surname: "Stoneback", class: mage, race: 1, level: 90, rank: 3},
	)

	// when the roster sync runs
	err := fake.runSync(db)

	// then I expect the Death Knight skipped, the mage added without a level, and the skip in the log
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	want := []syncedCharacter{
		{Name: "Overleveled", SecondaryName: "Stoneback", Realm: "Emberreach", Class: "Mage", Role: "dps", IsMain: true, Race: strPtr("Human"), GuildRank: intPtr(3)},
	}
	if diff := cmp.Diff(want, syncedCharacters(t, db)); diff != "" {
		t.Errorf("unexpected characters (-want +got):\n%s", diff)
	}
	wantLog := []syncLogEntry{{Source: "blizzard-roster", Status: "ok", Message: "2 members: 1 added, 0 updated, 0 left, 1 skipped (class 6 x1)"}}
	if diff := cmp.Diff(wantLog, syncLogEntries(t, db)); diff != "" {
		t.Errorf("unexpected sync log (-want +got):\n%s", diff)
	}
}

func TestSyncBlizzardRoster_ListsSkippedClassesMostCommonFirst_WhenSeveralClassesAreUnsupported(t *testing.T) {
	// given a guild with three Monks, one Death Knight and two Evokers
	db := testutil.DB(t)
	fake := newFakeBlizzardRoster(t,
		fakeMember{name: "Monka", surname: "Stoneback", class: 10, race: 1, level: 60, rank: 3},
		fakeMember{name: "Monkb", surname: "Stoneback", class: 10, race: 1, level: 60, rank: 3},
		fakeMember{name: "Monkc", surname: "Stoneback", class: 10, race: 1, level: 60, rank: 3},
		fakeMember{name: "Deathk", surname: "Stoneback", class: deathKnight, race: 1, level: 60, rank: 3},
		fakeMember{name: "Evokera", surname: "Stoneback", class: 13, race: 1, level: 60, rank: 3},
		fakeMember{name: "Evokerb", surname: "Stoneback", class: 13, race: 1, level: 60, rank: 3},
	)

	// when the roster sync runs
	err := fake.runSync(db)

	// then I expect the log to name each skipped class id with its count, most common first
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	wantLog := []syncLogEntry{{Source: "blizzard-roster", Status: "ok", Message: "6 members: 0 added, 0 updated, 0 left, 6 skipped (class 10 x3, 13 x2, 6 x1)"}}
	if diff := cmp.Diff(wantLog, syncLogEntries(t, db)); diff != "" {
		t.Errorf("unexpected sync log (-want +got):\n%s", diff)
	}
}

func TestSyncBlizzardRoster_LeavesTableUntouchedAndLogsError_WhenBlizzardFails(t *testing.T) {
	// given an imported character and a Blizzard that now returns a server error
	db := testutil.DB(t)
	if err := newFakeBlizzardRoster(t, fakeMember{name: "Stayer", surname: "Stoneback", class: mage, race: 1, level: 60, rank: 1}).runSync(db); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	down := newFakeBlizzardRoster(t)
	down.status = http.StatusInternalServerError

	// when the roster sync runs
	err := down.runSync(db)

	// then I expect an error, the character unchanged and an error row in the sync log
	if err == nil {
		t.Fatal("sync succeeded, want an error")
	}
	for _, c := range syncedCharacters(t, db) {
		if c.Left {
			t.Errorf("%s was marked as left by a failed sync", c.Name)
		}
	}
	entries := syncLogEntries(t, db)
	if len(entries) != 2 || entries[1].Status != "error" {
		t.Errorf("sync log = %+v, want the second entry to be an error", entries)
	}
}

func TestSyncBlizzardRoster_ReturnsErrorWithoutMarkingAnyoneAsLeft_WhenBlizzardReturnsNoMembers(t *testing.T) {
	// given an imported character and a Blizzard that returns an empty roster
	db := testutil.DB(t)
	if err := newFakeBlizzardRoster(t, fakeMember{name: "Stayer", surname: "Stoneback", class: mage, race: 1, level: 60, rank: 1}).runSync(db); err != nil {
		t.Fatalf("first sync: %v", err)
	}

	// when the roster sync runs
	err := newFakeBlizzardRoster(t).runSync(db)

	// then I expect an error and nobody marked as having left
	if err == nil {
		t.Fatal("sync succeeded, want an error")
	}
	for _, c := range syncedCharacters(t, db) {
		if c.Left {
			t.Errorf("%s was marked as left by an empty roster", c.Name)
		}
	}
}

func TestListRoster_HidesCharactersWhoLeftTheGuild(t *testing.T) {
	// given one current character and one marked as having left
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, role, left_guild_at)
		VALUES
			('11111111-1111-1111-1111-111111111111', 'Stayer', 'Unset', 'Emberreach', 'Mage', 'dps', NULL),
			('22222222-2222-2222-2222-222222222222', 'Quitter', 'Unset', 'Emberreach', 'Druid', 'dps', now())`)

	// when I ask for the roster
	resp := srv.Get(t, "/api/roster")

	// then I expect only the current character
	resp.RequireStatus(t, 200)
	var characters []characterJSON
	resp.DecodeJSON(t, &characters)
	if len(characters) != 1 || characters[0].Name != "Stayer" {
		t.Errorf("roster = %+v, want only Stayer", characters)
	}
}

func TestSyncBlizzardRoster_KeepsExistingRaceAndLogsIt_WhenRaceIdIsNotMapped(t *testing.T) {
	// given an officer-set Skyborne character and a new member, both with race ids the sync cannot name
	db := testutil.DB(t)
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, role, race)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Skywing', 'Stoneback', 'Emberreach', 'Mage', 'dps', 'Skyborne (High Order)')`)
	fake := newFakeBlizzardRoster(t,
		fakeMember{name: "Skywing", surname: "Stoneback", class: mage, race: 99, level: 60, rank: 1},
		fakeMember{name: "Newcomer", surname: "Stoneback", class: mage, race: 99, level: 60, rank: 1},
	)

	// when the roster sync runs
	err := fake.runSync(db)

	// then I expect the officer's race kept, the new member without a race, and the ids in the log
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	wantRaces := map[string]*string{"Skywing": strPtr("Skyborne (High Order)"), "Newcomer": nil}
	gotRaces := map[string]*string{}
	for _, c := range syncedCharacters(t, db) {
		gotRaces[c.Name] = c.Race
	}
	if diff := cmp.Diff(wantRaces, gotRaces); diff != "" {
		t.Errorf("unexpected races (-want +got):\n%s", diff)
	}
	wantLog := []syncLogEntry{{Source: "blizzard-roster", Status: "ok", Message: "2 members: 1 added, 1 updated, 0 left, 0 skipped (unmapped race 99 x2)"}}
	if diff := cmp.Diff(wantLog, syncLogEntries(t, db)); diff != "" {
		t.Errorf("unexpected sync log (-want +got):\n%s", diff)
	}
}

// Guild rank numbers for these tests; Forever's real ones are not known yet.
const (
	guildMasterRank = 0
	memberRank      = 1
	altRank         = 4
	bankAltRank     = 5
)

func TestSyncBlizzardRoster_SetsMainOrAltFromGuildRank_WhenAltRanksAreConfigured(t *testing.T) {
	// given an empty roster and a Blizzard guild with a guild master, a member, an alt and a bank alt
	db := testutil.DB(t)
	fake := newFakeBlizzardRoster(t,
		fakeMember{name: "Guildlord", surname: "Stoneback", class: warrior, race: 1, level: 60, rank: guildMasterRank},
		fakeMember{name: "Raider", surname: "Stoneback", class: mage, race: 1, level: 60, rank: memberRank},
		fakeMember{name: "Twinkalt", surname: "Stoneback", class: druid, race: 1, level: 60, rank: altRank},
		fakeMember{name: "Vaultkeep", surname: "Stoneback", class: paladin, race: 1, level: 1, rank: bankAltRank},
	)

	// when the roster sync runs with ranks 4 and 5 as alt ranks
	err := fake.runSync(db, altRank, bankAltRank)

	// then I expect alt-rank members imported as alts and every other rank as mains
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	want := []syncedCharacter{
		{Name: "Guildlord", SecondaryName: "Stoneback", Realm: "Emberreach", Class: "Warrior", Role: "dps", IsMain: true, Race: strPtr("Human"), Level: intPtr(60), GuildRank: intPtr(guildMasterRank)},
		{Name: "Raider", SecondaryName: "Stoneback", Realm: "Emberreach", Class: "Mage", Role: "dps", IsMain: true, Race: strPtr("Human"), Level: intPtr(60), GuildRank: intPtr(memberRank)},
		{Name: "Twinkalt", SecondaryName: "Stoneback", Realm: "Emberreach", Class: "Druid", Role: "dps", IsMain: false, Race: strPtr("Human"), Level: intPtr(60), GuildRank: intPtr(altRank)},
		{Name: "Vaultkeep", SecondaryName: "Stoneback", Realm: "Emberreach", Class: "Paladin", Role: "dps", IsMain: false, Race: strPtr("Human"), Level: intPtr(1), GuildRank: intPtr(bankAltRank)},
	}
	if diff := cmp.Diff(want, syncedCharacters(t, db)); diff != "" {
		t.Errorf("unexpected characters (-want +got):\n%s", diff)
	}
}

func TestSyncBlizzardRoster_MakesCharacterAMain_WhenItMovesOutOfAnAltRank(t *testing.T) {
	// given a character an earlier sync imported as an alt from the alt rank
	db := testutil.DB(t)
	if err := newFakeBlizzardRoster(t, fakeMember{name: "Raider", surname: "Stoneback", class: mage, race: 1, level: 60, rank: altRank}).runSync(db, altRank); err != nil {
		t.Fatalf("first sync: %v", err)
	}

	// when a sync lists the character in a member rank
	err := newFakeBlizzardRoster(t, fakeMember{name: "Raider", surname: "Stoneback", class: mage, race: 1, level: 60, rank: memberRank}).runSync(db, altRank)

	// then I expect the character to become a main
	if err != nil {
		t.Fatalf("second sync: %v", err)
	}
	want := []syncedCharacter{
		{Name: "Raider", SecondaryName: "Stoneback", Realm: "Emberreach", Class: "Mage", Role: "dps", IsMain: true, Race: strPtr("Human"), Level: intPtr(60), GuildRank: intPtr(memberRank)},
	}
	if diff := cmp.Diff(want, syncedCharacters(t, db)); diff != "" {
		t.Errorf("unexpected characters (-want +got):\n%s", diff)
	}
}

func TestSyncBlizzardRoster_KeepsMainAnOfficerSet_WhenCharacterIsInAnAltRank(t *testing.T) {
	// given a synced alt-rank character an officer has promoted to main
	db := testutil.DB(t)
	fake := newFakeBlizzardRoster(t, fakeMember{name: "Twinkalt", surname: "Stoneback", class: druid, race: 1, level: 60, rank: altRank})
	if err := fake.runSync(db, altRank); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	srv.Patch(t, "/api/roster/"+characterID(t, db, "Twinkalt"), map[string]any{"isMain": true}).RequireStatus(t, http.StatusOK)

	// when the roster sync runs again with the character still in the alt rank
	err := fake.runSync(db, altRank)

	// then I expect the officer's choice of main kept
	if err != nil {
		t.Fatalf("second sync: %v", err)
	}
	want := []syncedCharacter{
		{Name: "Twinkalt", SecondaryName: "Stoneback", Realm: "Emberreach", Class: "Druid", Role: "dps", IsMain: true, Race: strPtr("Human"), Level: intPtr(60), GuildRank: intPtr(altRank)},
	}
	if diff := cmp.Diff(want, syncedCharacters(t, db)); diff != "" {
		t.Errorf("unexpected characters (-want +got):\n%s", diff)
	}
}

func TestSyncBlizzardRoster_KeepsAltAnOfficerSet_WhenCharacterIsInAMemberRank(t *testing.T) {
	// given a synced member-rank character an officer has marked as an alt
	db := testutil.DB(t)
	fake := newFakeBlizzardRoster(t, fakeMember{name: "Raider", surname: "Stoneback", class: mage, race: 1, level: 60, rank: memberRank})
	if err := fake.runSync(db, altRank); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	srv.Patch(t, "/api/roster/"+characterID(t, db, "Raider"), map[string]any{"isMain": false}).RequireStatus(t, http.StatusOK)

	// when the roster sync runs again with the character still in the member rank
	err := fake.runSync(db, altRank)

	// then I expect the officer's choice of alt kept
	if err != nil {
		t.Fatalf("second sync: %v", err)
	}
	want := []syncedCharacter{
		{Name: "Raider", SecondaryName: "Stoneback", Realm: "Emberreach", Class: "Mage", Role: "dps", IsMain: false, Race: strPtr("Human"), Level: intPtr(60), GuildRank: intPtr(memberRank)},
	}
	if diff := cmp.Diff(want, syncedCharacters(t, db)); diff != "" {
		t.Errorf("unexpected characters (-want +got):\n%s", diff)
	}
}

func TestSyncBlizzardRoster_KeepsMainOrAltFollowingRank_WhenAnOfficerEditLeftItUnchanged(t *testing.T) {
	// given a synced alt-rank character whose raid team, but not main/alt, an officer has changed
	db := testutil.DB(t)
	if err := newFakeBlizzardRoster(t, fakeMember{name: "Raider", surname: "Stoneback", class: mage, race: 1, level: 60, rank: altRank}).runSync(db, altRank); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	srv.Patch(t, "/api/roster/"+characterID(t, db, "Raider"), map[string]any{"raidTeam": "Team 1"}).RequireStatus(t, http.StatusOK)

	// when a sync lists the character in a member rank
	err := newFakeBlizzardRoster(t, fakeMember{name: "Raider", surname: "Stoneback", class: mage, race: 1, level: 60, rank: memberRank}).runSync(db, altRank)

	// then I expect the character to become a main
	if err != nil {
		t.Fatalf("second sync: %v", err)
	}
	want := []syncedCharacter{
		{Name: "Raider", SecondaryName: "Stoneback", Realm: "Emberreach", Class: "Mage", Role: "dps", IsMain: true, RaidTeam: strPtr("Team 1"), Race: strPtr("Human"), Level: intPtr(60), GuildRank: intPtr(memberRank)},
	}
	if diff := cmp.Diff(want, syncedCharacters(t, db)); diff != "" {
		t.Errorf("unexpected characters (-want +got):\n%s", diff)
	}
}

func TestSyncBlizzardRoster_KeepsMainOrAltOfOfficerEnteredCharacter_WhenAltRanksAreConfigured(t *testing.T) {
	// given a main an officer entered by hand
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	srv.Post(t, "/api/roster", map[string]any{
		"name": "Tankard", "secondaryName": "Ironhide", "class": "Warrior", "role": "tank", "isMain": true,
	}).RequireStatus(t, http.StatusCreated)
	fake := newFakeBlizzardRoster(t, fakeMember{name: "Tankard", surname: "Ironhide", class: warrior, race: 1, level: 60, rank: altRank})

	// when the roster sync lists the character in an alt rank
	err := fake.runSync(db, altRank)

	// then I expect the character to stay a main
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	want := []syncedCharacter{
		{Name: "Tankard", SecondaryName: "Ironhide", Realm: "Emberreach", Class: "Warrior", Role: "tank", IsMain: true, Race: strPtr("Human"), Level: intPtr(60), GuildRank: intPtr(altRank)},
	}
	if diff := cmp.Diff(want, syncedCharacters(t, db)); diff != "" {
		t.Errorf("unexpected characters (-want +got):\n%s", diff)
	}
}

func TestSyncBlizzardRoster_SyncsCharactersAsMains_WhenNoAltRanksAreConfigured(t *testing.T) {
	// given a member-rank character an earlier sync imported without alt ranks
	db := testutil.DB(t)
	fake := newFakeBlizzardRoster(t, fakeMember{name: "Raider", surname: "Stoneback", class: mage, race: 1, level: 60, rank: memberRank})
	if err := fake.runSync(db); err != nil {
		t.Fatalf("first sync: %v", err)
	}

	// when the roster sync runs again without alt ranks
	err := fake.runSync(db)

	// then I expect the character to be a main, whatever its rank
	if err != nil {
		t.Fatalf("second sync: %v", err)
	}
	want := []syncedCharacter{
		{Name: "Raider", SecondaryName: "Stoneback", Realm: "Emberreach", Class: "Mage", Role: "dps", IsMain: true, Race: strPtr("Human"), Level: intPtr(60), GuildRank: intPtr(memberRank)},
	}
	if diff := cmp.Diff(want, syncedCharacters(t, db)); diff != "" {
		t.Errorf("unexpected characters (-want +got):\n%s", diff)
	}
}

func TestSyncBlizzardRoster_TreatsSameNameWithDifferentSurnamesAsSeparateCharacters(t *testing.T) {
	// given a Blizzard guild with two characters that share a first name
	db := testutil.DB(t)
	fake := newFakeBlizzardRoster(t,
		fakeMember{name: "Kirk", surname: "Fel", class: warrior, race: 1, level: 11, rank: memberRank},
		fakeMember{name: "Kirk", surname: "Smash", class: warrior, race: 1, level: 1, rank: memberRank},
	)

	// when the roster sync runs
	err := fake.runSync(db)

	// then I expect both stored, each with its own surname as the secondary name
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	want := []syncedCharacter{
		{Name: "Kirk", SecondaryName: "Fel", Realm: "Emberreach", Class: "Warrior", Role: "dps", IsMain: true, Race: strPtr("Human"), Level: intPtr(11), GuildRank: intPtr(memberRank)},
		{Name: "Kirk", SecondaryName: "Smash", Realm: "Emberreach", Class: "Warrior", Role: "dps", IsMain: true, Race: strPtr("Human"), Level: intPtr(1), GuildRank: intPtr(memberRank)},
	}
	if diff := cmp.Diff(want, syncedCharacters(t, db)); diff != "" {
		t.Errorf("unexpected characters (-want +got):\n%s", diff)
	}
}

func TestSyncBlizzardRoster_FillsInSurname_WhenCharacterHasThePlaceholderSecondaryName(t *testing.T) {
	// given a character an earlier sync imported with the placeholder secondary name
	db := testutil.DB(t)
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, realm, class, role, raid_team)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Tankard', 'Unset', 'Emberreach', 'Warrior', 'tank', 'Team 1')`)
	fake := newFakeBlizzardRoster(t, fakeMember{name: "Tankard", surname: "Ironhide", class: warrior, race: 1, level: 60, rank: memberRank})

	// when the roster sync runs
	err := fake.runSync(db)

	// then I expect that row to take Blizzard's surname instead of a second Tankard being added
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	team1 := "Team 1"
	want := []syncedCharacter{
		{Name: "Tankard", SecondaryName: "Ironhide", Realm: "Emberreach", Class: "Warrior", Role: "tank", IsMain: true, RaidTeam: &team1, Race: strPtr("Human"), Level: intPtr(60), GuildRank: intPtr(memberRank)},
	}
	if diff := cmp.Diff(want, syncedCharacters(t, db)); diff != "" {
		t.Errorf("unexpected characters (-want +got):\n%s", diff)
	}
}

func TestSyncBlizzardRoster_MarksOnlyTheMissingSurnameAsLeft_WhenAnotherCharacterSharesTheName(t *testing.T) {
	// given two characters named Kirk imported by a first sync
	db := testutil.DB(t)
	if err := newFakeBlizzardRoster(t,
		fakeMember{name: "Kirk", surname: "Fel", class: warrior, race: 1, level: 11, rank: memberRank},
		fakeMember{name: "Kirk", surname: "Smash", class: warrior, race: 1, level: 1, rank: memberRank},
	).runSync(db); err != nil {
		t.Fatalf("first sync: %v", err)
	}

	// when a later sync lists only Kirk Fel
	err := newFakeBlizzardRoster(t, fakeMember{name: "Kirk", surname: "Fel", class: warrior, race: 1, level: 11, rank: memberRank}).runSync(db)

	// then I expect only Kirk Smash marked as having left
	if err != nil {
		t.Fatalf("second sync: %v", err)
	}
	var left []string
	for _, c := range syncedCharacters(t, db) {
		if c.Left {
			left = append(left, c.Name+" "+c.SecondaryName)
		}
	}
	if diff := cmp.Diff([]string{"Kirk Smash"}, left); diff != "" {
		t.Errorf("unexpected characters marked as left (-want +got):\n%s", diff)
	}
}
