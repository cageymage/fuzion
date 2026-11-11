package rankrequests_test

import (
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/cageymage/fuzion/backend/internal/testutil"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.Run(m))
}

const (
	tankardID = "11111111-1111-1111-1111-111111111111"
	healbotID = "22222222-2222-2222-2222-222222222222"
	openID    = "aaaaaaaa-0000-0000-0000-000000000001"
	doneID    = "aaaaaaaa-0000-0000-0000-000000000002"
	declineID = "aaaaaaaa-0000-0000-0000-000000000003"
	cancelID  = "aaaaaaaa-0000-0000-0000-000000000004"
	laterID   = "aaaaaaaa-0000-0000-0000-000000000005"
)

type requestJSON struct {
	ID                     string  `json:"id"`
	CharacterID            string  `json:"characterId"`
	CharacterName          string  `json:"characterName"`
	CharacterSecondaryName string  `json:"characterSecondaryName"`
	CharacterClass         string  `json:"characterClass"`
	Direction              string  `json:"direction"`
	Status                 string  `json:"status"`
	Note                   string  `json:"note"`
	ResolutionNote         string  `json:"resolutionNote"`
	RequestedBy            string  `json:"requestedBy"`
	CreatedAt              string  `json:"createdAt"`
	ResolvedAt             *string `json:"resolvedAt"`
	ResolvedBy             *string `json:"resolvedBy"`
}

type storedRequest struct {
	ID             string  `db:"id"`
	Status         string  `db:"status"`
	ResolutionNote string  `db:"resolution_note"`
	ResolvedAt     *string `db:"resolved_at"`
	ResolvedBy     *string `db:"resolved_by"`
}

func storedRequests(t *testing.T, db *sqlx.DB) []storedRequest {
	t.Helper()
	var rows []storedRequest
	err := db.Select(&rows, `
		SELECT id, status, resolution_note,
		       to_char(resolved_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS resolved_at,
		       resolved_by::text AS resolved_by
		FROM rank_change_requests ORDER BY id`)
	if err != nil {
		t.Fatalf("select rank_change_requests: %v", err)
	}
	return rows
}

func insertCharacter(t *testing.T, db *sqlx.DB, id, name, surname, class string, isMain bool, owner *uuid.UUID) {
	t.Helper()
	db.MustExec(`
		INSERT INTO characters (id, name, secondary_name, class, is_main, guild_rank, user_id)
		VALUES ($1, $2, $3, $4, $5, 6, $6)`, id, name, surname, class, isMain, owner)
}

func insertRequest(t *testing.T, db *sqlx.DB, id, characterID string, requestedBy uuid.UUID, direction, status, createdAt string, resolvedAt *string) {
	t.Helper()
	db.MustExec(`
		INSERT INTO rank_change_requests (id, character_id, requested_by, direction, status, note, created_at, resolved_at)
		VALUES ($1, $2, $3, $4, $5, 'please', $6, $7)`, id, characterID, requestedBy, direction, status, createdAt, resolvedAt)
}

func strPtr(s string) *string { return &s }

func requireBody(t *testing.T, resp testutil.Response, want string) {
	t.Helper()
	if diff := cmp.Diff(want+"\n", string(resp.Body)); diff != "" {
		t.Errorf("unexpected body (-want +got):\n%s", diff)
	}
}

func decodeCreated(t *testing.T, resp testutil.Response) requestJSON {
	t.Helper()
	var got requestJSON
	resp.DecodeJSON(t, &got)
	if _, err := uuid.Parse(got.ID); err != nil {
		t.Errorf("expected a generated id, got %q", got.ID)
	}
	got.ID = ""
	return got
}

func TestCreateRankChangeRequest_AsksForAnAltRank_WhenTheCharacterIsAMain(t *testing.T) {
	// given my main
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	me := srv.LoginAs(t, "80351110224678912", "thundermane")
	insertCharacter(t, db, tankardID, "Tankard", "Ironhide", "Warrior", true, &me.ID)

	// when I ask for its rank to be changed, with stray whitespace around the note
	resp := srv.Post(t, "/api/me/characters/"+tankardID+"/rank-requests", map[string]any{"note": "  Levelling a new main  "})

	// then I expect an open request to move it to an alt rank
	resp.RequireStatus(t, http.StatusCreated)
	want := requestJSON{
		CharacterID: tankardID, CharacterName: "Tankard", CharacterSecondaryName: "Ironhide", CharacterClass: "Warrior",
		Direction: "to_alt", Status: "open", Note: "Levelling a new main", RequestedBy: "thundermane",
		CreatedAt: "2026-03-14T20:00:00Z",
	}
	if diff := cmp.Diff(want, decodeCreated(t, resp)); diff != "" {
		t.Errorf("unexpected request (-want +got):\n%s", diff)
	}
}

