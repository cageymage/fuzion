package applications_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
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

	TurnstileToken string `json:"turnstileToken"`
}

func validApplication() applicationBody {
	return applicationBody{
		TurnstileToken: testutil.ValidTurnstileToken,
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

		TurnstileToken: testutil.ValidTurnstileToken,
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

func TestSubmitApplication_PostsSummaryToDiscordWebhook(t *testing.T) {
	// given a server whose recruiting webhook is a fake Discord channel
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I submit the application
	resp := srv.Post(t, "/api/applications", validApplication())

	// then I expect a 201
	resp.RequireStatus(t, 201)

	// and exactly one message was posted to the webhook, summarising the applicant
	bodies := srv.Webhook.Bodies()
	if len(bodies) != 1 {
		t.Fatalf("expected exactly 1 webhook post, got %d", len(bodies))
	}
	var got struct {
		Embeds []struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Fields      []struct {
				Name   string `json:"name"`
				Value  string `json:"value"`
				Inline bool   `json:"inline"`
			} `json:"fields"`
		} `json:"embeds"`
	}
	if err := json.Unmarshal(bodies[0], &got); err != nil {
		t.Fatalf("decode webhook body %q: %v", bodies[0], err)
	}
	if len(got.Embeds) != 1 {
		t.Fatalf("expected 1 embed, got %d in %s", len(got.Embeds), bodies[0])
	}
	embed := got.Embeds[0]
	if embed.Title != "New application: Thornleaf" {
		t.Errorf("expected title %q, got %q", "New application: Thornleaf", embed.Title)
	}
	if !strings.Contains(embed.Description, "review") {
		t.Errorf("expected description to point officers at the review queue, got %q", embed.Description)
	}
	type field struct{ Name, Value string }
	var gotFields []field
	for _, f := range embed.Fields {
		gotFields = append(gotFields, field{f.Name, f.Value})
	}
	wantFields := []field{
		{"Applicant", "Mira"},
		{"Character", "Thornleaf"},
		{"Class / Role", "Druid / healer"},
		{"Availability", "Tue/Thu 8-11pm ET"},
		{"Discord", "mira.heals"},
	}
	if diff := cmp.Diff(wantFields, gotFields); diff != "" {
		t.Errorf("unexpected embed fields (-want +got):\n%s", diff)
	}
}

func TestSubmitApplication_PostsEmbedLinkingToReviewPage_WhenRecruitingWebhookIsConfigured(t *testing.T) {
	// given a server whose recruiting webhook is a fake Discord channel
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I submit the application
	resp := srv.Post(t, "/api/applications", validApplication())

	// then I expect a 201
	resp.RequireStatus(t, 201)

	// and the posted embed links to the officer review page on the site
	bodies := srv.Webhook.Bodies()
	if len(bodies) != 1 {
		t.Fatalf("expected exactly 1 webhook post, got %d", len(bodies))
	}
	var got struct {
		Embeds []struct {
			URL string `json:"url"`
		} `json:"embeds"`
	}
	if err := json.Unmarshal(bodies[0], &got); err != nil {
		t.Fatalf("decode webhook body %q: %v", bodies[0], err)
	}
	if len(got.Embeds) != 1 {
		t.Fatalf("expected 1 embed, got %d in %s", len(got.Embeds), bodies[0])
	}
	if want := testutil.SiteBaseURL + "/officer/applications"; got.Embeds[0].URL != want {
		t.Errorf("expected embed url %q, got %q", want, got.Embeds[0].URL)
	}
}

func TestSubmitApplication_StillReturnsCreated_WhenDiscordWebhookFails(t *testing.T) {
	// given a recruiting webhook that answers 500
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.Webhook.Status = 500

	// when I submit the application
	resp := srv.Post(t, "/api/applications", validApplication())

	// then I expect a 201 and the application is stored
	resp.RequireStatus(t, 201)
	if got := countApplications(t, db); got != 1 {
		t.Errorf("expected 1 stored application, got %d", got)
	}
}

func TestSubmitApplication_DoesNotCallDiscord_WhenWebhookURLIsEmpty(t *testing.T) {
	// given a deploy without DISCORD_RECRUITING_WEBHOOK_URL
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db, testutil.WithoutRecruitingWebhook())

	// when I submit the application
	resp := srv.Post(t, "/api/applications", validApplication())

	// then I expect a 201 and no message reached the fake Discord
	resp.RequireStatus(t, 201)
	if got := len(srv.Webhook.Bodies()); got != 0 {
		t.Errorf("expected no webhook posts, got %d", got)
	}
}

