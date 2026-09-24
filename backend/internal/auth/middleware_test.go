package auth_test

import (
	"testing"

	"github.com/cageymage/fuzion/backend/internal/testutil"
)

func TestRequireOfficer_ReturnsUnauthorized_WhenAnonymous(t *testing.T) {
	// given nobody is logged in
	db := testutil.DB(t)
	srv := testutil.NewMiddlewareProbeServer(t, db)

	// when I hit an officer-gated route
	resp := srv.Get(t, "/officer-only")

	// then I expect a 401
	resp.RequireStatus(t, 401)
}

func TestRequireOfficer_ReturnsForbidden_WhenUserIsNotAnOfficer(t *testing.T) {
	// given a logged-in member with no officer or admin access
	db := testutil.DB(t)
	srv := testutil.NewMiddlewareProbeServer(t, db)
	srv.LoginAs(t, "80351110224678912", "thundermane")

	// when I hit an officer-gated route
	resp := srv.Get(t, "/officer-only")

	// then I expect a 403
	resp.RequireStatus(t, 403)
}

func TestRequireOfficer_AllowsRequest_WhenUserIsOfficer(t *testing.T) {
	// given a logged-in officer
	db := testutil.DB(t)
	srv := testutil.NewMiddlewareProbeServer(t, db)
	srv.LoginAs(t, "80351110224678912", "thundermane", testutil.AsOfficer())

	// when I hit an officer-gated route
	resp := srv.Get(t, "/officer-only")

	// then I expect a 200
	resp.RequireStatus(t, 200)
}

func TestRequireOfficer_AllowsRequest_WhenUserIsAdmin(t *testing.T) {
	// given a logged-in admin who was never separately granted officer access
	db := testutil.DB(t)
	srv := testutil.NewMiddlewareProbeServer(t, db)
	srv.LoginAs(t, "80351110224678912", "thundermane", testutil.AsAdmin())

	// when I hit an officer-gated route
	resp := srv.Get(t, "/officer-only")

	// then I expect a 200, since admins get officer-level access for free
	resp.RequireStatus(t, 200)
}

func TestRequireAdmin_ReturnsUnauthorized_WhenAnonymous(t *testing.T) {
	// given nobody is logged in
	db := testutil.DB(t)
	srv := testutil.NewMiddlewareProbeServer(t, db)

	// when I hit an admin-gated route
	resp := srv.Get(t, "/admin-only")

	// then I expect a 401
	resp.RequireStatus(t, 401)
}

func TestRequireAdmin_ReturnsForbidden_WhenUserIsOnlyAnOfficer(t *testing.T) {
	// given a logged-in officer with no admin access
	db := testutil.DB(t)
	srv := testutil.NewMiddlewareProbeServer(t, db)
	srv.LoginAs(t, "80351110224678912", "thundermane", testutil.AsOfficer())

	// when I hit an admin-gated route
	resp := srv.Get(t, "/admin-only")

	// then I expect a 403
	resp.RequireStatus(t, 403)
}

func TestRequireAdmin_AllowsRequest_WhenUserIsAdmin(t *testing.T) {
	// given a logged-in admin
	db := testutil.DB(t)
	srv := testutil.NewMiddlewareProbeServer(t, db)
	srv.LoginAs(t, "80351110224678912", "thundermane", testutil.AsAdmin())

	// when I hit an admin-gated route
	resp := srv.Get(t, "/admin-only")

	// then I expect a 200
	resp.RequireStatus(t, 200)
}