func TestCreateRankChangeRequest_AsksForAMainRank_WhenTheCharacterIsAnAlt(t *testing.T) {
	// given my alt
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	me := srv.LoginAs(t, "80351110224678912", "thundermane")
	insertCharacter(t, db, healbotID, "Healbot", "Lightward", "Priest", false, &me.ID)

	// when I ask for its rank to be changed, without a note
	resp := srv.Post(t, "/api/me/characters/"+healbotID+"/rank-requests", map[string]any{})

	// then I expect an open request to move it to a main rank
	resp.RequireStatus(t, http.StatusCreated)
	want := requestJSON{
		CharacterID: healbotID, CharacterName: "Healbot", CharacterSecondaryName: "Lightward", CharacterClass: "Priest",
		Direction: "to_main", Status: "open", RequestedBy: "thundermane", CreatedAt: "2026-03-14T20:00:00Z",
	}
	if diff := cmp.Diff(want, decodeCreated(t, resp)); diff != "" {
		t.Errorf("unexpected request (-want +got):\n%s", diff)
	}
}

func TestCreateRankChangeRequest_ReturnsConflict_WhenOpenRequestExists(t *testing.T) {
	// given my main with an open request already
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	me := srv.LoginAs(t, "80351110224678912", "thundermane")
	insertCharacter(t, db, tankardID, "Tankard", "Ironhide", "Warrior", true, &me.ID)
	insertRequest(t, db, openID, tankardID, me.ID, "to_alt", "open", "2026-03-01T12:00:00Z", nil)

	// when I ask again
	resp := srv.Post(t, "/api/me/characters/"+tankardID+"/rank-requests", map[string]any{"note": "again"})

	// then I expect a 409 and still only the first request
	resp.RequireStatus(t, http.StatusConflict)
	requireBody(t, resp, `{"error":"this character already has an open rank request"}`)
	want := []storedRequest{{ID: openID, Status: "open"}}
	if diff := cmp.Diff(want, storedRequests(t, db)); diff != "" {
		t.Errorf("unexpected requests (-want +got):\n%s", diff)
	}
}

func TestCreateRankChangeRequest_ReturnsForbidden_WhenNotOwner(t *testing.T) {
	// given another member's main
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	other := srv.LoginAs(t, "11111111111111111", "someoneelse")
	srv.LoginAs(t, "80351110224678912", "thundermane")
	insertCharacter(t, db, tankardID, "Tankard", "Ironhide", "Warrior", true, &other.ID)

	// when I ask for its rank to be changed
	resp := srv.Post(t, "/api/me/characters/"+tankardID+"/rank-requests", map[string]any{})

	// then I expect a 403 and no request stored
	resp.RequireStatus(t, http.StatusForbidden)
	requireBody(t, resp, `{"error":"character is not yours"}`)
	if diff := cmp.Diff([]storedRequest(nil), storedRequests(t, db)); diff != "" {
		t.Errorf("unexpected requests (-want +got):\n%s", diff)
	}
}

func TestCreateRankChangeRequest_ReturnsNotFound_WhenTheCharacterDoesNotExist(t *testing.T) {
	// given a logged-in member and no such character
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "80351110224678912", "thundermane")

	// when I ask for its rank to be changed
	resp := srv.Post(t, "/api/me/characters/"+tankardID+"/rank-requests", map[string]any{})

	// then I expect a 404
	resp.RequireStatus(t, http.StatusNotFound)
	requireBody(t, resp, `{"error":"character not found"}`)
}

func TestCreateRankChangeRequest_ReturnsBadRequest_WhenTheNoteIsTooLong(t *testing.T) {
	// given my main
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	me := srv.LoginAs(t, "80351110224678912", "thundermane")
	insertCharacter(t, db, tankardID, "Tankard", "Ironhide", "Warrior", true, &me.ID)

	// when I ask with a 501-character note
	resp := srv.Post(t, "/api/me/characters/"+tankardID+"/rank-requests", map[string]any{"note": strings.Repeat("a", 501)})

	// then I expect a 400 naming the limit
	resp.RequireStatus(t, http.StatusBadRequest)
	requireBody(t, resp, `{"error":"note: must be at most 500 characters"}`)
}