func TestSubmitApplication_SendsSecretTokenAndClientIPToCloudflare_WhenTokenIsValid(t *testing.T) {
	// given a deploy with Turnstile configured
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I submit the application with a valid token
	resp := srv.Post(t, "/api/applications", validApplication())

	// then I expect a 201 and exactly one siteverify call carrying the secret, the token and an IP
	resp.RequireStatus(t, 201)
	requests := srv.Turnstile.Requests()
	if len(requests) != 1 {
		t.Fatalf("expected exactly 1 siteverify request, got %d", len(requests))
	}
	got := requests[0]
	if got.Secret != testutil.TurnstileSecretKey || got.Response != testutil.ValidTurnstileToken {
		t.Errorf("expected secret %q and response %q, got secret %q and response %q",
			testutil.TurnstileSecretKey, testutil.ValidTurnstileToken, got.Secret, got.Response)
	}
	if got.RemoteIP != "127.0.0.1" {
		t.Errorf("expected remoteip %q, got %q", "127.0.0.1", got.RemoteIP)
	}
}

func TestSubmitApplication_ReturnsBadRequest_WhenTurnstileTokenIsInvalid(t *testing.T) {
	// given a token Cloudflare rejects
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	body := validApplication()
	body.TurnstileToken = "forged-token"

	// when I submit the application
	resp := srv.Post(t, "/api/applications", body)

	// then I expect a 400 naming the token, nothing stored and no Discord ping
	resp.RequireStatus(t, 400)
	var got errorJSON
	resp.DecodeJSON(t, &got)
	if diff := cmp.Diff(errorJSON{Error: "turnstileToken: verification failed"}, got); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
	if n := countApplications(t, db); n != 0 {
		t.Errorf("expected no stored applications, found %d", n)
	}
	if n := len(srv.Webhook.Bodies()); n != 0 {
		t.Errorf("expected no webhook posts, got %d", n)
	}
}

func TestSubmitApplication_ReturnsBadRequest_WhenTurnstileTokenIsMissing(t *testing.T) {
	// given an application with no Turnstile token
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	body := validApplication()
	body.TurnstileToken = ""

	// when I submit the application
	resp := srv.Post(t, "/api/applications", body)

	// then I expect a 400 saying the token is required, and Cloudflare was never called
	resp.RequireStatus(t, 400)
	var got errorJSON
	resp.DecodeJSON(t, &got)
	if diff := cmp.Diff(errorJSON{Error: "turnstileToken: is required"}, got); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
	if n := len(srv.Turnstile.Requests()); n != 0 {
		t.Errorf("expected no siteverify requests, got %d", n)
	}
	if n := countApplications(t, db); n != 0 {
		t.Errorf("expected no stored applications, found %d", n)
	}
}

func TestSubmitApplication_ReturnsBadGateway_WhenCloudflareIsUnreachable(t *testing.T) {
	// given a Cloudflare siteverify endpoint that answers 500
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.Turnstile.Status = 500

	// when I submit the application
	resp := srv.Post(t, "/api/applications", validApplication())

	// then I expect a 502 and nothing stored
	resp.RequireStatus(t, 502)
	var got errorJSON
	resp.DecodeJSON(t, &got)
	if diff := cmp.Diff(errorJSON{Error: "verification service unavailable"}, got); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
	if n := countApplications(t, db); n != 0 {
		t.Errorf("expected no stored applications, found %d", n)
	}
}

