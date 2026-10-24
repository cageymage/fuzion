package raidhelper_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jmoiron/sqlx"

	"github.com/cageymage/fuzion/backend/internal/testutil"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.Run(m))
}

const webhookPath = "/api/webhooks/raid-helper"

// The real captures in testdata/ all describe one event, 2026-10-13 00:00 UTC.
const capturedEventID = "1100000000000000001"

var capturedStart = time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)

type raidRow struct {
	EventID      *string    `db:"raid_helper_event_id"`
	Difficulty   string     `db:"difficulty"`
	InstanceName string     `db:"instance_name"`
	StartsAt     time.Time  `db:"starts_at"`
	LastUpdated  *int64     `db:"raid_helper_last_updated"`
	DeletedAt    *time.Time `db:"deleted_at"`
}

func raidRows(t *testing.T, db *sqlx.DB) []raidRow {
	t.Helper()
	var rows []raidRow
	if err := db.Select(&rows, `SELECT raid_helper_event_id, difficulty, instance_name, starts_at, raid_helper_last_updated, deleted_at FROM raids ORDER BY instance_name`); err != nil {
		t.Fatalf("select raids: %v", err)
	}
	for i := range rows {
		rows[i].StartsAt = rows[i].StartsAt.UTC()
	}
	return rows
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return body
}

// withFields returns the fixture with some top-level fields replaced, keeping the rest real.
func withFields(t *testing.T, name string, fields map[string]any) []byte {
	t.Helper()
	var event map[string]any
	if err := json.Unmarshal(fixture(t, name), &event); err != nil {
		t.Fatalf("decode fixture %s: %v", name, err)
	}
	for key, value := range fields {
		event[key] = value
	}
	body, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("encode fixture %s: %v", name, err)
	}
	return body
}

func deliver(t *testing.T, srv *testutil.Server, eventType string, body []byte) testutil.Response {
	t.Helper()
	return srv.PostRawWithHeaders(t, webhookPath, body, map[string]string{
		"Content-Type":       "application/json",
		"User-Agent":         "Raid-Helper/1.0",
		"Authorization":      testutil.RaidHelperWebhookKey,
		"X-Raidhelper-Event": eventType,
	})
}

func ptr[T any](v T) *T { return &v }

func TestRaidHelperWebhook_CreatesRaid_WhenEventCreatedIsReceived(t *testing.T) {
	// given no raids
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when Raid-Helper delivers the captured create
	resp := deliver(t, srv, "webhookeventcreate", fixture(t, "webhookeventcreate.json"))

	// then I expect a 200 and one raid mirrored from the event
	resp.RequireStatus(t, 200)
	want := []raidRow{{
		EventID:      ptr(capturedEventID),
		Difficulty:   "Unknown",
		InstanceName: "Test Raid Event",
		StartsAt:     capturedStart,
		LastUpdated:  ptr(int64(1791781357)),
	}}
	if diff := cmp.Diff(want, raidRows(t, db)); diff != "" {
		t.Errorf("unexpected raids (-want +got):\n%s", diff)
	}
}

func TestRaidHelperWebhook_StoresOneRaid_WhenCreateAndItsUpdateArriveTogether(t *testing.T) {
	// given Raid-Helper has delivered the create sent at event creation
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	deliver(t, srv, "webhookeventcreate", fixture(t, "webhookeventcreate.json")).RequireStatus(t, 200)

	// when the identical update sent at the same moment arrives
	resp := deliver(t, srv, "webhookeventupdate", fixture(t, "webhookeventupdate-after-create.json"))

	// then I expect a 200 and still exactly one raid
	resp.RequireStatus(t, 200)
	want := []raidRow{{
		EventID:      ptr(capturedEventID),
		Difficulty:   "Unknown",
		InstanceName: "Test Raid Event",
		StartsAt:     capturedStart,
		LastUpdated:  ptr(int64(1791781357)),
	}}
	if diff := cmp.Diff(want, raidRows(t, db)); diff != "" {
		t.Errorf("unexpected raids (-want +got):\n%s", diff)
	}
}

