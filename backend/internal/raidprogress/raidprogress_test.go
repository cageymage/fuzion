package raidprogress_test

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

type bossJSON struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	KilledAt *string `json:"killedAt"`
}

type progressJSON struct {
	Tier struct {
		Name string `json:"name"`
	} `json:"tier"`
	Bosses []bossJSON `json:"bosses"`
	Killed int        `json:"killed"`
	Total  int        `json:"total"`
}

type tierJSON struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	IsCurrent bool       `json:"isCurrent"`
	SortOrder int        `json:"sortOrder"`
	Bosses    []bossJSON `json:"bosses"`
}

func requireErrorBody(t *testing.T, resp testutil.Response, want string) {
	t.Helper()

	var body map[string]string
	resp.DecodeJSON(t, &body)
	if diff := cmp.Diff(map[string]string{"error": want}, body); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
}

func insertTier(t *testing.T, db *sqlx.DB, id, name string, isCurrent bool) {
	t.Helper()
	db.MustExec(`INSERT INTO raid_tiers (id, name, is_current) VALUES ($1, $2, $3)`, id, name, isCurrent)
}

func insertTierWithCreatedAt(t *testing.T, db *sqlx.DB, id, name string, isCurrent bool, createdAt string) {
	t.Helper()
	db.MustExec(`INSERT INTO raid_tiers (id, name, is_current, created_at) VALUES ($1, $2, $3, $4)`,
		id, name, isCurrent, createdAt)
}

func insertTierWithSortOrder(t *testing.T, db *sqlx.DB, id, name string, isCurrent bool, sortOrder int, createdAt string) {
	t.Helper()
	db.MustExec(`INSERT INTO raid_tiers (id, name, is_current, sort_order, created_at) VALUES ($1, $2, $3, $4, $5)`,
		id, name, isCurrent, sortOrder, createdAt)
}

func insertBoss(t *testing.T, db *sqlx.DB, id, tierID, name string, sortOrder int, killedAt *string) {
	t.Helper()
	db.MustExec(`INSERT INTO raid_bosses (id, tier_id, name, sort_order, killed_at) VALUES ($1, $2, $3, $4, $5)`,
		id, tierID, name, sortOrder, killedAt)
}