func TestCreateRankChangeRequest_ReturnsUnauthorized_WhenNotLoggedIn(t *testing.T) {
	// given nobody is logged in
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertCharacter(t, db, tankardID, "Tankard", "Ironhide", "Warrior", true, nil)

	// when I ask for a rank change
	resp := srv.Post(t, "/api/me/characters/"+tankardID+"/rank-requests", map[string]any{})

	// then I expect a 401
	resp.RequireStatus(t, http.StatusUnauthorized)
}

func TestCreateRankChangeRequest_PostsDiscordEmbed_WhenWebhookConfigured(t *testing.T) {
	// given my main and a deploy with the rank requests webhook set
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	me := srv.LoginAs(t, "80351110224678912", "thundermane")
	insertCharacter(t, db, tankardID, "Tankard", "Ironhide", "Warrior", true, &me.ID)

	// when I ask for its rank to be changed
	resp := srv.Post(t, "/api/me/characters/"+tankardID+"/rank-requests", map[string]any{"note": "Levelling a new main"})

	// then I expect one embed linking officers to the requests page
	resp.RequireStatus(t, http.StatusCreated)
	want := []string{`{"embeds":[{"title":"Rank request: Tankard Ironhide","url":"https://fuzion.example/officer/rank-requests",` +
		`"description":"An officer needs to change this character's rank in game.","fields":[` +
		`{"name":"Character","value":"Tankard Ironhide (Warrior)"},` +
		`{"name":"Request","value":"Main to alt rank"},` +
		`{"name":"Member","value":"thundermane"},` +
		`{"name":"Note","value":"Levelling a new main"}]}]}`}
	var got []string
	for _, body := range srv.RankRequests.Bodies() {
		got = append(got, string(body))
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected webhook posts (-want +got):\n%s", diff)
	}
}

func TestCreateRankChangeRequest_EscapesMarkdownInTheEmbedNote_WhenTheNoteHasALink(t *testing.T) {
	// given my main and a deploy with the rank requests webhook set
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	me := srv.LoginAs(t, "80351110224678912", "thundermane")
	insertCharacter(t, db, tankardID, "Tankard", "Ironhide", "Warrior", true, &me.ID)

	// when I ask for its rank to be changed with a masked link and bold text in the note
	resp := srv.Post(t, "/api/me/characters/"+tankardID+"/rank-requests", map[string]any{"note": "[Officer form](https://example.com) **now**"})

	// then I expect the note posted with its markdown escaped, and stored as I wrote it
	resp.RequireStatus(t, http.StatusCreated)
	want := []string{`{"embeds":[{"title":"Rank request: Tankard Ironhide","url":"https://fuzion.example/officer/rank-requests",` +
		`"description":"An officer needs to change this character's rank in game.","fields":[` +
		`{"name":"Character","value":"Tankard Ironhide (Warrior)"},` +
		`{"name":"Request","value":"Main to alt rank"},` +
		`{"name":"Member","value":"thundermane"},` +
		`{"name":"Note","value":"\\[Officer form\\]\\(https://example.com\\) \\*\\*now\\*\\*"}]}]}`}
	var got []string
	for _, body := range srv.RankRequests.Bodies() {
		got = append(got, string(body))
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected webhook posts (-want +got):\n%s", diff)
	}
	if created := decodeCreated(t, resp); created.Note != "[Officer form](https://example.com) **now**" {
		t.Errorf("expected the stored note unescaped, got %q", created.Note)
	}
}

func TestCreateRankChangeRequest_Succeeds_WhenWebhookFails(t *testing.T) {
	// given my main and a rank requests webhook that is down
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.RankRequests.Status = http.StatusInternalServerError
	me := srv.LoginAs(t, "80351110224678912", "thundermane")
	insertCharacter(t, db, tankardID, "Tankard", "Ironhide", "Warrior", true, &me.ID)

	// when I ask for its rank to be changed
	resp := srv.Post(t, "/api/me/characters/"+tankardID+"/rank-requests", map[string]any{})

	// then I expect a 201 and the request stored open
	resp.RequireStatus(t, http.StatusCreated)
	created := decodeCreated(t, resp)
	want := requestJSON{
		CharacterID: tankardID, CharacterName: "Tankard", CharacterSecondaryName: "Ironhide", CharacterClass: "Warrior",
		Direction: "to_alt", Status: "open", RequestedBy: "thundermane", CreatedAt: "2026-03-14T20:00:00Z",
	}
	if diff := cmp.Diff(want, created); diff != "" {
		t.Errorf("unexpected request (-want +got):\n%s", diff)
	}
	stored := storedRequests(t, db)
	for i := range stored {
		stored[i].ID = ""
	}
	if diff := cmp.Diff([]storedRequest{{Status: "open"}}, stored); diff != "" {
		t.Errorf("unexpected stored requests (-want +got):\n%s", diff)
	}
}