func TestRaidHelperWebhook_UpdatesExistingRaid_WhenEventUpdatedIsReceived(t *testing.T) {
	// given a raid mirrored from the captured create
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	deliver(t, srv, "webhookeventcreate", fixture(t, "webhookeventcreate.json")).RequireStatus(t, 200)

	// when the update captured after the title was edited arrives
	resp := deliver(t, srv, "webhookeventupdate", fixture(t, "webhookeventupdate-signup.json"))

	// then I expect the same raid changed in place
	resp.RequireStatus(t, 200)
	want := []raidRow{{
		EventID:      ptr(capturedEventID),
		Difficulty:   "Unknown",
		InstanceName: "Test Raid Event (edited)",
		StartsAt:     capturedStart,
		LastUpdated:  ptr(int64(1791781505)),
	}}
	if diff := cmp.Diff(want, raidRows(t, db)); diff != "" {
		t.Errorf("unexpected raids (-want +got):\n%s", diff)
	}
}

func TestRaidHelperWebhook_IgnoresUpdate_WhenItIsOlderThanTheStoredOne(t *testing.T) {
	// given a raid already holding the newer, edited update
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	deliver(t, srv, "webhookeventupdate", fixture(t, "webhookeventupdate-signup.json")).RequireStatus(t, 200)

	// when the older update sent at creation arrives late
	resp := deliver(t, srv, "webhookeventupdate", fixture(t, "webhookeventupdate-after-create.json"))

	// then I expect a 200 and the newer title kept
	resp.RequireStatus(t, 200)
	want := []raidRow{{
		EventID:      ptr(capturedEventID),
		Difficulty:   "Unknown",
		InstanceName: "Test Raid Event (edited)",
		StartsAt:     capturedStart,
		LastUpdated:  ptr(int64(1791781505)),
	}}
	if diff := cmp.Diff(want, raidRows(t, db)); diff != "" {
		t.Errorf("unexpected raids (-want +got):\n%s", diff)
	}
}

func TestRaidHelperWebhook_AppliesUpdate_WhenLastUpdatedEqualsTheStoredOne(t *testing.T) {
	// given a raid mirrored from the captured create
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	deliver(t, srv, "webhookeventcreate", fixture(t, "webhookeventcreate.json")).RequireStatus(t, 200)

	// when an update with the same lastUpdated but a new title arrives
	body := withFields(t, "webhookeventupdate-after-create.json", map[string]any{"title": "Retitled without a lastUpdated bump"})
	resp := deliver(t, srv, "webhookeventupdate", body)

	// then I expect the new title applied, since lastUpdated is unverified for title-only edits
	resp.RequireStatus(t, 200)
	want := []raidRow{{
		EventID:      ptr(capturedEventID),
		Difficulty:   "Unknown",
		InstanceName: "Retitled without a lastUpdated bump",
		StartsAt:     capturedStart,
		LastUpdated:  ptr(int64(1791781357)),
	}}
	if diff := cmp.Diff(want, raidRows(t, db)); diff != "" {
		t.Errorf("unexpected raids (-want +got):\n%s", diff)
	}
}

func TestRaidHelperWebhook_DeletesRaid_WhenEventDeletedIsReceived(t *testing.T) {
	// given a raid mirrored from the captured Absence update
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	deliver(t, srv, "webhookeventupdate", fixture(t, "webhookeventupdate-absence.json")).RequireStatus(t, 200)

	// when the captured delete arrives, carrying the same lastUpdated as that update
	resp := deliver(t, srv, "webhookeventdelete", fixture(t, "webhookeventdelete.json"))

	// then I expect a 200 and the raid tombstoned
	resp.RequireStatus(t, 200)
	rows := raidRows(t, db)
	if len(rows) != 1 || rows[0].DeletedAt == nil {
		t.Fatalf("expected one tombstoned raid, got %+v", rows)
	}
	rows[0].DeletedAt = nil
	want := raidRow{
		EventID:      ptr(capturedEventID),
		Difficulty:   "Unknown",
		InstanceName: "Test Raid Event (edited)",
		StartsAt:     capturedStart,
		LastUpdated:  ptr(int64(1791781634)),
	}
	if diff := cmp.Diff(want, rows[0]); diff != "" {
		t.Errorf("unexpected tombstone (-want +got):\n%s", diff)
	}
}