func TestGetRaidProgress_ReturnsCurrentTierWithBossesInOrder(t *testing.T) {
	// given a current tier with three bosses, one already killed, inserted out of order
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Molten Depths", true)
	insertBoss(t, db, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "11111111-1111-1111-1111-111111111111", "Ashveil", 2, nil)
	killedAt := "2026-03-01T20:00:00Z"
	insertBoss(t, db, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "11111111-1111-1111-1111-111111111111", "Grimjaw", 1, &killedAt)
	insertBoss(t, db, "cccccccc-cccc-cccc-cccc-cccccccccccc", "11111111-1111-1111-1111-111111111111", "Pyrelord", 3, nil)

	// when I ask for raid progress
	resp := srv.Get(t, "/api/raid-progress")

	// then I expect the current tier with bosses in sort order and a kill count
	resp.RequireStatus(t, http.StatusOK)
	var progress []progressJSON
	resp.DecodeJSON(t, &progress)

	want := []progressJSON{
		{
			Tier: struct {
				Name string `json:"name"`
			}{Name: "Molten Depths"},
			Bosses: []bossJSON{
				{ID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", Name: "Grimjaw", KilledAt: &killedAt},
				{ID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", Name: "Ashveil", KilledAt: nil},
				{ID: "cccccccc-cccc-cccc-cccc-cccccccccccc", Name: "Pyrelord", KilledAt: nil},
			},
			Killed: 1,
			Total:  3,
		},
	}
	if diff := cmp.Diff(want, progress); diff != "" {
		t.Errorf("unexpected progress (-want +got):\n%s", diff)
	}
}

func TestGetRaidProgress_ReturnsEveryCurrentTier_WhenMultipleAreCurrent(t *testing.T) {
	// given two current tiers (Forever ships at least two raids per tier) and one retired tier
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertTierWithCreatedAt(t, db, "11111111-1111-1111-1111-111111111111", "Barrow Deeps", true, "2026-01-01T00:00:00Z")
	insertTierWithCreatedAt(t, db, "22222222-2222-2222-2222-222222222222", "Onyxia's Lair", true, "2026-01-02T00:00:00Z")
	insertTierWithCreatedAt(t, db, "33333333-3333-3333-3333-333333333333", "Shattered Spire", false, "2025-12-01T00:00:00Z")
	insertBoss(t, db, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "22222222-2222-2222-2222-222222222222", "Onyxia", 0, nil)

	// when I ask for raid progress
	resp := srv.Get(t, "/api/raid-progress")

	// then I expect both current tiers, newest first, and the retired tier excluded
	resp.RequireStatus(t, http.StatusOK)
	var progress []progressJSON
	resp.DecodeJSON(t, &progress)

	want := []progressJSON{
		{
			Tier: struct {
				Name string `json:"name"`
			}{Name: "Onyxia's Lair"},
			Bosses: []bossJSON{{ID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", Name: "Onyxia", KilledAt: nil}},
			Killed: 0,
			Total:  1,
		},
		{
			Tier: struct {
				Name string `json:"name"`
			}{Name: "Barrow Deeps"},
			Bosses: []bossJSON{},
			Killed: 0,
			Total:  0,
		},
	}
	if diff := cmp.Diff(want, progress); diff != "" {
		t.Errorf("unexpected progress (-want +got):\n%s", diff)
	}
}

func TestGetRaidProgress_ReturnsNotFound_WhenNoTierIsCurrent(t *testing.T) {
	// given a tier that exists but is not current
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Molten Depths", false)

	// when I ask for raid progress
	resp := srv.Get(t, "/api/raid-progress")

	// then I expect a 404
	resp.RequireStatus(t, http.StatusNotFound)
	requireErrorBody(t, resp, "no current raid tier")
}

func TestGetRaidProgress_ReturnsNotFound_WhenNoTiersExist(t *testing.T) {
	// given no raid tiers have been created yet
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I ask for raid progress
	resp := srv.Get(t, "/api/raid-progress")

	// then I expect a 404
	resp.RequireStatus(t, http.StatusNotFound)
	requireErrorBody(t, resp, "no current raid tier")
}

func TestListRaidTiers_ReturnsTiersNewestFirstWithBosses(t *testing.T) {
	// given an older finished tier and a newer current tier, inserted oldest first
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertTierWithCreatedAt(t, db, "11111111-1111-1111-1111-111111111111", "Sunken Reliquary", false, "2026-01-01T00:00:00Z")
	insertTierWithCreatedAt(t, db, "22222222-2222-2222-2222-222222222222", "Molten Depths", true, "2026-03-01T00:00:00Z")
	insertBoss(t, db, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "22222222-2222-2222-2222-222222222222", "Grimjaw", 1, nil)
	killedAt := "2026-01-05T00:00:00Z"
	insertBoss(t, db, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "11111111-1111-1111-1111-111111111111", "Voidshard Sentinel", 1, &killedAt)

	// when I list raid tiers
	resp := srv.Get(t, "/api/raid-tiers")

	// then I expect the newer tier first, each with its own bosses
	resp.RequireStatus(t, http.StatusOK)
	var tiers []tierJSON
	resp.DecodeJSON(t, &tiers)

	want := []tierJSON{
		{
			ID:        "22222222-2222-2222-2222-222222222222",
			Name:      "Molten Depths",
			IsCurrent: true,
			Bosses:    []bossJSON{{ID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", Name: "Grimjaw", KilledAt: nil}},
		},
		{
			ID:        "11111111-1111-1111-1111-111111111111",
			Name:      "Sunken Reliquary",
			IsCurrent: false,
			Bosses:    []bossJSON{{ID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", Name: "Voidshard Sentinel", KilledAt: &killedAt}},
		},
	}
	if diff := cmp.Diff(want, tiers); diff != "" {
		t.Errorf("unexpected tiers (-want +got):\n%s", diff)
	}
}

func TestListRaidTiers_OrdersBySortOrder_RegardlessOfCreationOrder(t *testing.T) {
	// given a tier created later but given a lower sort_order than an older tier
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertTierWithSortOrder(t, db, "11111111-1111-1111-1111-111111111111", "Hyjal Summit", true, 1, "2026-01-01T00:00:00Z")
	insertTierWithSortOrder(t, db, "22222222-2222-2222-2222-222222222222", "Barrow Deeps", true, 0, "2026-03-01T00:00:00Z")

	// when I list raid tiers
	resp := srv.Get(t, "/api/raid-tiers")

	// then I expect the higher sort_order first, regardless of creation time
	resp.RequireStatus(t, http.StatusOK)
	var tiers []tierJSON
	resp.DecodeJSON(t, &tiers)

	names := make([]string, len(tiers))
	for i, tier := range tiers {
		names[i] = tier.Name
	}
	want := []string{"Hyjal Summit", "Barrow Deeps"}
	if diff := cmp.Diff(want, names); diff != "" {
		t.Errorf("unexpected tier order (-want +got):\n%s", diff)
	}
}

func TestGetRaidProgress_OrdersBySortOrder_RegardlessOfCreationOrder(t *testing.T) {
	// given a current tier created later but given a lower sort_order than another current tier
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertTierWithSortOrder(t, db, "11111111-1111-1111-1111-111111111111", "Hyjal Summit", true, 1, "2026-01-01T00:00:00Z")
	insertTierWithSortOrder(t, db, "22222222-2222-2222-2222-222222222222", "Barrow Deeps", true, 0, "2026-03-01T00:00:00Z")

	// when I ask for raid progress
	resp := srv.Get(t, "/api/raid-progress")

	// then I expect the higher sort_order first, regardless of creation time
	resp.RequireStatus(t, http.StatusOK)
	var progress []progressJSON
	resp.DecodeJSON(t, &progress)

	names := make([]string, len(progress))
	for i, p := range progress {
		names[i] = p.Tier.Name
	}
	want := []string{"Hyjal Summit", "Barrow Deeps"}
	if diff := cmp.Diff(want, names); diff != "" {
		t.Errorf("unexpected tier order (-want +got):\n%s", diff)
	}
}

func TestListRaidTiers_ReturnsEmptyArray_WhenNoTiersExist(t *testing.T) {
	// given no raid tiers exist
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I list raid tiers
	resp := srv.Get(t, "/api/raid-tiers")

	// then I expect a 200 with an empty JSON array rather than null
	resp.RequireStatus(t, http.StatusOK)
	if got := string(resp.Body); got != "[]\n" {
		t.Errorf("expected an empty JSON array, got %q", got)
	}
}

func TestCreateRaidTier_ReturnsCreatedTier_WhenOfficerSubmitsValidBody(t *testing.T) {
	// given I am logged in as an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I create a tier with two bosses
	resp := srv.Post(t, "/api/raid-tiers", map[string]any{
		"name":   "Molten Depths",
		"bosses": []string{"Grimjaw", "Ashveil"},
	})

	// then I expect a 201 with the tier not current and bosses in the given order
	resp.RequireStatus(t, http.StatusCreated)
	var created tierJSON
	resp.DecodeJSON(t, &created)

	want := tierJSON{
		ID:        created.ID,
		Name:      "Molten Depths",
		IsCurrent: false,
		Bosses: []bossJSON{
			{ID: created.Bosses[0].ID, Name: "Grimjaw", KilledAt: nil},
			{ID: created.Bosses[1].ID, Name: "Ashveil", KilledAt: nil},
		},
	}
	if diff := cmp.Diff(want, created); diff != "" {
		t.Errorf("unexpected created tier (-want +got):\n%s", diff)
	}
	if created.ID == "" || created.Bosses[0].ID == "" || created.Bosses[1].ID == "" {
		t.Errorf("expected the server to generate ids, got %+v", created)
	}
}

func TestCreateRaidTier_ReturnsCreatedTierWithGivenSortOrder(t *testing.T) {
	// given I am logged in as an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I create a tier with an explicit sort order
	resp := srv.Post(t, "/api/raid-tiers", map[string]any{"name": "Molten Depths", "sortOrder": 3})

	// then I expect the sort order stored and returned
	resp.RequireStatus(t, http.StatusCreated)
	var created tierJSON
	resp.DecodeJSON(t, &created)
	if created.SortOrder != 3 {
		t.Errorf("expected sortOrder 3, got %d", created.SortOrder)
	}
}

func TestCreateRaidTier_ReturnsCreatedTierWithoutBosses_WhenBossesOmitted(t *testing.T) {
	// given I am logged in as an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I create a tier without any bosses
	resp := srv.Post(t, "/api/raid-tiers", map[string]any{"name": "Molten Depths"})

	// then I expect a 201 with an empty boss list
	resp.RequireStatus(t, http.StatusCreated)
	var created tierJSON
	resp.DecodeJSON(t, &created)
	if len(created.Bosses) != 0 {
		t.Errorf("expected no bosses, got %+v", created.Bosses)
	}
}

func TestCreateRaidTier_ReturnsBadRequest_WhenNameIsEmpty(t *testing.T) {
	// given I am logged in as an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I create a tier with a blank name
	resp := srv.Post(t, "/api/raid-tiers", map[string]any{"name": "   "})

	// then I expect a 400 naming the name field
	resp.RequireStatus(t, http.StatusBadRequest)
	requireErrorBody(t, resp, "name: must not be empty")
}

func TestCreateRaidTier_ReturnsBadRequest_WhenABossNameIsEmpty(t *testing.T) {
	// given I am logged in as an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I create a tier whose second boss name is blank
	resp := srv.Post(t, "/api/raid-tiers", map[string]any{
		"name":   "Molten Depths",
		"bosses": []string{"Grimjaw", "  "},
	})

	// then I expect a 400 naming the bosses field
	resp.RequireStatus(t, http.StatusBadRequest)
	requireErrorBody(t, resp, "bosses: must not contain an empty name")
}

func TestCreateRaidTier_ReturnsUnauthorized_WhenCallerIsAnonymous(t *testing.T) {
	// given nobody is logged in
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I create a tier
	resp := srv.Post(t, "/api/raid-tiers", map[string]any{"name": "Molten Depths"})

	// then I expect a 401
	resp.RequireStatus(t, http.StatusUnauthorized)
}

func TestCreateRaidTier_ReturnsForbidden_WhenCallerIsNotAnOfficer(t *testing.T) {
	// given I am logged in as a regular member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "member-1", "Member")

	// when I create a tier
	resp := srv.Post(t, "/api/raid-tiers", map[string]any{"name": "Molten Depths"})

	// then I expect a 403
	resp.RequireStatus(t, http.StatusForbidden)
}

func TestSetCurrentTier_DoesNotClearOtherCurrentTiers(t *testing.T) {
	// given one already-current tier (Forever ships at least two raids per tier, so this is expected)
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Barrow Deeps", true)
	insertTier(t, db, "22222222-2222-2222-2222-222222222222", "Onyxia's Lair", false)

	// when I mark the second tier current too
	resp := srv.Patch(t, "/api/raid-tiers/22222222-2222-2222-2222-222222222222", map[string]any{"isCurrent": true})

	// then I expect both tiers current
	resp.RequireStatus(t, http.StatusOK)
	var updated tierJSON
	resp.DecodeJSON(t, &updated)
	want := tierJSON{ID: "22222222-2222-2222-2222-222222222222", Name: "Onyxia's Lair", IsCurrent: true, Bosses: []bossJSON{}}
	if diff := cmp.Diff(want, updated); diff != "" {
		t.Errorf("unexpected patched tier (-want +got):\n%s", diff)
	}

	var stillCurrent bool
	if err := db.Get(&stillCurrent, `SELECT is_current FROM raid_tiers WHERE id = '11111111-1111-1111-1111-111111111111'`); err != nil {
		t.Fatalf("check other tier: %v", err)
	}
	if !stillCurrent {
		t.Errorf("expected the other tier to remain current")
	}
}

func TestSetCurrentTier_AllowsNoCurrentTier_WhenSetToFalse(t *testing.T) {
	// given a current tier
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Molten Depths", true)

	// when I clear it without naming a replacement
	resp := srv.Patch(t, "/api/raid-tiers/11111111-1111-1111-1111-111111111111", map[string]any{"isCurrent": false})

	// then I expect it cleared, and the progress endpoint now 404s
	resp.RequireStatus(t, http.StatusOK)
	var updated tierJSON
	resp.DecodeJSON(t, &updated)
	if updated.IsCurrent {
		t.Errorf("expected the tier to no longer be current, got %+v", updated)
	}

	progressResp := srv.Get(t, "/api/raid-progress")
	progressResp.RequireStatus(t, http.StatusNotFound)
}

func TestSetCurrentTier_ReturnsNotFound_WhenTierDoesNotExist(t *testing.T) {
	// given I am logged in as an officer and no tiers exist
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I mark a nonexistent tier current
	resp := srv.Patch(t, "/api/raid-tiers/99999999-9999-9999-9999-999999999999", map[string]any{"isCurrent": true})

	// then I expect a 404
	resp.RequireStatus(t, http.StatusNotFound)
}

func TestSetCurrentTier_ReturnsForbidden_WhenCallerIsNotAnOfficer(t *testing.T) {
	// given a tier and a logged-in regular member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "member-1", "Member")
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Molten Depths", false)

	// when I mark it current
	resp := srv.Patch(t, "/api/raid-tiers/11111111-1111-1111-1111-111111111111", map[string]any{"isCurrent": true})

	// then I expect a 403
	resp.RequireStatus(t, http.StatusForbidden)
}

func TestSetCurrentTier_ReturnsUnauthorized_WhenCallerIsAnonymous(t *testing.T) {
	// given a tier and nobody logged in
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Molten Depths", false)

	// when I mark it current
	resp := srv.Patch(t, "/api/raid-tiers/11111111-1111-1111-1111-111111111111", map[string]any{"isCurrent": true})

	// then I expect a 401
	resp.RequireStatus(t, http.StatusUnauthorized)
}

func TestMarkBossKilled_SetsKilledAt_WhenOfficerRequests(t *testing.T) {
	// given an unkilled boss in a tier
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Molten Depths", true)
	insertBoss(t, db, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "11111111-1111-1111-1111-111111111111", "Grimjaw", 1, nil)

	// when I mark the boss killed
	resp := srv.Patch(t, "/api/raid-bosses/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", map[string]any{"killed": true})

	// then I expect a 200 with killedAt set to the current fixed time
	resp.RequireStatus(t, http.StatusOK)
	var updated bossJSON
	resp.DecodeJSON(t, &updated)

	fixedNow := "2026-03-14T20:00:00Z"
	want := bossJSON{ID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", Name: "Grimjaw", KilledAt: &fixedNow}
	if diff := cmp.Diff(want, updated); diff != "" {
		t.Errorf("unexpected boss (-want +got):\n%s", diff)
	}
}

func TestMarkBossKilled_ClearsKilledAt_WhenSetToFalse(t *testing.T) {
	// given an already-killed boss
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Molten Depths", true)
	killedAt := "2026-03-01T20:00:00Z"
	insertBoss(t, db, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "11111111-1111-1111-1111-111111111111", "Grimjaw", 1, &killedAt)

	// when I clear the kill
	resp := srv.Patch(t, "/api/raid-bosses/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", map[string]any{"killed": false})

	// then I expect a 200 with killedAt cleared
	resp.RequireStatus(t, http.StatusOK)
	var updated bossJSON
	resp.DecodeJSON(t, &updated)
	if updated.KilledAt != nil {
		t.Errorf("expected killedAt cleared, got %+v", updated)
	}
}

func TestMarkBossKilled_ReturnsNotFound_WhenBossDoesNotExist(t *testing.T) {
	// given I am logged in as an officer and no bosses exist
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I mark a nonexistent boss killed
	resp := srv.Patch(t, "/api/raid-bosses/99999999-9999-9999-9999-999999999999", map[string]any{"killed": true})

	// then I expect a 404
	resp.RequireStatus(t, http.StatusNotFound)
}

func TestMarkBossKilled_ReturnsForbidden_WhenCallerIsNotAnOfficer(t *testing.T) {
	// given a boss and a logged-in regular member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "member-1", "Member")
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Molten Depths", true)
	insertBoss(t, db, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "11111111-1111-1111-1111-111111111111", "Grimjaw", 1, nil)

	// when I mark the boss killed
	resp := srv.Patch(t, "/api/raid-bosses/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", map[string]any{"killed": true})

	// then I expect a 403
	resp.RequireStatus(t, http.StatusForbidden)
}

func TestMarkBossKilled_ReturnsUnauthorized_WhenCallerIsAnonymous(t *testing.T) {
	// given a boss and nobody logged in
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Molten Depths", true)
	insertBoss(t, db, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "11111111-1111-1111-1111-111111111111", "Grimjaw", 1, nil)

	// when I mark the boss killed
	resp := srv.Patch(t, "/api/raid-bosses/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", map[string]any{"killed": true})

	// then I expect a 401
	resp.RequireStatus(t, http.StatusUnauthorized)
}

func countRows(t *testing.T, db *sqlx.DB, table string) int {
	t.Helper()
	var count int
	if err := db.Get(&count, `SELECT count(*) FROM `+table); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return count
}

func TestDeleteRaidTier_DeletesTierAndItsBosses_WhenTierIsNotCurrent(t *testing.T) {
	// given a retired tier with a boss
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Shattered Spire", false)
	insertBoss(t, db, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "11111111-1111-1111-1111-111111111111", "Voidshard Sentinel", 0, nil)

	// when I delete the tier
	resp := srv.Delete(t, "/api/raid-tiers/11111111-1111-1111-1111-111111111111")

	// then I expect a 204 with the tier and its boss both gone
	resp.RequireStatus(t, http.StatusNoContent)
	if got := countRows(t, db, "raid_tiers"); got != 0 {
		t.Errorf("expected the tier to be deleted, found %d rows", got)
	}
	if got := countRows(t, db, "raid_bosses"); got != 0 {
		t.Errorf("expected the tier's bosses to be deleted, found %d rows", got)
	}
}

func TestDeleteRaidTier_ReturnsConflict_WhenTierIsCurrent(t *testing.T) {
	// given a current tier
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Molten Depths", true)

	// when I delete the tier
	resp := srv.Delete(t, "/api/raid-tiers/11111111-1111-1111-1111-111111111111")

	// then I expect a 409 and the tier still there
	resp.RequireStatus(t, http.StatusConflict)
	requireErrorBody(t, resp, "raid tier is current; clear its current flag before deleting it")
	if got := countRows(t, db, "raid_tiers"); got != 1 {
		t.Errorf("expected the current tier to remain, found %d rows", got)
	}
}

func TestDeleteRaidTier_ReturnsNotFound_WhenTierDoesNotExist(t *testing.T) {
	// given I am logged in as an officer and no tiers exist
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I delete a nonexistent tier
	resp := srv.Delete(t, "/api/raid-tiers/99999999-9999-9999-9999-999999999999")

	// then I expect a 404
	resp.RequireStatus(t, http.StatusNotFound)
	requireErrorBody(t, resp, "raid tier not found")
}

func TestDeleteRaidTier_ReturnsForbidden_WhenCallerIsNotAnOfficer(t *testing.T) {
	// given a retired tier and a logged-in regular member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "member-1", "Member")
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Shattered Spire", false)

	// when I delete the tier
	resp := srv.Delete(t, "/api/raid-tiers/11111111-1111-1111-1111-111111111111")

	// then I expect a 403
	resp.RequireStatus(t, http.StatusForbidden)
}

func TestDeleteRaidTier_ReturnsUnauthorized_WhenCallerIsAnonymous(t *testing.T) {
	// given a retired tier and nobody logged in
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Shattered Spire", false)

	// when I delete the tier
	resp := srv.Delete(t, "/api/raid-tiers/11111111-1111-1111-1111-111111111111")

	// then I expect a 401
	resp.RequireStatus(t, http.StatusUnauthorized)
}

func TestDeleteRaidBoss_DeletesOnlyThatBoss(t *testing.T) {
	// given a current tier with two bosses
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Molten Depths", true)
	insertBoss(t, db, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "11111111-1111-1111-1111-111111111111", "Grimjaw", 0, nil)
	insertBoss(t, db, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "11111111-1111-1111-1111-111111111111", "Ashveil", 1, nil)

	// when I delete the first boss
	resp := srv.Delete(t, "/api/raid-bosses/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

	// then I expect a 204 and only the other boss left in the tier
	resp.RequireStatus(t, http.StatusNoContent)
	var remaining []string
	if err := db.Select(&remaining, `SELECT name FROM raid_bosses ORDER BY sort_order`); err != nil {
		t.Fatalf("select remaining bosses: %v", err)
	}
	if diff := cmp.Diff([]string{"Ashveil"}, remaining); diff != "" {
		t.Errorf("unexpected remaining bosses (-want +got):\n%s", diff)
	}
}

func TestDeleteRaidBoss_ReturnsNotFound_WhenBossDoesNotExist(t *testing.T) {
	// given I am logged in as an officer and no bosses exist
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I delete a nonexistent boss
	resp := srv.Delete(t, "/api/raid-bosses/99999999-9999-9999-9999-999999999999")

	// then I expect a 404
	resp.RequireStatus(t, http.StatusNotFound)
	requireErrorBody(t, resp, "raid boss not found")
}

func TestDeleteRaidBoss_ReturnsForbidden_WhenCallerIsNotAnOfficer(t *testing.T) {
	// given a boss and a logged-in regular member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "member-1", "Member")
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Molten Depths", true)
	insertBoss(t, db, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "11111111-1111-1111-1111-111111111111", "Grimjaw", 0, nil)

	// when I delete the boss
	resp := srv.Delete(t, "/api/raid-bosses/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

	// then I expect a 403
	resp.RequireStatus(t, http.StatusForbidden)
}

func TestDeleteRaidBoss_ReturnsUnauthorized_WhenCallerIsAnonymous(t *testing.T) {
	// given a boss and nobody logged in
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Molten Depths", true)
	insertBoss(t, db, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "11111111-1111-1111-1111-111111111111", "Grimjaw", 0, nil)

	// when I delete the boss
	resp := srv.Delete(t, "/api/raid-bosses/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

	// then I expect a 401
	resp.RequireStatus(t, http.StatusUnauthorized)
}

func tierNamesInListOrder(t *testing.T, srv *testutil.Server) []string {
	t.Helper()
	resp := srv.Get(t, "/api/raid-tiers")
	resp.RequireStatus(t, http.StatusOK)
	var tiers []tierJSON
	resp.DecodeJSON(t, &tiers)
	names := make([]string, len(tiers))
	for i, tier := range tiers {
		names[i] = tier.Name
	}
	return names
}

type bossRow struct {
	Name      string  `db:"name"`
	SortOrder int     `db:"sort_order"`
	KilledAt  *string `db:"killed_at"`
}

func bossRowsInOrder(t *testing.T, db *sqlx.DB, tierID string) []bossRow {
	t.Helper()
	var rows []bossRow
	const query = `SELECT name, sort_order, to_char(killed_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS killed_at
		FROM raid_bosses WHERE tier_id = $1 ORDER BY sort_order`
	if err := db.Select(&rows, query, tierID); err != nil {
		t.Fatalf("select bosses of tier %s: %v", tierID, err)
	}
	return rows
}

func TestRenameRaidTier_ReturnsRenamedTier_KeepingItsCurrentFlag(t *testing.T) {
	// given a current tier with a misspelled name
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Molten Depthz", true)

	// when I rename it with surrounding whitespace
	resp := srv.Patch(t, "/api/raid-tiers/11111111-1111-1111-1111-111111111111", map[string]any{"name": "  Molten Depths  "})

	// then I expect the trimmed name and the tier still current
	resp.RequireStatus(t, http.StatusOK)
	var updated tierJSON
	resp.DecodeJSON(t, &updated)
	want := tierJSON{ID: "11111111-1111-1111-1111-111111111111", Name: "Molten Depths", IsCurrent: true, Bosses: []bossJSON{}}
	if diff := cmp.Diff(want, updated); diff != "" {
		t.Errorf("unexpected renamed tier (-want +got):\n%s", diff)
	}
}

func TestRenameRaidTier_ReturnsBadRequest_WhenNameIsEmpty(t *testing.T) {
	// given a tier
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Molten Depths", false)

	// when I rename it to a blank name
	resp := srv.Patch(t, "/api/raid-tiers/11111111-1111-1111-1111-111111111111", map[string]any{"name": "   "})

	// then I expect a 400 and the name unchanged
	resp.RequireStatus(t, http.StatusBadRequest)
	requireErrorBody(t, resp, "name: must not be empty")
	if diff := cmp.Diff([]string{"Molten Depths"}, tierNamesInListOrder(t, srv)); diff != "" {
		t.Errorf("unexpected tier names (-want +got):\n%s", diff)
	}
}

func TestUpdateRaidTier_ReturnsBadRequest_WhenBodySetsNoField(t *testing.T) {
	// given a tier
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Molten Depths", false)

	// when I patch it with an empty body
	resp := srv.Patch(t, "/api/raid-tiers/11111111-1111-1111-1111-111111111111", map[string]any{})

	// then I expect a 400 asking for at least one field
	resp.RequireStatus(t, http.StatusBadRequest)
	requireErrorBody(t, resp, "body: must set name or isCurrent")
}

func TestRenameRaidTier_ReturnsNotFound_WhenTierDoesNotExist(t *testing.T) {
	// given I am logged in as an officer and no tiers exist
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I rename a nonexistent tier
	resp := srv.Patch(t, "/api/raid-tiers/99999999-9999-9999-9999-999999999999", map[string]any{"name": "Molten Depths"})

	// then I expect a 404
	resp.RequireStatus(t, http.StatusNotFound)
	requireErrorBody(t, resp, "raid tier not found")
}

func TestReorderRaidTiers_ListsTiersInTheGivenOrder(t *testing.T) {
	// given three tiers listed newest first
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	insertTierWithSortOrder(t, db, "11111111-1111-1111-1111-111111111111", "Barrow Deeps", false, 0, "2026-01-01T00:00:00Z")
	insertTierWithSortOrder(t, db, "22222222-2222-2222-2222-222222222222", "Hyjal Summit", true, 1, "2026-02-01T00:00:00Z")
	insertTierWithSortOrder(t, db, "33333333-3333-3333-3333-333333333333", "Onyxia's Lair", true, 2, "2026-03-01T00:00:00Z")

	// when I put them in a new order
	resp := srv.Put(t, "/api/raid-tiers/order", map[string]any{"ids": []string{
		"22222222-2222-2222-2222-222222222222",
		"11111111-1111-1111-1111-111111111111",
		"33333333-3333-3333-3333-333333333333",
	}})

	// then I expect a 204 and the tier list in exactly that order
	resp.RequireStatus(t, http.StatusNoContent)
	want := []string{"Hyjal Summit", "Barrow Deeps", "Onyxia's Lair"}
	if diff := cmp.Diff(want, tierNamesInListOrder(t, srv)); diff != "" {
		t.Errorf("unexpected tier order (-want +got):\n%s", diff)
	}
}

func TestReorderRaidTiers_ReturnsBadRequestAndChangesNothing_WhenListIsIncomplete(t *testing.T) {
	// given two tiers
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	insertTierWithSortOrder(t, db, "11111111-1111-1111-1111-111111111111", "Barrow Deeps", false, 0, "2026-01-01T00:00:00Z")
	insertTierWithSortOrder(t, db, "22222222-2222-2222-2222-222222222222", "Hyjal Summit", true, 1, "2026-02-01T00:00:00Z")

	// when I reorder with only one of them
	resp := srv.Put(t, "/api/raid-tiers/order", map[string]any{"ids": []string{"11111111-1111-1111-1111-111111111111"}})

	// then I expect a 400 and the original order kept
	resp.RequireStatus(t, http.StatusBadRequest)
	requireErrorBody(t, resp, "ids: must list every raid exactly once")
	if diff := cmp.Diff([]string{"Hyjal Summit", "Barrow Deeps"}, tierNamesInListOrder(t, srv)); diff != "" {
		t.Errorf("unexpected tier order (-want +got):\n%s", diff)
	}
}

func TestReorderRaidTiers_ReturnsBadRequestAndChangesNothing_WhenListHasAStaleID(t *testing.T) {
	// given two tiers
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	insertTierWithSortOrder(t, db, "11111111-1111-1111-1111-111111111111", "Barrow Deeps", false, 0, "2026-01-01T00:00:00Z")
	insertTierWithSortOrder(t, db, "22222222-2222-2222-2222-222222222222", "Hyjal Summit", true, 1, "2026-02-01T00:00:00Z")

	// when I reorder with a list naming a since-deleted tier in place of one that exists
	resp := srv.Put(t, "/api/raid-tiers/order", map[string]any{"ids": []string{
		"11111111-1111-1111-1111-111111111111",
		"99999999-9999-9999-9999-999999999999",
	}})

	// then I expect a 400 and the original order kept
	resp.RequireStatus(t, http.StatusBadRequest)
	requireErrorBody(t, resp, "ids: must list every raid exactly once")
	if diff := cmp.Diff([]string{"Hyjal Summit", "Barrow Deeps"}, tierNamesInListOrder(t, srv)); diff != "" {
		t.Errorf("unexpected tier order (-want +got):\n%s", diff)
	}
}

func TestReorderRaidTiers_ReturnsForbidden_WhenCallerIsNotAnOfficer(t *testing.T) {
	// given a tier and a logged-in regular member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "member-1", "Member")
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Molten Depths", false)

	// when I reorder the tiers
	resp := srv.Put(t, "/api/raid-tiers/order", map[string]any{"ids": []string{"11111111-1111-1111-1111-111111111111"}})

	// then I expect a 403
	resp.RequireStatus(t, http.StatusForbidden)
}

func TestReorderRaidTiers_ReturnsUnauthorized_WhenCallerIsAnonymous(t *testing.T) {
	// given a tier and nobody logged in
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Molten Depths", false)

	// when I reorder the tiers
	resp := srv.Put(t, "/api/raid-tiers/order", map[string]any{"ids": []string{"11111111-1111-1111-1111-111111111111"}})

	// then I expect a 401
	resp.RequireStatus(t, http.StatusUnauthorized)
}

func TestAddRaidBoss_ReturnsCreatedBossAddedLast(t *testing.T) {
	// given a tier with two bosses
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Molten Depths", true)
	insertBoss(t, db, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "11111111-1111-1111-1111-111111111111", "Grimjaw", 0, nil)
	insertBoss(t, db, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "11111111-1111-1111-1111-111111111111", "Ashveil", 1, nil)

	// when I add a boss with surrounding whitespace
	resp := srv.Post(t, "/api/raid-tiers/11111111-1111-1111-1111-111111111111/bosses", map[string]any{"name": " Pyrelord "})

	// then I expect a 201 with the trimmed, unkilled boss, placed after the existing bosses
	resp.RequireStatus(t, http.StatusCreated)
	var added bossJSON
	resp.DecodeJSON(t, &added)
	if diff := cmp.Diff(bossJSON{ID: added.ID, Name: "Pyrelord", KilledAt: nil}, added); diff != "" {
		t.Errorf("unexpected added boss (-want +got):\n%s", diff)
	}
	if added.ID == "" {
		t.Errorf("expected the server to generate an id")
	}

	want := []bossRow{
		{Name: "Grimjaw", SortOrder: 0},
		{Name: "Ashveil", SortOrder: 1},
		{Name: "Pyrelord", SortOrder: 2},
	}
	if diff := cmp.Diff(want, bossRowsInOrder(t, db, "11111111-1111-1111-1111-111111111111")); diff != "" {
		t.Errorf("unexpected bosses (-want +got):\n%s", diff)
	}
}

func TestAddRaidBoss_StartsAtSortOrderZero_WhenTierHasNoBosses(t *testing.T) {
	// given a tier with no bosses
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Molten Depths", true)

	// when I add a boss
	resp := srv.Post(t, "/api/raid-tiers/11111111-1111-1111-1111-111111111111/bosses", map[string]any{"name": "Grimjaw"})

	// then I expect it stored as the first boss
	resp.RequireStatus(t, http.StatusCreated)
	want := []bossRow{{Name: "Grimjaw", SortOrder: 0}}
	if diff := cmp.Diff(want, bossRowsInOrder(t, db, "11111111-1111-1111-1111-111111111111")); diff != "" {
		t.Errorf("unexpected bosses (-want +got):\n%s", diff)
	}
}

func TestAddRaidBoss_ReturnsBadRequest_WhenNameIsEmpty(t *testing.T) {
	// given a tier
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Molten Depths", true)

	// when I add a boss with a blank name
	resp := srv.Post(t, "/api/raid-tiers/11111111-1111-1111-1111-111111111111/bosses", map[string]any{"name": "  "})

	// then I expect a 400 and no boss stored
	resp.RequireStatus(t, http.StatusBadRequest)
	requireErrorBody(t, resp, "name: must not be empty")
	if got := countRows(t, db, "raid_bosses"); got != 0 {
		t.Errorf("expected no bosses, found %d rows", got)
	}
}

func TestAddRaidBoss_ReturnsNotFound_WhenTierDoesNotExist(t *testing.T) {
	// given I am logged in as an officer and no tiers exist
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I add a boss to a nonexistent tier
	resp := srv.Post(t, "/api/raid-tiers/99999999-9999-9999-9999-999999999999/bosses", map[string]any{"name": "Grimjaw"})

	// then I expect a 404
	resp.RequireStatus(t, http.StatusNotFound)
	requireErrorBody(t, resp, "raid tier not found")
}

func TestAddRaidBoss_ReturnsForbidden_WhenCallerIsNotAnOfficer(t *testing.T) {
	// given a tier and a logged-in regular member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "member-1", "Member")
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Molten Depths", true)

	// when I add a boss
	resp := srv.Post(t, "/api/raid-tiers/11111111-1111-1111-1111-111111111111/bosses", map[string]any{"name": "Grimjaw"})

	// then I expect a 403
	resp.RequireStatus(t, http.StatusForbidden)
}

func TestAddRaidBoss_ReturnsUnauthorized_WhenCallerIsAnonymous(t *testing.T) {
	// given a tier and nobody logged in
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Molten Depths", true)

	// when I add a boss
	resp := srv.Post(t, "/api/raid-tiers/11111111-1111-1111-1111-111111111111/bosses", map[string]any{"name": "Grimjaw"})

	// then I expect a 401
	resp.RequireStatus(t, http.StatusUnauthorized)
}

func TestRenameRaidBoss_ReturnsRenamedBoss_KeepingItsKill(t *testing.T) {
	// given a killed boss with a misspelled name
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Molten Depths", true)
	killedAt := "2026-03-01T20:00:00Z"
	insertBoss(t, db, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "11111111-1111-1111-1111-111111111111", "Grimjow", 0, &killedAt)

	// when I rename it
	resp := srv.Patch(t, "/api/raid-bosses/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", map[string]any{"name": " Grimjaw "})

	// then I expect the trimmed name and the original kill time
	resp.RequireStatus(t, http.StatusOK)
	var updated bossJSON
	resp.DecodeJSON(t, &updated)
	want := bossJSON{ID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", Name: "Grimjaw", KilledAt: &killedAt}
	if diff := cmp.Diff(want, updated); diff != "" {
		t.Errorf("unexpected renamed boss (-want +got):\n%s", diff)
	}
}

func TestRenameRaidBoss_ReturnsBadRequest_WhenNameIsEmpty(t *testing.T) {
	// given a boss
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Molten Depths", true)
	insertBoss(t, db, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "11111111-1111-1111-1111-111111111111", "Grimjaw", 0, nil)

	// when I rename it to a blank name
	resp := srv.Patch(t, "/api/raid-bosses/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", map[string]any{"name": ""})

	// then I expect a 400 and the name unchanged
	resp.RequireStatus(t, http.StatusBadRequest)
	requireErrorBody(t, resp, "name: must not be empty")
	want := []bossRow{{Name: "Grimjaw", SortOrder: 0}}
	if diff := cmp.Diff(want, bossRowsInOrder(t, db, "11111111-1111-1111-1111-111111111111")); diff != "" {
		t.Errorf("unexpected bosses (-want +got):\n%s", diff)
	}
}

func TestUpdateRaidBoss_ReturnsBadRequest_WhenBodySetsNoField(t *testing.T) {
	// given a boss
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Molten Depths", true)
	insertBoss(t, db, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "11111111-1111-1111-1111-111111111111", "Grimjaw", 0, nil)

	// when I patch it with an empty body
	resp := srv.Patch(t, "/api/raid-bosses/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", map[string]any{})

	// then I expect a 400 asking for at least one field
	resp.RequireStatus(t, http.StatusBadRequest)
	requireErrorBody(t, resp, "body: must set name or killed")
}

func TestRenameRaidBoss_ReturnsNotFound_WhenBossDoesNotExist(t *testing.T) {
	// given I am logged in as an officer and no bosses exist
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I rename a nonexistent boss
	resp := srv.Patch(t, "/api/raid-bosses/99999999-9999-9999-9999-999999999999", map[string]any{"name": "Grimjaw"})

	// then I expect a 404
	resp.RequireStatus(t, http.StatusNotFound)
	requireErrorBody(t, resp, "raid boss not found")
}

func TestReorderRaidBosses_StoresTheGivenOrder_WithoutTrippingTheUniqueSortOrder(t *testing.T) {
	// given a tier with three bosses at sort orders 0..2, the middle one killed
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Molten Depths", true)
	killedAt := "2026-03-01T20:00:00Z"
	insertBoss(t, db, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "11111111-1111-1111-1111-111111111111", "Grimjaw", 0, nil)
	insertBoss(t, db, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "11111111-1111-1111-1111-111111111111", "Ashveil", 1, &killedAt)
	insertBoss(t, db, "cccccccc-cccc-cccc-cccc-cccccccccccc", "11111111-1111-1111-1111-111111111111", "Pyrelord", 2, nil)

	// when I reverse them, so each boss takes a sort order another boss holds
	resp := srv.Put(t, "/api/raid-tiers/11111111-1111-1111-1111-111111111111/bosses/order", map[string]any{"ids": []string{
		"cccccccc-cccc-cccc-cccc-cccccccccccc",
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	}})

	// then I expect a 204 and the bosses in the new order with their kill state kept
	resp.RequireStatus(t, http.StatusNoContent)
	want := []bossRow{
		{Name: "Pyrelord", SortOrder: 0},
		{Name: "Ashveil", SortOrder: 1, KilledAt: &killedAt},
		{Name: "Grimjaw", SortOrder: 2},
	}
	if diff := cmp.Diff(want, bossRowsInOrder(t, db, "11111111-1111-1111-1111-111111111111")); diff != "" {
		t.Errorf("unexpected bosses (-want +got):\n%s", diff)
	}
}

func TestReorderRaidBosses_ReturnsBadRequestAndChangesNothing_WhenListIsIncomplete(t *testing.T) {
	// given a tier with two bosses
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Molten Depths", true)
	insertBoss(t, db, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "11111111-1111-1111-1111-111111111111", "Grimjaw", 0, nil)
	insertBoss(t, db, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "11111111-1111-1111-1111-111111111111", "Ashveil", 1, nil)

	// when I reorder with only one of them
	resp := srv.Put(t, "/api/raid-tiers/11111111-1111-1111-1111-111111111111/bosses/order", map[string]any{"ids": []string{
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
	}})

	// then I expect a 400 and the original order kept
	resp.RequireStatus(t, http.StatusBadRequest)
	requireErrorBody(t, resp, "ids: must list every boss in the raid exactly once")
	want := []bossRow{{Name: "Grimjaw", SortOrder: 0}, {Name: "Ashveil", SortOrder: 1}}
	if diff := cmp.Diff(want, bossRowsInOrder(t, db, "11111111-1111-1111-1111-111111111111")); diff != "" {
		t.Errorf("unexpected bosses (-want +got):\n%s", diff)
	}
}

func TestReorderRaidBosses_ReturnsBadRequestAndChangesNothing_WhenListHasABossFromAnotherTier(t *testing.T) {
	// given two tiers with one boss each
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Molten Depths", true)
	insertTier(t, db, "22222222-2222-2222-2222-222222222222", "Shattered Spire", false)
	insertBoss(t, db, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "11111111-1111-1111-1111-111111111111", "Grimjaw", 0, nil)
	insertBoss(t, db, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "22222222-2222-2222-2222-222222222222", "Voidshard Sentinel", 0, nil)

	// when I reorder the first tier's bosses using the other tier's boss
	resp := srv.Put(t, "/api/raid-tiers/11111111-1111-1111-1111-111111111111/bosses/order", map[string]any{"ids": []string{
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
	}})

	// then I expect a 400 and neither tier's bosses changed
	resp.RequireStatus(t, http.StatusBadRequest)
	requireErrorBody(t, resp, "ids: must list every boss in the raid exactly once")
	if diff := cmp.Diff([]bossRow{{Name: "Grimjaw", SortOrder: 0}}, bossRowsInOrder(t, db, "11111111-1111-1111-1111-111111111111")); diff != "" {
		t.Errorf("unexpected bosses in first tier (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]bossRow{{Name: "Voidshard Sentinel", SortOrder: 0}}, bossRowsInOrder(t, db, "22222222-2222-2222-2222-222222222222")); diff != "" {
		t.Errorf("unexpected bosses in second tier (-want +got):\n%s", diff)
	}
}

func TestReorderRaidBosses_ReturnsNotFound_WhenTierDoesNotExist(t *testing.T) {
	// given I am logged in as an officer and no tiers exist
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I reorder the bosses of a nonexistent tier
	resp := srv.Put(t, "/api/raid-tiers/99999999-9999-9999-9999-999999999999/bosses/order", map[string]any{"ids": []string{}})

	// then I expect a 404
	resp.RequireStatus(t, http.StatusNotFound)
	requireErrorBody(t, resp, "raid tier not found")
}

func TestReorderRaidBosses_ReturnsForbidden_WhenCallerIsNotAnOfficer(t *testing.T) {
	// given a tier with a boss and a logged-in regular member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "member-1", "Member")
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Molten Depths", true)
	insertBoss(t, db, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "11111111-1111-1111-1111-111111111111", "Grimjaw", 0, nil)

	// when I reorder the bosses
	resp := srv.Put(t, "/api/raid-tiers/11111111-1111-1111-1111-111111111111/bosses/order", map[string]any{"ids": []string{
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	}})

	// then I expect a 403
	resp.RequireStatus(t, http.StatusForbidden)
}

func TestReorderRaidBosses_ReturnsUnauthorized_WhenCallerIsAnonymous(t *testing.T) {
	// given a tier with a boss and nobody logged in
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertTier(t, db, "11111111-1111-1111-1111-111111111111", "Molten Depths", true)
	insertBoss(t, db, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "11111111-1111-1111-1111-111111111111", "Grimjaw", 0, nil)

	// when I reorder the bosses
	resp := srv.Put(t, "/api/raid-tiers/11111111-1111-1111-1111-111111111111/bosses/order", map[string]any{"ids": []string{
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	}})

	// then I expect a 401
	resp.RequireStatus(t, http.StatusUnauthorized)
}