func TestCreateRankChangeRequest_PostsNothing_WhenWebhookIsNotConfigured(t *testing.T) {
	// given my main and a deploy without DISCORD_RANK_REQUESTS_WEBHOOK_URL
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db, testutil.WithoutRankRequestsWebhook())
	me := srv.LoginAs(t, "80351110224678912", "thundermane")
	insertCharacter(t, db, tankardID, "Tankard", "Ironhide", "Warrior", true, &me.ID)

	// when I ask for its rank to be changed
	resp := srv.Post(t, "/api/me/characters/"+tankardID+"/rank-requests", map[string]any{})

	// then I expect a 201 and no webhook post
	resp.RequireStatus(t, http.StatusCreated)
	if got := len(srv.RankRequests.Bodies()); got != 0 {
		t.Errorf("expected no webhook posts, got %d", got)
	}
}

func TestListMyRankChangeRequests_ReturnsTheLatestRequestPerCharacter_WhenOlderOnesExist(t *testing.T) {
	// given my main with a declined then a cancelled request, my alt with an open one,
	// and another member's request
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	other := srv.LoginAs(t, "11111111111111111", "someoneelse")
	me := srv.LoginAs(t, "80351110224678912", "thundermane")
	insertCharacter(t, db, tankardID, "Tankard", "Ironhide", "Warrior", true, &me.ID)
	insertCharacter(t, db, healbotID, "Healbot", "Lightward", "Priest", false, &me.ID)
	insertCharacter(t, db, "33333333-3333-3333-3333-333333333333", "Stranger", "Elsewhere", "Mage", true, &other.ID)
	insertRequest(t, db, declineID, tankardID, me.ID, "to_alt", "declined", "2026-03-01T12:00:00Z", strPtr("2026-03-02T12:00:00Z"))
	db.MustExec(`UPDATE rank_change_requests SET resolution_note = 'Raid needs you' WHERE id = $1`, declineID)
	insertRequest(t, db, cancelID, tankardID, me.ID, "to_alt", "cancelled", "2026-03-05T12:00:00Z", strPtr("2026-03-05T13:00:00Z"))
	insertRequest(t, db, openID, healbotID, me.ID, "to_main", "open", "2026-03-06T12:00:00Z", nil)
	insertRequest(t, db, laterID, "33333333-3333-3333-3333-333333333333", other.ID, "to_alt", "open", "2026-03-07T12:00:00Z", nil)

	// when I list my rank requests
	resp := srv.Get(t, "/api/me/rank-requests")

	// then I expect the declined one for my main, since cancelled ones are hidden, and the open one for my alt
	resp.RequireStatus(t, http.StatusOK)
	var got []requestJSON
	resp.DecodeJSON(t, &got)
	want := []requestJSON{
		{
			ID: openID, CharacterID: healbotID, CharacterName: "Healbot", CharacterSecondaryName: "Lightward", CharacterClass: "Priest",
			Direction: "to_main", Status: "open", Note: "please", RequestedBy: "thundermane", CreatedAt: "2026-03-06T12:00:00Z",
		},
		{
			ID: declineID, CharacterID: tankardID, CharacterName: "Tankard", CharacterSecondaryName: "Ironhide", CharacterClass: "Warrior",
			Direction: "to_alt", Status: "declined", Note: "please", ResolutionNote: "Raid needs you", RequestedBy: "thundermane",
			CreatedAt: "2026-03-01T12:00:00Z", ResolvedAt: strPtr("2026-03-02T12:00:00Z"),
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected requests (-want +got):\n%s", diff)
	}
}

func TestListMyRankChangeRequests_ReturnsUnauthorized_WhenNotLoggedIn(t *testing.T) {
	// given nobody is logged in
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I list my rank requests
	resp := srv.Get(t, "/api/me/rank-requests")

	// then I expect a 401
	resp.RequireStatus(t, http.StatusUnauthorized)
}

func TestCancelRankChangeRequest_MarksItCancelled_WhenTheRequesterCancelsAnOpenRequest(t *testing.T) {
	// given my open request
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	me := srv.LoginAs(t, "80351110224678912", "thundermane")
	insertCharacter(t, db, tankardID, "Tankard", "Ironhide", "Warrior", true, &me.ID)
	insertRequest(t, db, openID, tankardID, me.ID, "to_alt", "open", "2026-03-01T12:00:00Z", nil)

	// when I cancel it
	resp := srv.Delete(t, "/api/me/rank-requests/"+openID)

	// then I expect a 204 and the request cancelled
	resp.RequireStatus(t, http.StatusNoContent)
	want := []storedRequest{{ID: openID, Status: "cancelled", ResolvedAt: strPtr("2026-03-14T20:00:00Z")}}
	if diff := cmp.Diff(want, storedRequests(t, db)); diff != "" {
		t.Errorf("unexpected requests (-want +got):\n%s", diff)
	}
}

func TestCancelRankChangeRequest_ReturnsForbidden_WhenTheCharacterBelongsToSomeoneElse(t *testing.T) {
	// given another member's open request
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	other := srv.LoginAs(t, "11111111111111111", "someoneelse")
	srv.LoginAs(t, "80351110224678912", "thundermane")
	insertCharacter(t, db, tankardID, "Tankard", "Ironhide", "Warrior", true, &other.ID)
	insertRequest(t, db, openID, tankardID, other.ID, "to_alt", "open", "2026-03-01T12:00:00Z", nil)

	// when I try to cancel it
	resp := srv.Delete(t, "/api/me/rank-requests/"+openID)

	// then I expect a 403 and the request still open
	resp.RequireStatus(t, http.StatusForbidden)
	requireBody(t, resp, `{"error":"rank request is not yours"}`)
	want := []storedRequest{{ID: openID, Status: "open"}}
	if diff := cmp.Diff(want, storedRequests(t, db)); diff != "" {
		t.Errorf("unexpected requests (-want +got):\n%s", diff)
	}
}

func TestCancelRankChangeRequest_MarksItCancelled_WhenTheCharacterChangedHandsAfterItWasFiled(t *testing.T) {
	// given an open request the previous owner filed on a character I now own
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	previous := srv.LoginAs(t, "11111111111111111", "someoneelse")
	me := srv.LoginAs(t, "80351110224678912", "thundermane")
	insertCharacter(t, db, tankardID, "Tankard", "Ironhide", "Warrior", true, &me.ID)
	insertRequest(t, db, openID, tankardID, previous.ID, "to_alt", "open", "2026-03-01T12:00:00Z", nil)

	// when I cancel it
	resp := srv.Delete(t, "/api/me/rank-requests/"+openID)

	// then I expect a 204 and the request cancelled
	resp.RequireStatus(t, http.StatusNoContent)
	want := []storedRequest{{ID: openID, Status: "cancelled", ResolvedAt: strPtr("2026-03-14T20:00:00Z")}}
	if diff := cmp.Diff(want, storedRequests(t, db)); diff != "" {
		t.Errorf("unexpected requests (-want +got):\n%s", diff)
	}
}

func TestCancelRankChangeRequest_ReturnsForbidden_WhenTheCharacterIsNoLongerMine(t *testing.T) {
	// given an open request I filed on a character that has since moved to another member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	other := srv.LoginAs(t, "11111111111111111", "someoneelse")
	me := srv.LoginAs(t, "80351110224678912", "thundermane")
	insertCharacter(t, db, tankardID, "Tankard", "Ironhide", "Warrior", true, &other.ID)
	insertRequest(t, db, openID, tankardID, me.ID, "to_alt", "open", "2026-03-01T12:00:00Z", nil)

	// when I try to cancel it
	resp := srv.Delete(t, "/api/me/rank-requests/"+openID)

	// then I expect a 403 and the request still open
	resp.RequireStatus(t, http.StatusForbidden)
	requireBody(t, resp, `{"error":"rank request is not yours"}`)
	want := []storedRequest{{ID: openID, Status: "open"}}
	if diff := cmp.Diff(want, storedRequests(t, db)); diff != "" {
		t.Errorf("unexpected requests (-want +got):\n%s", diff)
	}
}

func TestCancelRankChangeRequest_ReturnsConflict_WhenTheRequestIsResolved(t *testing.T) {
	// given my request an officer already declined
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	me := srv.LoginAs(t, "80351110224678912", "thundermane")
	insertCharacter(t, db, tankardID, "Tankard", "Ironhide", "Warrior", true, &me.ID)
	insertRequest(t, db, declineID, tankardID, me.ID, "to_alt", "declined", "2026-03-01T12:00:00Z", strPtr("2026-03-02T12:00:00Z"))

	// when I try to cancel it
	resp := srv.Delete(t, "/api/me/rank-requests/"+declineID)

	// then I expect a 409 and the request still declined
	resp.RequireStatus(t, http.StatusConflict)
	requireBody(t, resp, `{"error":"rank request is no longer open"}`)
	want := []storedRequest{{ID: declineID, Status: "declined", ResolvedAt: strPtr("2026-03-02T12:00:00Z")}}
	if diff := cmp.Diff(want, storedRequests(t, db)); diff != "" {
		t.Errorf("unexpected requests (-want +got):\n%s", diff)
	}
}

func TestCancelRankChangeRequest_ReturnsNotFound_WhenTheRequestDoesNotExist(t *testing.T) {
	// given a logged-in member and no such request
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "80351110224678912", "thundermane")

	// when I try to cancel it
	resp := srv.Delete(t, "/api/me/rank-requests/"+openID)

	// then I expect a 404
	resp.RequireStatus(t, http.StatusNotFound)
	requireBody(t, resp, `{"error":"rank request not found"}`)
}