func TestRaidHelperWebhook_KeepsRaidDeleted_WhenAnUpdateArrivesAfterTheDelete(t *testing.T) {
	// given a raid that has been deleted
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	deliver(t, srv, "webhookeventupdate", fixture(t, "webhookeventupdate-absence.json")).RequireStatus(t, 200)
	deliver(t, srv, "webhookeventdelete", fixture(t, "webhookeventdelete.json")).RequireStatus(t, 200)

	// when a late update for the same event arrives with a newer lastUpdated and title
	resp := deliver(t, srv, "webhookeventupdate", withFields(t, "webhookeventupdate-absence.json", map[string]any{"lastUpdated": 1791781999, "title": "Brought back"}))

	// then I expect a 200 and the tombstone exactly as the delete left it
	resp.RequireStatus(t, 200)
	rows := raidRows(t, db)
	if len(rows) != 1 || rows[0].DeletedAt == nil {
		t.Fatalf("expected one tombstoned raid, got %+v", rows)
	}
	rows[0].DeletedAt = nil
	want := raidRow{
		EventID:      ptr(capturedEventID),
		Difficulty:   "Unknown",
		InstanceName: "Test Raid Event (edited)",
		StartsAt:     capturedStart,
		LastUpdated:  ptr(int64(1791781634)),
	}
	if diff := cmp.Diff(want, rows[0]); diff != "" {
		t.Errorf("tombstone changed by the late update (-want +got):\n%s", diff)
	}
}

func TestRaidHelperWebhook_StoresTombstone_WhenDeleteArrivesForAnUnknownEvent(t *testing.T) {
	// given no raids
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when a delete arrives for an event the site never saw
	resp := deliver(t, srv, "webhookeventdelete", fixture(t, "webhookeventdelete.json"))

	// then I expect a 200 and a tombstone, so a create arriving later cannot add the raid
	resp.RequireStatus(t, 200)
	rows := raidRows(t, db)
	if len(rows) != 1 || rows[0].DeletedAt == nil {
		t.Fatalf("expected one tombstone for the event, got %+v", rows)
	}
	rows[0].DeletedAt = nil
	want := raidRow{
		EventID:      ptr(capturedEventID),
		Difficulty:   "Unknown",
		InstanceName: "Test Raid Event (edited)",
		StartsAt:     capturedStart,
		LastUpdated:  ptr(int64(1791781634)),
	}
	if diff := cmp.Diff(want, rows[0]); diff != "" {
		t.Errorf("unexpected tombstone (-want +got):\n%s", diff)
	}
}

func TestRaidHelperWebhook_ReturnsUnauthorized_WhenKeyIsWrong(t *testing.T) {
	// given no raids
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when a create arrives with the wrong key
	resp := srv.PostRawWithHeaders(t, webhookPath, fixture(t, "webhookeventcreate.json"), map[string]string{
		"Authorization":      "zZ9yY8xX7wW6vV5uU4tT3sS2rR1qQ0",
		"X-Raidhelper-Event": "webhookeventcreate",
	})

	// then I expect a 401 and nothing written
	resp.RequireStatus(t, 401)
	if rows := raidRows(t, db); len(rows) != 0 {
		t.Errorf("expected no raids, got %+v", rows)
	}
}

func TestRaidHelperWebhook_ReturnsUnauthorized_WhenAuthorizationHeaderIsMissing(t *testing.T) {
	// given no raids
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when a create arrives with no Authorization header
	resp := srv.PostRawWithHeaders(t, webhookPath, fixture(t, "webhookeventcreate.json"), map[string]string{
		"X-Raidhelper-Event": "webhookeventcreate",
	})

	// then I expect a 401 and nothing written
	resp.RequireStatus(t, 401)
	if rows := raidRows(t, db); len(rows) != 0 {
		t.Errorf("expected no raids, got %+v", rows)
	}
}

