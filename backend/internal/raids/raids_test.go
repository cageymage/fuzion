package raids_test

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/cageymage/fuzion/backend/internal/testutil"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.Run(m))
}

type raidJSON struct {
	ID              string `json:"id"`
	Difficulty      string `json:"difficulty"`
	InstanceName    string `json:"instanceName"`
	StartsAt        string `json:"startsAt"`
	ProgressSummary string `json:"progressSummary"`
}

func TestGetNextRaid_ReturnsSoonestUpcomingRaid_WhenSeveralAreScheduled(t *testing.T) {
	// given a finished raid and two upcoming ones
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO raids (id, difficulty, instance_name, starts_at, progress_summary)
		VALUES
			('11111111-1111-1111-1111-111111111111', 'Heroic', 'Amirdrassil', now() - interval '2 days', '9/9 Heroic cleared'),
			('22222222-2222-2222-2222-222222222222', 'Mythic', 'Nerubar Palace', '2099-01-02T20:00:00Z', '8/8 Heroic cleared'),
			('33333333-3333-3333-3333-333333333333', 'Mythic', 'Liberation of Undermine', '2099-01-01T20:00:00Z', '8/8 Heroic cleared')`)

	// when I ask for the next raid
	resp := srv.Get(t, "/api/raids/next")

	// then I expect a 200 with the soonest raid still in the future
	resp.RequireStatus(t, 200)
	var raid raidJSON
	resp.DecodeJSON(t, &raid)

	want := raidJSON{
		ID:              "33333333-3333-3333-3333-333333333333",
		Difficulty:      "Mythic",
		InstanceName:    "Liberation of Undermine",
		StartsAt:        "2099-01-01T20:00:00Z",
		ProgressSummary: "8/8 Heroic cleared",
	}
	if diff := cmp.Diff(want, raid); diff != "" {
		t.Errorf("unexpected next raid (-want +got):\n%s", diff)
	}
}

func TestGetNextRaid_ReturnsNull_WhenEveryRaidHasAlreadyStarted(t *testing.T) {
	// given only raids whose start time has passed
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO raids (id, difficulty, instance_name, starts_at, progress_summary)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Heroic', 'Amirdrassil', now() - interval '1 hour', '9/9 Heroic cleared')`)

	// when I ask for the next raid
	resp := srv.Get(t, "/api/raids/next")

	// then I expect a 200 with a null body, because an empty calendar is not an error
	resp.RequireStatus(t, 200)
	if got := string(resp.Body); got != "null\n" {
		t.Errorf("expected a null body, got %q", got)
	}
}
