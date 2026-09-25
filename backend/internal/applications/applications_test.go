package applications_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jmoiron/sqlx"

	"github.com/cageymage/fuzion/backend/internal/testutil"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.Run(m))
}

type applicationBody struct {
	ApplicantName string `json:"applicantName"`
	CharacterName string `json:"characterName"`
	Class         string `json:"class"`
	Role          string `json:"role"`
	Availability  string `json:"availability"`
	DiscordHandle string `json:"discordHandle"`
	Notes         string `json:"notes"`
}

func validApplication() applicationBody {
	return applicationBody{
		ApplicantName: "Mira",
		CharacterName: "Thornleaf",
		Class:         "Druid",
		Role:          "healer",
		Availability:  "Tue/Thu 8-11pm ET",
		DiscordHandle: "mira.heals",
		Notes:         "Cleared Mythic Ansurek with my previous guild.",
	}
}

type errorJSON struct {
	Error string `json:"error"`
}

func countApplications(t *testing.T, db *sqlx.DB) int {
	t.Helper()
	var count int
	if err := db.Get(&count, `SELECT count(*) FROM applications`); err != nil {
		t.Fatalf("count applications: %v", err)
	}
	return count
}

func TestSubmitApplication_ReturnsCreated_WhenBodyIsValid(t *testing.T) {
	// given an applicant whose form fields have stray whitespace around them
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	body := applicationBody{
		ApplicantName: "  Mira ",
		CharacterName: " Thornleaf",
		Class:         "Druid ",
		Role:          " healer ",
		Availability:  "Tue/Thu 8-11pm ET\n",
		DiscordHandle: "\tmira.heals",
		Notes:         "  Cleared Mythic Ansurek with my previous guild.  ",
	}

	// when I submit the application
	resp := srv.Post(t, "/api/applications", body)

	// then I expect a 201 with a pending status, an id, and a submission time
	resp.RequireStatus(t, 201)
	var got struct {
		ID          string `json:"id"`
		Status      string `json:"status"`
		SubmittedAt string `json:"submittedAt"`
	}
	resp.DecodeJSON(t, &got)
	if got.ID == "" {
		t.Errorf("expected a generated id, got an empty one")
	}
	if got.Status != "pending" {
		t.Errorf("expected status %q, got %q", "pending", got.Status)
	}
	if _, err := time.Parse(time.RFC3339, got.SubmittedAt); err != nil {
		t.Errorf("expected submittedAt in RFC3339, got %q: %v", got.SubmittedAt, err)
	}

	// and the row is stored trimmed, with a pending status
	type applicationRow struct {
		ApplicantName string `db:"applicant_name"`
		CharacterName string `db:"character_name"`
		Class         string `db:"class"`
		Role          string `db:"role"`
		Availability  string `db:"availability"`
		DiscordHandle string `db:"discord_handle"`
		Notes         string `db:"notes"`
		Status        string `db:"status"`
	}
	var row applicationRow
	err := db.Get(&row, `
		SELECT applicant_name, character_name, class, role, availability, discord_handle, notes, status
		FROM applications
		WHERE id = $1`, got.ID)
	if err != nil {
		t.Fatalf("select submitted application %s: %v", got.ID, err)
	}
	want := applicationRow{
		ApplicantName: "Mira",
		CharacterName: "Thornleaf",
		Class:         "Druid",
		Role:          "healer",
		Availability:  "Tue/Thu 8-11pm ET",
		DiscordHandle: "mira.heals",
		Notes:         "Cleared Mythic Ansurek with my previous guild.",
		Status:        "pending",
	}
	if diff := cmp.Diff(want, row); diff != "" {
		t.Errorf("unexpected stored application (-want +got):\n%s", diff)
	}
}