func TestListRankChangeRequests_ReturnsOpenOldestFirstThenResolvedNewestFirst_WhenOfficer(t *testing.T) {
	// given two open requests, one done, one declined and one cancelled
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	member := srv.LoginAs(t, "80351110224678912", "thundermane")
	officer := srv.LoginAs(t, "22222222222222222", "officerbob", testutil.AsOfficer())
	insertCharacter(t, db, tankardID, "Tankard", "Ironhide", "Warrior", true, &member.ID)
	insertCharacter(t, db, healbotID, "Healbot", "Lightward", "Priest", false, &member.ID)
	insertRequest(t, db, doneID, tankardID, member.ID, "to_alt", "done", "2026-02-01T12:00:00Z", strPtr("2026-02-02T12:00:00Z"))
	db.MustExec(`UPDATE rank_change_requests SET resolved_by = $1 WHERE id = $2`, officer.ID, doneID)
	insertRequest(t, db, declineID, tankardID, member.ID, "to_alt", "declined", "2026-02-10T12:00:00Z", strPtr("2026-02-11T12:00:00Z"))
	db.MustExec(`UPDATE rank_change_requests SET resolved_by = $1, resolution_note = 'Not yet' WHERE id = $2`, officer.ID, declineID)
	insertRequest(t, db, cancelID, healbotID, member.ID, "to_main", "cancelled", "2026-02-12T12:00:00Z", strPtr("2026-02-12T13:00:00Z"))
	insertRequest(t, db, laterID, tankardID, member.ID, "to_alt", "open", "2026-03-05T12:00:00Z", nil)
	insertRequest(t, db, openID, healbotID, member.ID, "to_main", "open", "2026-03-01T12:00:00Z", nil)

	// when I list rank requests
	resp := srv.Get(t, "/api/rank-requests")

	// then I expect the open ones oldest first, then the resolved ones newest first, without the cancelled one
	resp.RequireStatus(t, http.StatusOK)
	var got []requestJSON
	resp.DecodeJSON(t, &got)
	want := []requestJSON{
		{
			ID: openID, CharacterID: healbotID, CharacterName: "Healbot", CharacterSecondaryName: "Lightward", CharacterClass: "Priest",
			Direction: "to_main", Status: "open", Note: "please", RequestedBy: "thundermane", CreatedAt: "2026-03-01T12:00:00Z",
		},
		{
			ID: laterID, CharacterID: tankardID, CharacterName: "Tankard", CharacterSecondaryName: "Ironhide", CharacterClass: "Warrior",
			Direction: "to_alt", Status: "open", Note: "please", RequestedBy: "thundermane", CreatedAt: "2026-03-05T12:00:00Z",
		},
		{
			ID: declineID, CharacterID: tankardID, CharacterName: "Tankard", CharacterSecondaryName: "Ironhide", CharacterClass: "Warrior",
			Direction: "to_alt", Status: "declined", Note: "please", ResolutionNote: "Not yet", RequestedBy: "thundermane",
			CreatedAt: "2026-02-10T12:00:00Z", ResolvedAt: strPtr("2026-02-11T12:00:00Z"), ResolvedBy: strPtr("officerbob"),
		},
		{
			ID: doneID, CharacterID: tankardID, CharacterName: "Tankard", CharacterSecondaryName: "Ironhide", CharacterClass: "Warrior",
			Direction: "to_alt", Status: "done", Note: "please", RequestedBy: "thundermane",
			CreatedAt: "2026-02-01T12:00:00Z", ResolvedAt: strPtr("2026-02-02T12:00:00Z"), ResolvedBy: strPtr("officerbob"),
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected requests (-want +got):\n%s", diff)
	}
}

func TestListRankChangeRequests_ReturnsForbidden_WhenNotOfficer(t *testing.T) {
	// given a logged-in member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "80351110224678912", "thundermane")

	// when I list rank requests
	resp := srv.Get(t, "/api/rank-requests")

	// then I expect a 403
	resp.RequireStatus(t, http.StatusForbidden)
}

func TestResolveRankChangeRequest_MarksItDeclinedWithANote_WhenOfficer(t *testing.T) {
	// given a member's open request
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	member := srv.LoginAs(t, "80351110224678912", "thundermane")
	officer := srv.LoginAs(t, "22222222222222222", "officerbob", testutil.AsOfficer())
	insertCharacter(t, db, tankardID, "Tankard", "Ironhide", "Warrior", true, &member.ID)
	insertRequest(t, db, openID, tankardID, member.ID, "to_alt", "open", "2026-03-01T12:00:00Z", nil)

	// when I decline it with a note
	resp := srv.Patch(t, "/api/rank-requests/"+openID, map[string]any{"status": "declined", "note": " We need your tank "})

	// then I expect it declined, naming me, with the trimmed note
	resp.RequireStatus(t, http.StatusOK)
	var got requestJSON
	resp.DecodeJSON(t, &got)
	want := requestJSON{
		ID: openID, CharacterID: tankardID, CharacterName: "Tankard", CharacterSecondaryName: "Ironhide", CharacterClass: "Warrior",
		Direction: "to_alt", Status: "declined", Note: "please", ResolutionNote: "We need your tank", RequestedBy: "thundermane",
		CreatedAt: "2026-03-01T12:00:00Z", ResolvedAt: strPtr("2026-03-14T20:00:00Z"), ResolvedBy: strPtr("officerbob"),
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected request (-want +got):\n%s", diff)
	}
	wantStored := []storedRequest{{ID: openID, Status: "declined", ResolutionNote: "We need your tank", ResolvedAt: strPtr("2026-03-14T20:00:00Z"), ResolvedBy: strPtr(officer.ID.String())}}
	if diff := cmp.Diff(wantStored, storedRequests(t, db)); diff != "" {
		t.Errorf("unexpected stored requests (-want +got):\n%s", diff)
	}
}

func TestResolveRankChangeRequest_MarksItDone_WhenOfficer(t *testing.T) {
	// given a member's open request
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	member := srv.LoginAs(t, "80351110224678912", "thundermane")
	officer := srv.LoginAs(t, "22222222222222222", "officerbob", testutil.AsOfficer())
	insertCharacter(t, db, tankardID, "Tankard", "Ironhide", "Warrior", true, &member.ID)
	insertRequest(t, db, openID, tankardID, member.ID, "to_alt", "open", "2026-03-01T12:00:00Z", nil)

	// when I mark it done without a note
	resp := srv.Patch(t, "/api/rank-requests/"+openID, map[string]any{"status": "done"})

	// then I expect it done, naming me, with no resolution note
	resp.RequireStatus(t, http.StatusOK)
	var got requestJSON
	resp.DecodeJSON(t, &got)
	want := requestJSON{
		ID: openID, CharacterID: tankardID, CharacterName: "Tankard", CharacterSecondaryName: "Ironhide", CharacterClass: "Warrior",
		Direction: "to_alt", Status: "done", Note: "please", RequestedBy: "thundermane",
		CreatedAt: "2026-03-01T12:00:00Z", ResolvedAt: strPtr("2026-03-14T20:00:00Z"), ResolvedBy: strPtr("officerbob"),
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected request (-want +got):\n%s", diff)
	}
	wantStored := []storedRequest{{ID: openID, Status: "done", ResolvedAt: strPtr("2026-03-14T20:00:00Z"), ResolvedBy: strPtr(officer.ID.String())}}
	if diff := cmp.Diff(wantStored, storedRequests(t, db)); diff != "" {
		t.Errorf("unexpected stored requests (-want +got):\n%s", diff)
	}
}

func TestResolveRankChangeRequest_ReturnsNotFound_WhenTheRequestDoesNotExist(t *testing.T) {
	// given an officer and no such request
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "22222222222222222", "officerbob", testutil.AsOfficer())

	// when I try to mark it done
	resp := srv.Patch(t, "/api/rank-requests/"+openID, map[string]any{"status": "done"})

	// then I expect a 404
	resp.RequireStatus(t, http.StatusNotFound)
	requireBody(t, resp, `{"error":"rank request not found"}`)
}