func TestSubmitApplication_SkipsVerification_WhenTurnstileSecretIsNotConfigured(t *testing.T) {
	// given a deploy without TURNSTILE_SECRET_KEY
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db, testutil.WithoutTurnstile())
	body := validApplication()
	body.TurnstileToken = ""

	// when I submit the application without a token
	resp := srv.Post(t, "/api/applications", body)

	// then I expect a 201 and Cloudflare was never called
	resp.RequireStatus(t, 201)
	if n := len(srv.Turnstile.Requests()); n != 0 {
		t.Errorf("expected no siteverify requests, got %d", n)
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

type reviewedApplicationJSON struct {
	ID            string  `json:"id"`
	ApplicantName string  `json:"applicantName"`
	CharacterName string  `json:"characterName"`
	Class         string  `json:"class"`
	Role          string  `json:"role"`
	Availability  string  `json:"availability"`
	DiscordHandle string  `json:"discordHandle"`
	Notes         string  `json:"notes"`
	Status        string  `json:"status"`
	SubmittedAt   string  `json:"submittedAt"`
	ReviewedBy    *string `json:"reviewedBy"`
	ReviewedAt    *string `json:"reviewedAt"`
	ReviewNote    string  `json:"reviewNote"`
}

func insertApplication(t *testing.T, db *sqlx.DB, characterName, status string, submittedAt time.Time) string {
	t.Helper()
	id := uuid.NewString()
	db.MustExec(`
		INSERT INTO applications (id, applicant_name, character_name, class, role, availability, discord_handle, status, submitted_at)
		VALUES ($1, 'Mira', $2, 'Druid', 'healer', 'Tue/Thu 8-11pm ET', 'mira.heals', $3, $4)`,
		id, characterName, status, submittedAt)
	return id
}

func TestListApplications_ReturnsPendingOnly_WhenStatusFilterIsPending(t *testing.T) {
	// given one pending, one accepted and one declined application, and an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	base := time.Date(2026, 3, 1, 18, 0, 0, 0, time.UTC)
	pendingID := insertApplication(t, db, "Thornleaf", "pending", base)
	insertApplication(t, db, "Ashfall", "accepted", base.Add(time.Hour))
	insertApplication(t, db, "Bramble", "declined", base.Add(2*time.Hour))
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I list applications filtered to pending
	resp := srv.Get(t, "/api/applications?status=pending")

	// then I expect a 200 with only the pending application, in the full shape
	resp.RequireStatus(t, 200)
	var got []reviewedApplicationJSON
	resp.DecodeJSON(t, &got)
	want := []reviewedApplicationJSON{{
		ID:            pendingID,
		ApplicantName: "Mira",
		CharacterName: "Thornleaf",
		Class:         "Druid",
		Role:          "healer",
		Availability:  "Tue/Thu 8-11pm ET",
		DiscordHandle: "mira.heals",
		Status:        "pending",
		SubmittedAt:   "2026-03-01T18:00:00Z",
	}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected applications (-want +got):\n%s", diff)
	}
}

func TestListApplications_ReturnsNewestFirst_WhenNoStatusFilterIsGiven(t *testing.T) {
	// given three applications submitted at different times, and an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	base := time.Date(2026, 3, 1, 18, 0, 0, 0, time.UTC)
	insertApplication(t, db, "Oldest", "pending", base)
	insertApplication(t, db, "Newest", "accepted", base.Add(2*time.Hour))
	insertApplication(t, db, "Middle", "declined", base.Add(time.Hour))
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I list applications with no filter
	resp := srv.Get(t, "/api/applications")

	// then I expect all three, newest first
	resp.RequireStatus(t, 200)
	var got []reviewedApplicationJSON
	resp.DecodeJSON(t, &got)
	var names []string
	for _, a := range got {
		names = append(names, a.CharacterName)
	}
	if diff := cmp.Diff([]string{"Newest", "Middle", "Oldest"}, names); diff != "" {
		t.Errorf("unexpected order (-want +got):\n%s", diff)
	}
}

func TestListApplications_ReturnsBadRequest_WhenStatusIsUnknown(t *testing.T) {
	// given an officer
	srv := testutil.NewServer(t, testutil.DB(t))
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I list applications with an unknown status
	resp := srv.Get(t, "/api/applications?status=archived")

	// then I expect a 400 naming the status field
	resp.RequireStatus(t, 400)
	var got errorJSON
	resp.DecodeJSON(t, &got)
	if diff := cmp.Diff(errorJSON{Error: "status: must be one of pending, accepted, declined"}, got); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
}

func TestListApplications_ReturnsForbidden_WhenCallerIsNotAnOfficer(t *testing.T) {
	// given a logged-in member who is not an officer
	srv := testutil.NewServer(t, testutil.DB(t))
	srv.LoginAs(t, "member-1", "Member")

	// when I list applications
	resp := srv.Get(t, "/api/applications")

	// then I expect a 403
	resp.RequireStatus(t, 403)
}

func TestListApplications_ReturnsUnauthorized_WhenCallerIsAnonymous(t *testing.T) {
	// given nobody is logged in
	srv := testutil.NewServer(t, testutil.DB(t))

	// when I list applications
	resp := srv.Get(t, "/api/applications")

	// then I expect a 401
	resp.RequireStatus(t, 401)
}

func TestGetApplication_ReturnsTheApplication_WhenOfficerRequestsItsID(t *testing.T) {
	// given a stored application and an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	id := insertApplication(t, db, "Thornleaf", "pending", time.Date(2026, 3, 1, 18, 0, 0, 0, time.UTC))
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I get the application by id
	resp := srv.Get(t, "/api/applications/"+id)

	// then I expect a 200 with the full application
	resp.RequireStatus(t, 200)
	var got reviewedApplicationJSON
	resp.DecodeJSON(t, &got)
	want := reviewedApplicationJSON{
		ID:            id,
		ApplicantName: "Mira",
		CharacterName: "Thornleaf",
		Class:         "Druid",
		Role:          "healer",
		Availability:  "Tue/Thu 8-11pm ET",
		DiscordHandle: "mira.heals",
		Status:        "pending",
		SubmittedAt:   "2026-03-01T18:00:00Z",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected application (-want +got):\n%s", diff)
	}
}

func TestGetApplication_ReturnsNotFound_WhenIDDoesNotExist(t *testing.T) {
	// given an officer and no applications
	srv := testutil.NewServer(t, testutil.DB(t))
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I get an application by an id nobody has
	resp := srv.Get(t, "/api/applications/"+uuid.NewString())

	// then I expect a 404
	resp.RequireStatus(t, 404)
}

func TestGetApplication_ReturnsBadRequest_WhenIDIsNotAUUID(t *testing.T) {
	// given an officer
	srv := testutil.NewServer(t, testutil.DB(t))
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I get an application with an id that is not a UUID
	resp := srv.Get(t, "/api/applications/not-a-uuid")

	// then I expect a 400 naming the id field
	resp.RequireStatus(t, 400)
	var got errorJSON
	resp.DecodeJSON(t, &got)
	if diff := cmp.Diff(errorJSON{Error: "id: must be a valid UUID"}, got); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
}

func TestGetApplication_ReturnsForbidden_WhenCallerIsNotAnOfficer(t *testing.T) {
	// given a stored application and a logged-in member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	id := insertApplication(t, db, "Thornleaf", "pending", time.Now())
	srv.LoginAs(t, "member-1", "Member")

	// when I get the application
	resp := srv.Get(t, "/api/applications/"+id)

	// then I expect a 403
	resp.RequireStatus(t, 403)
}

func TestGetApplication_ReturnsUnauthorized_WhenCallerIsAnonymous(t *testing.T) {
	// given a stored application and nobody logged in
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	id := insertApplication(t, db, "Thornleaf", "pending", time.Now())

	// when I get the application
	resp := srv.Get(t, "/api/applications/"+id)

	// then I expect a 401
	resp.RequireStatus(t, 401)
}

func TestReviewApplication_MarksAccepted_AndRecordsReviewer(t *testing.T) {
	// given a pending application and an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	id := insertApplication(t, db, "Thornleaf", "pending", time.Date(2026, 3, 1, 18, 0, 0, 0, time.UTC))
	officer := srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I accept it with a note
	resp := srv.Patch(t, "/api/applications/"+id, map[string]string{"status": "accepted", "reviewNote": "Great logs, invite for Tuesday."})

	// then I expect a 200 with the updated application, stamped with me and the current time
	resp.RequireStatus(t, 200)
	var got reviewedApplicationJSON
	resp.DecodeJSON(t, &got)
	reviewer := officer.ID.String()
	reviewedAt := "2026-03-14T20:00:00Z"
	want := reviewedApplicationJSON{
		ID:            id,
		ApplicantName: "Mira",
		CharacterName: "Thornleaf",
		Class:         "Druid",
		Role:          "healer",
		Availability:  "Tue/Thu 8-11pm ET",
		DiscordHandle: "mira.heals",
		Status:        "accepted",
		SubmittedAt:   "2026-03-01T18:00:00Z",
		ReviewedBy:    &reviewer,
		ReviewedAt:    &reviewedAt,
		ReviewNote:    "Great logs, invite for Tuesday.",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected application (-want +got):\n%s", diff)
	}
}

func TestReviewApplication_OverwritesEarlierReview_WhenAlreadyReviewed(t *testing.T) {
	// given an application an officer already accepted
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	id := insertApplication(t, db, "Thornleaf", "pending", time.Now())
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	srv.Patch(t, "/api/applications/"+id, map[string]string{"status": "accepted", "reviewNote": "Looks good."}).RequireStatus(t, 200)

	// when I change my mind and decline it without a note
	resp := srv.Patch(t, "/api/applications/"+id, map[string]string{"status": "declined"})

	// then I expect a 200 with the declined status and the earlier note cleared
	resp.RequireStatus(t, 200)
	var got reviewedApplicationJSON
	resp.DecodeJSON(t, &got)
	if got.Status != "declined" {
		t.Errorf("expected status %q, got %q", "declined", got.Status)
	}
	if got.ReviewNote != "" {
		t.Errorf("expected the earlier review note to be cleared, got %q", got.ReviewNote)
	}
}

func TestReviewApplication_ReturnsBadRequest_WhenStatusIsNotAcceptedOrDeclined(t *testing.T) {
	// given a pending application and an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	id := insertApplication(t, db, "Thornleaf", "pending", time.Now())
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I try to set the status back to pending
	resp := srv.Patch(t, "/api/applications/"+id, map[string]string{"status": "pending"})

	// then I expect a 400 naming the status field
	resp.RequireStatus(t, 400)
	var got errorJSON
	resp.DecodeJSON(t, &got)
	if diff := cmp.Diff(errorJSON{Error: "status: must be one of accepted, declined"}, got); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
}

func TestReviewApplication_ReturnsBadRequest_WhenReviewNoteExceedsLimit(t *testing.T) {
	// given a pending application and an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	id := insertApplication(t, db, "Thornleaf", "pending", time.Now())
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I review it with a note one character over the 2000 limit
	resp := srv.Patch(t, "/api/applications/"+id, map[string]string{"status": "accepted", "reviewNote": strings.Repeat("a", 2001)})

	// then I expect a 400 naming the reviewNote field
	resp.RequireStatus(t, 400)
	var got errorJSON
	resp.DecodeJSON(t, &got)
	if diff := cmp.Diff(errorJSON{Error: "reviewNote: must be at most 2000 characters"}, got); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
}

func TestReviewApplication_ReturnsNotFound_WhenIDDoesNotExist(t *testing.T) {
	// given an officer and no applications
	srv := testutil.NewServer(t, testutil.DB(t))
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I review an application nobody has
	resp := srv.Patch(t, "/api/applications/"+uuid.NewString(), map[string]string{"status": "accepted"})

	// then I expect a 404
	resp.RequireStatus(t, 404)
}

func TestReviewApplication_ReturnsBadRequest_WhenIDIsNotAUUID(t *testing.T) {
	// given an officer
	srv := testutil.NewServer(t, testutil.DB(t))
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I review an application with an id that is not a UUID
	resp := srv.Patch(t, "/api/applications/not-a-uuid", map[string]string{"status": "accepted"})

	// then I expect a 400 naming the id field
	resp.RequireStatus(t, 400)
	var got errorJSON
	resp.DecodeJSON(t, &got)
	if diff := cmp.Diff(errorJSON{Error: "id: must be a valid UUID"}, got); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
}

func TestReviewApplication_ReturnsForbidden_WhenCallerIsNotAnOfficer(t *testing.T) {
	// given a pending application and a logged-in member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	id := insertApplication(t, db, "Thornleaf", "pending", time.Now())
	srv.LoginAs(t, "member-1", "Member")

	// when I try to accept it
	resp := srv.Patch(t, "/api/applications/"+id, map[string]string{"status": "accepted"})

	// then I expect a 403
	resp.RequireStatus(t, 403)
}

func TestReviewApplication_ReturnsUnauthorized_WhenCallerIsAnonymous(t *testing.T) {
	// given a pending application and nobody logged in
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	id := insertApplication(t, db, "Thornleaf", "pending", time.Now())

	// when I try to accept it
	resp := srv.Patch(t, "/api/applications/"+id, map[string]string{"status": "accepted"})

	// then I expect a 401
	resp.RequireStatus(t, 401)
}