func TestSubmitApplication_ReturnsBadRequest_WhenRoleIsInvalid(t *testing.T) {
	// given an application for a role the guild does not raid with
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	body := validApplication()
	body.Role = "bard"

	// when I submit the application
	resp := srv.Post(t, "/api/applications", body)

	// then I expect a 400 naming the role field, and nothing stored
	resp.RequireStatus(t, 400)
	var got errorJSON
	resp.DecodeJSON(t, &got)
	if diff := cmp.Diff(errorJSON{Error: "role: must be one of tank, healer, dps"}, got); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
	if n := countApplications(t, db); n != 0 {
		t.Errorf("expected no stored applications, found %d", n)
	}
}

func TestSubmitApplication_ReturnsBadRequest_WhenRequiredFieldIsBlank(t *testing.T) {
	tests := []struct {
		name      string
		blank     func(*applicationBody)
		wantError string
	}{
		{"applicantName is whitespace only", func(b *applicationBody) { b.ApplicantName = "   " }, "applicantName: is required"},
		{"characterName is whitespace only", func(b *applicationBody) { b.CharacterName = "   " }, "characterName: is required"},
		{"class is whitespace only", func(b *applicationBody) { b.Class = "   " }, "class: is required"},
		{"role is whitespace only", func(b *applicationBody) { b.Role = "   " }, "role: is required"},
		{"availability is whitespace only", func(b *applicationBody) { b.Availability = "   " }, "availability: is required"},
		{"discordHandle is whitespace only", func(b *applicationBody) { b.DiscordHandle = "   " }, "discordHandle: is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given an otherwise valid application with one required field left blank
			db := testutil.DB(t)
			srv := testutil.NewServer(t, db)
			body := validApplication()
			tt.blank(&body)

			// when I submit the application
			resp := srv.Post(t, "/api/applications", body)

			// then I expect a 400 naming the blank field, and nothing stored
			resp.RequireStatus(t, 400)
			var got errorJSON
			resp.DecodeJSON(t, &got)
			if diff := cmp.Diff(errorJSON{Error: tt.wantError}, got); diff != "" {
				t.Errorf("unexpected error body (-want +got):\n%s", diff)
			}
			if n := countApplications(t, db); n != 0 {
				t.Errorf("expected no stored applications, found %d", n)
			}
		})
	}
}

func TestSubmitApplication_ReturnsBadRequest_WhenNotesExceedLimit(t *testing.T) {
	// given an application whose notes are one character over the 2000 limit
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	body := validApplication()
	body.Notes = strings.Repeat("a", 2001)

	// when I submit the application
	resp := srv.Post(t, "/api/applications", body)

	// then I expect a 400 naming the notes field
	resp.RequireStatus(t, 400)
	var got errorJSON
	resp.DecodeJSON(t, &got)
	if diff := cmp.Diff(errorJSON{Error: "notes: must be at most 2000 characters"}, got); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
}

func TestSubmitApplication_ReturnsBadRequest_WhenDiscordHandleExceedsLimit(t *testing.T) {
	// given an application whose Discord handle is one character over the 64 limit
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	body := validApplication()
	body.DiscordHandle = strings.Repeat("a", 65)

	// when I submit the application
	resp := srv.Post(t, "/api/applications", body)

	// then I expect a 400 naming the discordHandle field
	resp.RequireStatus(t, 400)
	var got errorJSON
	resp.DecodeJSON(t, &got)
	if diff := cmp.Diff(errorJSON{Error: "discordHandle: must be at most 64 characters"}, got); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
}

func TestSubmitApplication_ReturnsBadRequest_WhenBodyIsNotJSON(t *testing.T) {
	// given a request body that is cut off mid-object
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I submit it as an application
	resp := srv.PostRaw(t, "/api/applications", `{"applicantName":`)

	// then I expect a 400 saying the body is not valid JSON
	resp.RequireStatus(t, 400)
	var got errorJSON
	resp.DecodeJSON(t, &got)
	if diff := cmp.Diff(errorJSON{Error: "body: must be valid JSON"}, got); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
}