func TestResolveRankChangeRequest_ReturnsForbidden_WhenNotOfficer(t *testing.T) {
	// given my own open request
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	me := srv.LoginAs(t, "80351110224678912", "thundermane")
	insertCharacter(t, db, tankardID, "Tankard", "Ironhide", "Warrior", true, &me.ID)
	insertRequest(t, db, openID, tankardID, me.ID, "to_alt", "open", "2026-03-01T12:00:00Z", nil)

	// when I try to mark it done
	resp := srv.Patch(t, "/api/rank-requests/"+openID, map[string]any{"status": "done"})

	// then I expect a 403 and the request still open
	resp.RequireStatus(t, http.StatusForbidden)
	want := []storedRequest{{ID: openID, Status: "open"}}
	if diff := cmp.Diff(want, storedRequests(t, db)); diff != "" {
		t.Errorf("unexpected requests (-want +got):\n%s", diff)
	}
}

func TestResolveRankChangeRequest_ReturnsBadRequest_WhenTheStatusIsNotDoneOrDeclined(t *testing.T) {
	// given a member's open request
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	member := srv.LoginAs(t, "80351110224678912", "thundermane")
	srv.LoginAs(t, "22222222222222222", "officerbob", testutil.AsOfficer())
	insertCharacter(t, db, tankardID, "Tankard", "Ironhide", "Warrior", true, &member.ID)
	insertRequest(t, db, openID, tankardID, member.ID, "to_alt", "open", "2026-03-01T12:00:00Z", nil)

	// when I try to mark it cancelled
	resp := srv.Patch(t, "/api/rank-requests/"+openID, map[string]any{"status": "cancelled"})

	// then I expect a 400 listing the allowed statuses
	resp.RequireStatus(t, http.StatusBadRequest)
	requireBody(t, resp, `{"error":"status: must be one of done, declined"}`)
}

