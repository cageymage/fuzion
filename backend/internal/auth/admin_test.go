package auth_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/cageymage/fuzion/backend/internal/testutil"
)

type adminUserJSON struct {
	ID        string  `json:"id"`
	Username  string  `json:"username"`
	AvatarURL *string `json:"avatarUrl"`
	IsOfficer bool    `json:"isOfficer"`
	IsAdmin   bool    `json:"isAdmin"`
}

func TestListUsers_ReturnsAllUsersWithRoleFlags_WhenCallerIsAdmin(t *testing.T) {
	// given an admin and a plain member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	member := srv.LoginAs(t, "2", "member-bob")
	admin := srv.LoginAs(t, "1", "admin-alice", testutil.AsAdmin())

	// when the admin lists users
	resp := srv.Get(t, "/api/admin/users")

	// then I expect both users back with their role flags
	resp.RequireStatus(t, 200)
	var users []adminUserJSON
	resp.DecodeJSON(t, &users)
	want := []adminUserJSON{
		{ID: admin.ID.String(), Username: "admin-alice", IsOfficer: false, IsAdmin: true},
		{ID: member.ID.String(), Username: "member-bob", IsOfficer: false, IsAdmin: false},
	}
	if diff := cmp.Diff(want, users); diff != "" {
		t.Errorf("unexpected users list (-want +got):\n%s", diff)
	}
}

func TestListUsers_ReturnsForbidden_WhenCallerIsOnlyAnOfficer(t *testing.T) {
	// given a logged-in officer with no admin access
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "1", "officer-alice", testutil.AsOfficer())

	// when they try to list users
	resp := srv.Get(t, "/api/admin/users")

	// then I expect a 403
	resp.RequireStatus(t, 403)
}

func TestUpdateUserRoles_GrantsOfficerAccess_WhenCallerIsAdmin(t *testing.T) {
	// given an admin and a plain member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	member := srv.LoginAs(t, "2", "member-bob")
	srv.LoginAs(t, "1", "admin-alice", testutil.AsAdmin())

	// when the admin grants the member officer access
	resp := srv.Patch(t, "/api/admin/users/"+member.ID.String(), map[string]any{"isOfficer": true, "isAdmin": false})

	// then I expect the updated user back with officer access granted
	resp.RequireStatus(t, 200)
	var got adminUserJSON
	resp.DecodeJSON(t, &got)
	want := adminUserJSON{ID: member.ID.String(), Username: "member-bob", IsOfficer: true, IsAdmin: false}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected user (-want +got):\n%s", diff)
	}

	var isOfficer bool
	if err := db.Get(&isOfficer, `SELECT is_officer FROM users WHERE id = $1`, member.ID); err != nil {
		t.Fatalf("select updated user: %v", err)
	}
	if !isOfficer {
		t.Error("expected is_officer to be persisted as true")
	}
}

func TestUpdateUserRoles_ReturnsForbidden_WhenCallerIsOnlyAnOfficer(t *testing.T) {
	// given an officer (not an admin) and a plain member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	member := srv.LoginAs(t, "2", "member-bob")
	srv.LoginAs(t, "1", "officer-alice", testutil.AsOfficer())

	// when the officer tries to grant the member officer access
	resp := srv.Patch(t, "/api/admin/users/"+member.ID.String(), map[string]any{"isOfficer": true, "isAdmin": false})

	// then I expect a 403
	resp.RequireStatus(t, 403)
}

func TestUpdateUserRoles_ReturnsBadRequest_WhenAdminRemovesOwnAdminFlag(t *testing.T) {
	// given an admin
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	admin := srv.LoginAs(t, "1", "admin-alice", testutil.AsAdmin())

	// when the admin tries to remove their own admin access
	resp := srv.Patch(t, "/api/admin/users/"+admin.ID.String(), map[string]any{"isOfficer": false, "isAdmin": false})

	// then I expect a 400, and their admin access left untouched
	resp.RequireStatus(t, 400)
	var isAdmin bool
	if err := db.Get(&isAdmin, `SELECT is_admin FROM users WHERE id = $1`, admin.ID); err != nil {
		t.Fatalf("select admin: %v", err)
	}
	if !isAdmin {
		t.Error("expected is_admin to remain true")
	}
}

