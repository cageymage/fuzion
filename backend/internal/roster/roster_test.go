package roster_test

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
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Realm     string  `json:"realm"`
	Class     string  `json:"class"`
	Spec      *string `json:"spec"`
	Role      string  `json:"role"`
	IsMain    bool    `json:"isMain"`
	RaidTeam  *string `json:"raidTeam"`
	CreatedAt string  `json:"createdAt"`
}

func TestListRoster_ReturnsMainsBeforeAlts(t *testing.T) {
	// given an alt whose name sorts alphabetically before the guild's only main
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO characters (id, name, realm, class, spec, role, is_main, raid_team, created_at)
		VALUES
			('11111111-1111-1111-1111-111111111111', 'Aeliana', 'Emberreach', 'Priest', 'Holy', 'healer', false, NULL, '2026-09-01T12:00:00Z'),
			('22222222-2222-2222-2222-222222222222', 'Zephyrion', 'Emberreach', 'Warrior', 'Protection', 'tank', true, 'Team 1', '2026-09-01T12:00:00Z')`)

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
			ID:        "22222222-2222-2222-2222-222222222222",
			Name:      "Zephyrion",
			Realm:     "Emberreach",
			Class:     "Warrior",
			Spec:      &protection,
			Role:      "tank",
			IsMain:    true,
			RaidTeam:  &team1,
			CreatedAt: "2026-09-01T12:00:00Z",
		},
		{
			ID:        "11111111-1111-1111-1111-111111111111",
			Name:      "Aeliana",
			Realm:     "Emberreach",
			Class:     "Priest",
			Spec:      &holy,
			Role:      "healer",
			IsMain:    false,
			RaidTeam:  nil,
			CreatedAt: "2026-09-01T12:00:00Z",
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
		INSERT INTO characters (id, name, realm, class, spec, role, is_main, raid_team, created_at)
		VALUES
			('11111111-1111-1111-1111-111111111111', 'Zenda', 'Emberreach', 'Mage', 'Frost', 'dps', true, 'Team 1', '2026-09-01T12:00:00Z'),
			('22222222-2222-2222-2222-222222222222', 'Aeliana', 'Emberreach', 'Priest', 'Holy', 'healer', true, 'Team 1', '2026-09-01T12:00:00Z'),
			('33333333-3333-3333-3333-333333333333', 'Yorick', 'Emberreach', 'Rogue', 'Assassination', 'dps', false, NULL, '2026-09-01T12:00:00Z'),
			('44444444-4444-4444-4444-444444444444', 'Brannor', 'Emberreach', 'Druid', 'Feral', 'dps', false, NULL, '2026-09-01T12:00:00Z')`)

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