func TestRaidHelperWebhook_IgnoresUnknownEventTypes(t *testing.T) {
	// given no raids
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when a correctly keyed delivery of a type the site does not handle arrives
	resp := deliver(t, srv, "webhookcompupdate", []byte(`{"id":"comp-1","title":"Week 8 comp"}`))

	// then I expect a 200 and nothing written
	resp.RequireStatus(t, 200)
	if rows := raidRows(t, db); len(rows) != 0 {
		t.Errorf("expected no raids, got %+v", rows)
	}
}

const (
	retailRaidPlanningChannelID = "1281325821813985391"
	unknownChannelID            = "1399999999999999999"
)

func TestRaidHelperWebhook_IgnoresEvent_WhenChannelIsUnknown(t *testing.T) {
	// given no raids
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when the captured create arrives from a channel the site does not know
	body := withFields(t, "webhookeventcreate.json", map[string]any{"channelId": unknownChannelID, "channelName": "officer-test"})
	resp := deliver(t, srv, "webhookeventcreate", body)

	// then I expect a 200 and nothing written
	resp.RequireStatus(t, 200)
	if rows := raidRows(t, db); len(rows) != 0 {
		t.Errorf("expected no raids, got %+v", rows)
	}
}

func TestRaidHelperWebhook_IgnoresEvent_WhenChannelIsRetailRaidPlanning(t *testing.T) {
	// given no raids
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when the captured create arrives from the Retail raid-planning channel
	body := withFields(t, "webhookeventcreate.json", map[string]any{"channelId": retailRaidPlanningChannelID, "channelName": "retail-raid-planning"})
	resp := deliver(t, srv, "webhookeventcreate", body)

	// then I expect a 200 and nothing written, since the Calendar mirrors only Forever raids
	resp.RequireStatus(t, 200)
	if rows := raidRows(t, db); len(rows) != 0 {
		t.Errorf("expected no raids, got %+v", rows)
	}
}

func TestRaidHelperWebhook_DoesNotUpdateRaid_WhenUpdateComesFromUnknownChannel(t *testing.T) {
	// given a raid mirrored from the captured create
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	deliver(t, srv, "webhookeventcreate", fixture(t, "webhookeventcreate.json")).RequireStatus(t, 200)

	// when the edited update for the same event arrives from an unknown channel
	body := withFields(t, "webhookeventupdate-signup.json", map[string]any{"channelId": unknownChannelID})
	resp := deliver(t, srv, "webhookeventupdate", body)

	// then I expect a 200 and the raid exactly as the create left it
	resp.RequireStatus(t, 200)
	want := []raidRow{{
		EventID:      ptr(capturedEventID),
		Difficulty:   "Unknown",
		InstanceName: "Test Raid Event",
		StartsAt:     capturedStart,
		LastUpdated:  ptr(int64(1791781357)),
	}}
	if diff := cmp.Diff(want, raidRows(t, db)); diff != "" {
		t.Errorf("unexpected raids (-want +got):\n%s", diff)
	}
}

func TestRaidHelperWebhook_DoesNotDeleteRaid_WhenDeleteComesFromUnknownChannel(t *testing.T) {
	// given a raid mirrored from the captured Absence update
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	deliver(t, srv, "webhookeventupdate", fixture(t, "webhookeventupdate-absence.json")).RequireStatus(t, 200)

	// when the captured delete arrives from an unknown channel
	body := withFields(t, "webhookeventdelete.json", map[string]any{"channelId": unknownChannelID})
	resp := deliver(t, srv, "webhookeventdelete", body)

	// then I expect a 200 and the raid still live
	resp.RequireStatus(t, 200)
	want := []raidRow{{
		EventID:      ptr(capturedEventID),
		Difficulty:   "Unknown",
		InstanceName: "Test Raid Event (edited)",
		StartsAt:     capturedStart,
		LastUpdated:  ptr(int64(1791781634)),
	}}
	if diff := cmp.Diff(want, raidRows(t, db)); diff != "" {
		t.Errorf("unexpected raids (-want +got):\n%s", diff)
	}
}

