package auth_test

import (
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/cageymage/fuzion/backend/internal/testutil"
)

type battleNetRow struct {
	ID  *string `db:"battlenet_id"`
	Tag *string `db:"battlenet_tag"`
}

type meBattleNetJSON struct {
	BattleNetLinked bool    `json:"battlenetLinked"`
	BattleTag       *string `json:"battletag"`
}

func startBattleNetLink(t *testing.T, srv *testutil.Server) string {
	t.Helper()
	resp := srv.Get(t, "/api/auth/battlenet/link")
	resp.RequireStatus(t, 302)
	location, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatalf("parse link redirect: %v", err)
	}
	return location.Query().Get("state")
}

func strPtr(s string) *string { return &s }

func TestBattlenetLink_RedirectsToBattleNetWithWowProfileScope(t *testing.T) {
	// given a logged-in member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "80351110224678912", "thundermane")

	// when I start linking Battle.net
	resp := srv.Get(t, "/api/auth/battlenet/link")

	// then I expect a redirect to Battle.net's authorize URL asking for wow.profile with a state matching the cookie
	resp.RequireStatus(t, 302)
	location, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatalf("parse redirect location: %v", err)
	}
	if got := location.Scheme + "://" + location.Host + location.Path; got != srv.BattleNet.URL+"/authorize" {
		t.Errorf("expected redirect to the fake Battle.net authorize endpoint, got %s", got)
	}
	query := location.Query()
	if query.Get("scope") != "wow.profile" || query.Get("client_id") != testutil.BattleNetClientID || query.Get("redirect_uri") != testutil.BattleNetRedirectURL {
		t.Errorf("unexpected authorize query: %v", query)
	}
	if cookie := setCookie(t, resp, "fuzion_oauth_state"); cookie.Value != query.Get("state") || cookie.Value == "" {
		t.Errorf("state cookie %q does not match redirect state %q", cookie.Value, query.Get("state"))
	}
}

func TestBattlenetLink_ReturnsUnauthorized_WhenNotLoggedIn(t *testing.T) {
	// given nobody is logged in
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I start linking Battle.net
	resp := srv.Get(t, "/api/auth/battlenet/link")

	// then I expect a 401
	resp.RequireStatus(t, 401)
}

func TestBattlenetLink_AttachesIDToCurrentUser(t *testing.T) {
	// given a logged-in member who started linking and a Battle.net that accepts the code
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "80351110224678912", "thundermane")
	state := startBattleNetLink(t, srv)
	srv.BattleNet.GrantCode("bnet-code", testutil.BattleNetUser{ID: 123456789, BattleTag: "Thunder#1234"})

	// when Battle.net sends the browser back with the code and matching state
	resp := srv.Get(t, "/api/auth/battlenet/callback?code=bnet-code&state="+state)

	// then I expect a redirect to the dashboard and the Battle.net identity stored on my row
	resp.RequireStatus(t, 302)
	if got := resp.Header.Get("Location"); got != "/dashboard" {
		t.Errorf("expected redirect to /dashboard, got %q", got)
	}
	var row battleNetRow
	if err := db.Get(&row, `SELECT battlenet_id, battlenet_tag FROM users WHERE discord_id = '80351110224678912'`); err != nil {
		t.Fatalf("select linked user: %v", err)
	}
	want := battleNetRow{ID: strPtr("123456789"), Tag: strPtr("Thunder#1234")}
	if diff := cmp.Diff(want, row); diff != "" {
		t.Errorf("unexpected battle.net columns (-want +got):\n%s", diff)
	}
	var sessions int
	if err := db.Get(&sessions, `SELECT count(*) FROM sessions`); err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if sessions != 1 {
		t.Errorf("expected linking to leave the single Discord session alone, found %d sessions", sessions)
	}
}

func TestBattlenetLink_RedirectsWithAlreadyLinked_WhenBattlenetBelongsToAnotherUser(t *testing.T) {
	// given a Battle.net account already linked to another member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	other := srv.LoginAs(t, "11111111111111111", "someoneelse")
	db.MustExec(`UPDATE users SET battlenet_id = '123456789', battlenet_tag = 'Thunder#1234' WHERE id = $1`, other.ID)
	srv.LoginAs(t, "80351110224678912", "thundermane")
	state := startBattleNetLink(t, srv)
	srv.BattleNet.GrantCode("bnet-code", testutil.BattleNetUser{ID: 123456789, BattleTag: "Thunder#1234"})

	// when I complete the link with that same Battle.net account
	resp := srv.Get(t, "/api/auth/battlenet/callback?code=bnet-code&state="+state)

	// then I expect to be sent back to the dashboard with an already-linked marker and nothing attached to my row
	resp.RequireStatus(t, 302)
	if got := resp.Header.Get("Location"); got != "/dashboard?battlenet=already-linked" {
		t.Errorf("expected redirect to the already-linked dashboard, got %q", got)
	}
	var mine battleNetRow
	if err := db.Get(&mine, `SELECT battlenet_id, battlenet_tag FROM users WHERE discord_id = '80351110224678912'`); err != nil {
		t.Fatalf("select user: %v", err)
	}
	if diff := cmp.Diff(battleNetRow{}, mine); diff != "" {
		t.Errorf("expected no battle.net columns on my row (-want +got):\n%s", diff)
	}
}