func TestResolveRankChangeRequest_ReturnsConflict_WhenTheRequestIsAlreadyResolved(t *testing.T) {
	// given a request the sync already closed
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	member := srv.LoginAs(t, "80351110224678912", "thundermane")
	srv.LoginAs(t, "22222222222222222", "officerbob", testutil.AsOfficer())
	insertCharacter(t, db, tankardID, "Tankard", "Ironhide", "Warrior", true, &member.ID)
	insertRequest(t, db, doneID, tankardID, member.ID, "to_alt", "done", "2026-03-01T12:00:00Z", strPtr("2026-03-02T12:00:00Z"))

	// when I try to decline it
	resp := srv.Patch(t, "/api/rank-requests/"+doneID, map[string]any{"status": "declined"})

	// then I expect a 409 and the request still done
	resp.RequireStatus(t, http.StatusConflict)
	requireBody(t, resp, `{"error":"rank request is no longer open"}`)
	want := []storedRequest{{ID: doneID, Status: "done", ResolvedAt: strPtr("2026-03-02T12:00:00Z")}}
	if diff := cmp.Diff(want, storedRequests(t, db)); diff != "" {
		t.Errorf("unexpected requests (-want +got):\n%s", diff)
	}
}

func TestCountOpenRankChangeRequests_ReturnsOpenCount_WhenOfficer(t *testing.T) {
	// given two open requests and a done one
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	member := srv.LoginAs(t, "80351110224678912", "thundermane")
	srv.LoginAs(t, "22222222222222222", "officerbob", testutil.AsOfficer())
	insertCharacter(t, db, tankardID, "Tankard", "Ironhide", "Warrior", true, &member.ID)
	insertCharacter(t, db, healbotID, "Healbot", "Lightward", "Priest", false, &member.ID)
	insertRequest(t, db, openID, tankardID, member.ID, "to_alt", "open", "2026-03-01T12:00:00Z", nil)
	insertRequest(t, db, laterID, healbotID, member.ID, "to_main", "open", "2026-03-02T12:00:00Z", nil)
	insertRequest(t, db, doneID, tankardID, member.ID, "to_alt", "done", "2026-02-01T12:00:00Z", strPtr("2026-02-02T12:00:00Z"))

	// when I ask how many are open
	resp := srv.Get(t, "/api/rank-requests/count")

	// then I expect 2
	resp.RequireStatus(t, http.StatusOK)
	requireBody(t, resp, `{"open":2}`)
}

func TestCountOpenRankChangeRequests_ReturnsForbidden_WhenNotOfficer(t *testing.T) {
	// given a logged-in member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "80351110224678912", "thundermane")

	// when I ask how many are open
	resp := srv.Get(t, "/api/rank-requests/count")

	// then I expect a 403
	resp.RequireStatus(t, http.StatusForbidden)
}