func TestUpdateUserRoles_ReturnsNotFound_WhenUserDoesNotExist(t *testing.T) {
	// given an admin and no user with the id being patched
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "1", "admin-alice", testutil.AsAdmin())

	// when the admin tries to update a nonexistent user
	resp := srv.Patch(t, "/api/admin/users/11111111-1111-1111-1111-111111111111", map[string]any{"isOfficer": true, "isAdmin": false})

	// then I expect a 404
	resp.RequireStatus(t, 404)
}

func TestUpdateUserRoles_ReturnsBadRequest_WhenUserIdIsNotAUuid(t *testing.T) {
	// given an admin
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "1", "admin-alice", testutil.AsAdmin())

	// when the admin sends a malformed user id
	resp := srv.Patch(t, "/api/admin/users/not-a-uuid", map[string]any{"isOfficer": true, "isAdmin": false})

	// then I expect a 400
	resp.RequireStatus(t, 400)
}

func TestAuthMe_ReturnsEffectiveOfficerAccess_WhenUserIsAdminButNotManuallyGrantedOfficer(t *testing.T) {
	// given an admin who was never separately granted officer access
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "1", "admin-alice", testutil.AsAdmin())

	// when I ask who I am
	resp := srv.Get(t, "/api/auth/me")

	// then I expect isOfficer to be true, since admins get officer-level access for free
	resp.RequireStatus(t, 200)
	var me adminUserJSON
	resp.DecodeJSON(t, &me)
	if !me.IsOfficer {
		t.Error("expected isOfficer to be true for an admin")
	}
	if !me.IsAdmin {
		t.Error("expected isAdmin to be true")
	}
}

func TestAuthCallback_GrantsAdmin_WhenNoUserExistsYet(t *testing.T) {
	// given a site with no users at all
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	state := startLogin(t, srv)
	srv.Discord.GrantCode("code-1", testutil.DiscordUser{ID: "80351110224678912", Username: "thundermane"})

	// when the first person signs in
	resp := srv.Get(t, "/api/auth/callback?code=code-1&state="+state)

	// then I expect them to be an admin
	resp.RequireStatus(t, 302)
	var isAdmin bool
	if err := db.Get(&isAdmin, `SELECT is_admin FROM users WHERE discord_id = '80351110224678912'`); err != nil {
		t.Fatalf("select user: %v", err)
	}
	if !isAdmin {
		t.Error("expected the first user to sign in to be granted admin")
	}
}

func TestAuthCallback_DoesNotGrantAdmin_WhenAnotherUserAlreadyExists(t *testing.T) {
	// given a site that already has a user
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO users (id, discord_id, username)
		VALUES ('11111111-1111-1111-1111-111111111111', '11111111111111111', 'firstmember')`)
	state := startLogin(t, srv)
	srv.Discord.GrantCode("code-1", testutil.DiscordUser{ID: "80351110224678912", Username: "thundermane"})

	// when a second person signs in
	resp := srv.Get(t, "/api/auth/callback?code=code-1&state="+state)

	// then I expect them to be an ordinary member
	resp.RequireStatus(t, 302)
	var isAdmin bool
	if err := db.Get(&isAdmin, `SELECT is_admin FROM users WHERE discord_id = '80351110224678912'`); err != nil {
		t.Fatalf("select user: %v", err)
	}
	if isAdmin {
		t.Error("expected a later signup to not be granted admin")
	}
}

func TestAuthCallback_KeepsAdmin_WhenAnExistingAdminLogsInAgain(t *testing.T) {
	// given an admin who has logged in before
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	db.MustExec(`
		INSERT INTO users (id, discord_id, username, is_admin)
		VALUES ('11111111-1111-1111-1111-111111111111', '80351110224678912', 'thundermane', true)`)
	state := startLogin(t, srv)
	srv.Discord.GrantCode("code-1", testutil.DiscordUser{ID: "80351110224678912", Username: "thundermane"})

	// when they log in again
	resp := srv.Get(t, "/api/auth/callback?code=code-1&state="+state)

	// then I expect them to still be an admin
	resp.RequireStatus(t, 302)
	var isAdmin bool
	if err := db.Get(&isAdmin, `SELECT is_admin FROM users WHERE discord_id = '80351110224678912'`); err != nil {
		t.Fatalf("select user: %v", err)
	}
	if !isAdmin {
		t.Error("expected a repeat login to leave an existing admin's access alone")
	}
}