func TestBattlenetLink_ReturnsBadRequest_WhenStateDoesNotMatch(t *testing.T) {
	// given a logged-in member who started linking
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "80351110224678912", "thundermane")
	startBattleNetLink(t, srv)
	srv.BattleNet.GrantCode("bnet-code", testutil.BattleNetUser{ID: 123456789, BattleTag: "Thunder#1234"})

	// when the callback arrives with a different state
	resp := srv.Get(t, "/api/auth/battlenet/callback?code=bnet-code&state=forged-by-someone-else")

	// then I expect a 400 and nothing linked
	resp.RequireStatus(t, 400)
	var linked int
	if err := db.Get(&linked, `SELECT count(*) FROM users WHERE battlenet_id IS NOT NULL`); err != nil {
		t.Fatalf("count linked users: %v", err)
	}
	if linked != 0 {
		t.Errorf("expected no linked users, found %d", linked)
	}
}

func TestBattlenetLink_ReturnsBadGateway_WhenBattlenetRejectsTheCode(t *testing.T) {
	// given a logged-in member who started linking but a code Battle.net does not recognise
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "80351110224678912", "thundermane")
	state := startBattleNetLink(t, srv)

	// when the callback arrives with that code
	resp := srv.Get(t, "/api/auth/battlenet/callback?code=bogus&state="+state)

	// then I expect a 502
	resp.RequireStatus(t, 502)
}

func TestBattlenetLink_RedirectsToDashboardUnlinked_WhenUserDeniesConsent(t *testing.T) {
	// given a logged-in member who started linking
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "80351110224678912", "thundermane")
	state := startBattleNetLink(t, srv)

	// when Battle.net sends the browser back with an error instead of a code
	resp := srv.Get(t, "/api/auth/battlenet/callback?error=access_denied&state="+state)

	// then I expect a redirect to the dashboard and nothing linked
	resp.RequireStatus(t, 302)
	if got := resp.Header.Get("Location"); got != "/dashboard" {
		t.Errorf("expected redirect to /dashboard, got %q", got)
	}
	var linked int
	if err := db.Get(&linked, `SELECT count(*) FROM users WHERE battlenet_id IS NOT NULL`); err != nil {
		t.Fatalf("count linked users: %v", err)
	}
	if linked != 0 {
		t.Errorf("expected no linked users, found %d", linked)
	}
}

func TestUnlinkBattlenet_ClearsIdentity(t *testing.T) {
	// given a member with a linked Battle.net account
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	user := srv.LoginAs(t, "80351110224678912", "thundermane")
	db.MustExec(`
		UPDATE users
		SET battlenet_id = '123456789', battlenet_tag = 'Thunder#1234'
		WHERE id = $1`, user.ID)

	// when I unlink it
	resp := srv.Delete(t, "/api/auth/battlenet")

	// then I expect a 204 and every Battle.net column cleared
	resp.RequireStatus(t, 204)
	var row battleNetRow
	if err := db.Get(&row, `SELECT battlenet_id, battlenet_tag FROM users WHERE id = $1`, user.ID); err != nil {
		t.Fatalf("select user: %v", err)
	}
	if diff := cmp.Diff(battleNetRow{}, row); diff != "" {
		t.Errorf("expected cleared battle.net columns (-want +got):\n%s", diff)
	}
}

func TestUnlinkBattlenet_ReturnsUnauthorized_WhenNotLoggedIn(t *testing.T) {
	// given nobody is logged in
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I unlink
	resp := srv.Delete(t, "/api/auth/battlenet")

	// then I expect a 401
	resp.RequireStatus(t, 401)
}

func TestAuthMe_ReturnsBattletag_WhenBattlenetIsLinked(t *testing.T) {
	// given a member with a linked Battle.net account
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	user := srv.LoginAs(t, "80351110224678912", "thundermane")
	db.MustExec(`UPDATE users SET battlenet_id = '123456789', battlenet_tag = 'Thunder#1234' WHERE id = $1`, user.ID)

	// when I ask who I am
	resp := srv.Get(t, "/api/auth/me")

	// then I expect the link status and battletag
	resp.RequireStatus(t, 200)
	var me meBattleNetJSON
	resp.DecodeJSON(t, &me)
	want := meBattleNetJSON{BattleNetLinked: true, BattleTag: strPtr("Thunder#1234")}
	if diff := cmp.Diff(want, me); diff != "" {
		t.Errorf("unexpected /me response (-want +got):\n%s", diff)
	}
}

func TestAuthMe_ReportsBattlenetNotLinked_WhenNothingIsLinked(t *testing.T) {
	// given a member who has not linked Battle.net
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "80351110224678912", "thundermane")

	// when I ask who I am
	resp := srv.Get(t, "/api/auth/me")

	// then I expect it unlinked with no battletag
	resp.RequireStatus(t, 200)
	var me meBattleNetJSON
	resp.DecodeJSON(t, &me)
	if diff := cmp.Diff(meBattleNetJSON{}, me); diff != "" {
		t.Errorf("unexpected /me response (-want +got):\n%s", diff)
	}
}