func TestRaidHelperWebhook_ReturnsUnauthorized_WhenKeyIsInvalidAndChannelIsUnknown(t *testing.T) {
	// given no raids
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when a create from an unknown channel arrives with the wrong key
	body := withFields(t, "webhookeventcreate.json", map[string]any{"channelId": unknownChannelID})
	resp := srv.PostRawWithHeaders(t, webhookPath, body, map[string]string{
		"Authorization":      "zZ9yY8xX7wW6vV5uU4tT3sS2rR1qQ0",
		"X-Raidhelper-Event": "webhookeventcreate",
	})

	// then I expect a 401, not the 200 an unknown channel gets, and nothing written
	resp.RequireStatus(t, 401)
	if rows := raidRows(t, db); len(rows) != 0 {
		t.Errorf("expected no raids, got %+v", rows)
	}
}

func TestRaidHelperWebhook_ReturnsBadRequest_WhenBodyHasNoEventId(t *testing.T) {
	// given no raids
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when a create arrives whose body has no id
	resp := deliver(t, srv, "webhookeventcreate", withFields(t, "webhookeventcreate.json", map[string]any{"id": ""}))

	// then I expect a 400 and nothing written
	resp.RequireStatus(t, 400)
	if rows := raidRows(t, db); len(rows) != 0 {
		t.Errorf("expected no raids, got %+v", rows)
	}
}

func TestRaidHelperWebhook_LeavesHandEnteredRaidsUntouched(t *testing.T) {
	// given a hand-entered raid with the same title as the captured event
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO raids (id, difficulty, instance_name, starts_at, progress_summary)
		VALUES ('11111111-1111-1111-1111-111111111111', 'Heroic', 'Test Raid Event', '2099-01-01T20:00:00Z', '8/8 Heroic cleared')`)

	// when the captured delete for the Raid-Helper event arrives
	resp := deliver(t, srv, "webhookeventdelete", fixture(t, "webhookeventdelete.json"))

	// then I expect the hand-entered raid unchanged
	resp.RequireStatus(t, 200)
	var handEntered raidRow
	if err := db.Get(&handEntered, `SELECT raid_helper_event_id, difficulty, instance_name, starts_at, raid_helper_last_updated, deleted_at FROM raids WHERE id = '11111111-1111-1111-1111-111111111111'`); err != nil {
		t.Fatalf("select hand-entered raid: %v", err)
	}
	handEntered.StartsAt = handEntered.StartsAt.UTC()
	want := raidRow{Difficulty: "Heroic", InstanceName: "Test Raid Event", StartsAt: time.Date(2099, 1, 1, 20, 0, 0, 0, time.UTC)}
	if diff := cmp.Diff(want, handEntered); diff != "" {
		t.Errorf("hand-entered raid changed (-want +got):\n%s", diff)
	}
}

func TestRaidHelperWebhook_ReturnsNotFound_WhenNoWebhookKeyIsConfigured(t *testing.T) {
	// given a deploy without RAID_HELPER_WEBHOOK_SECRET
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db, testutil.WithoutRaidHelperWebhook())

	// when a create arrives, even with no Authorization header to compare against
	resp := srv.PostRawWithHeaders(t, webhookPath, fixture(t, "webhookeventcreate.json"), map[string]string{
		"X-Raidhelper-Event": "webhookeventcreate",
	})

	// then I expect a 404 and nothing written
	resp.RequireStatus(t, 404)
	if rows := raidRows(t, db); len(rows) != 0 {
		t.Errorf("expected no raids, got %+v", rows)
	}
}

func TestRaidHelperWebhook_SetsDifficulty_WhenTheTitleNamesOne(t *testing.T) {
	// given no raids
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when a create arrives whose title names a difficulty
	resp := deliver(t, srv, "webhookeventcreate", withFields(t, "webhookeventcreate.json", map[string]any{"title": "Week 9 heroic clear"}))

	// then I expect the difficulty taken from the title
	resp.RequireStatus(t, 200)
	want := []raidRow{{
		EventID:      ptr(capturedEventID),
		Difficulty:   "Heroic",
		InstanceName: "Week 9 heroic clear",
		StartsAt:     capturedStart,
		LastUpdated:  ptr(int64(1791781357)),
	}}
	if diff := cmp.Diff(want, raidRows(t, db)); diff != "" {
		t.Errorf("unexpected raids (-want +got):\n%s", diff)
	}
}
