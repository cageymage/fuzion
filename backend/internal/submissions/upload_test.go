package submissions_test

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jmoiron/sqlx"

	"github.com/cageymage/fuzion/backend/internal/auth"
	"github.com/cageymage/fuzion/backend/internal/submissions"
	"github.com/cageymage/fuzion/backend/internal/testutil"
)

func requireError(t *testing.T, resp testutil.Response, want string) {
	t.Helper()
	var body map[string]string
	resp.DecodeJSON(t, &body)
	if diff := cmp.Diff(map[string]string{"error": want}, body); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
}

func requireNothingStored(t *testing.T, db *sqlx.DB) {
	t.Helper()
	if diff := cmp.Diff([]storedRow(nil), storedRows(t, db)); diff != "" {
		t.Errorf("expected no submissions (-want +got):\n%s", diff)
	}
}

// storedAt is the row an upload made at FixedNow by user should leave.
func storedAt(user auth.User, filename string, width, height, byteSize int, description string) storedRow {
	return storedRow{
		UserID: user.ID.String(), OriginalFilename: filename, ContentType: "image/png", Width: width, Height: height,
		ByteSize: byteSize, Description: description, AttestationAcceptedAt: testutil.FixedNow,
		AttestationVersion: submissions.AttestationVersion, CreatedAt: testutil.FixedNow,
	}
}

// useSessionOf switches the client back to a user LoginAs created earlier.
func useSessionOf(t *testing.T, srv *testutil.Server, user auth.User) {
	t.Helper()
	srv.SetCookie(t, &http.Cookie{Name: auth.SessionCookieName, Value: "session-for-" + user.DiscordID, Path: "/"})
}

func ptr[T any](v T) *T { return &v }

func TestUploadSubmission_ReturnsCreated_WhenMemberSubmitsImagesWithAttestation(t *testing.T) {
	// given a guild member with two screenshots
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	member := loginMember(t, srv)
	first, second := solidPNG(t, 64, 32, 1), solidPNG(t, 20, 40, 2)

	// when I submit both, with a description for the first, having accepted the attestation
	resp := srv.PostMultipart(t, uploadPath,
		acceptedMetadata(t, " Onyxia down! ", ""),
		filePart("onyxia.png", first),
		filePart("raid.png", second),
	)

	// then I expect a 201 with a result per file, and both stored with the attestation
	resp.RequireStatus(t, http.StatusCreated)
	var got uploadResponseJSON
	resp.DecodeJSON(t, &got)
	if len(got.Results) != 2 || got.Results[0].Submission == nil || got.Results[1].Submission == nil {
		t.Fatalf("expected two stored results, got %+v", got)
	}
	want := uploadResponseJSON{Results: []fileResultJSON{
		{Filename: "onyxia.png", Submission: ptr(expectedSubmission(got.Results[0].Submission.ID, member, "onyxia.png", "Onyxia down!", 64, 32, len(first)))},
		{Filename: "raid.png", Submission: ptr(expectedSubmission(got.Results[1].Submission.ID, member, "raid.png", "", 20, 40, len(second)))},
	}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected response (-want +got):\n%s", diff)
	}
	wantRows := []storedRow{
		storedAt(member, "onyxia.png", 64, 32, len(first), "Onyxia down!"),
		storedAt(member, "raid.png", 20, 40, len(second), ""),
	}
	if diff := cmp.Diff(wantRows, storedRows(t, db)); diff != "" {
		t.Errorf("unexpected stored submissions (-want +got):\n%s", diff)
	}
}

func TestUploadSubmission_StoresTheExactOriginalBytes(t *testing.T) {
	// given a guild member with a screenshot
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	loginMember(t, srv)
	original := solidPNG(t, 50, 50, 3)

	// when I submit it
	resp := srv.PostMultipart(t, uploadPath, acceptedMetadata(t), filePart("shot.png", original))

	// then I expect the stored original to be byte-for-byte what I sent, not a re-encode
	resp.RequireStatus(t, http.StatusCreated)
	var stored []byte
	if err := db.Get(&stored, `SELECT original_bytes FROM submissions`); err != nil {
		t.Fatalf("select original: %v", err)
	}
	if !bytes.Equal(original, stored) {
		t.Errorf("stored original differs from the upload (%d vs %d bytes)", len(stored), len(original))
	}
}

func TestUploadSubmission_ReturnsUnauthorized_WhenNotLoggedIn(t *testing.T) {
	// given nobody is logged in
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I submit a screenshot
	resp := srv.PostMultipart(t, uploadPath, acceptedMetadata(t), filePart("shot.png", solidPNG(t, 8, 8, 1)))

	// then I expect a 401 and nothing stored
	resp.RequireStatus(t, http.StatusUnauthorized)
	requireNothingStored(t, db)
}

func TestUploadSubmission_ReturnsForbidden_WhenUserIsNotGuildMember(t *testing.T) {
	// given a logged-in user who is not in the guild's Discord server
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "80351110224678912", "outsider")

	// when I submit a screenshot
	resp := srv.PostMultipart(t, uploadPath, acceptedMetadata(t), filePart("shot.png", solidPNG(t, 8, 8, 1)))

	// then I expect a 403 and nothing stored
	resp.RequireStatus(t, http.StatusForbidden)
	requireError(t, resp, "guild discord server membership required")
	requireNothingStored(t, db)
}

func TestUploadSubmission_ReturnsBadRequest_WhenAttestationMissing(t *testing.T) {
	// given a guild member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	loginMember(t, srv)

	// when I submit without accepting the attestation
	resp := srv.PostMultipart(t, uploadPath,
		metadataPart(t, false, submissions.AttestationVersion),
		filePart("shot.png", solidPNG(t, 8, 8, 1)),
	)

	// then I expect a 400 naming the attestation and nothing stored
	resp.RequireStatus(t, http.StatusBadRequest)
	requireError(t, resp, "attestationAccepted: must be accepted")
	requireNothingStored(t, db)
}

func TestUploadSubmission_ReturnsBadRequest_WhenAttestationVersionIsOutOfDate(t *testing.T) {
	// given a guild member whose page still shows an older wording
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	loginMember(t, srv)

	// when I submit having accepted that older wording
	resp := srv.PostMultipart(t, uploadPath,
		metadataPart(t, true, "2020-01-01"),
		filePart("shot.png", solidPNG(t, 8, 8, 1)),
	)

	// then I expect a 400 asking me to reload, and nothing stored
	resp.RequireStatus(t, http.StatusBadRequest)
	requireError(t, resp, "attestationVersion: is out of date; reload the page and accept the current wording")
	requireNothingStored(t, db)
}

func TestUploadSubmission_ReturnsBadRequest_WhenBatchHasMoreThanTenFiles(t *testing.T) {
	// given a guild member with eleven screenshots
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	loginMember(t, srv)
	parts := []testutil.MultipartPart{acceptedMetadata(t)}
	for i := range 11 {
		parts = append(parts, filePart("shot.png", solidPNG(t, 8, 8, uint8(i))))
	}

	// when I submit all eleven at once
	resp := srv.PostMultipart(t, uploadPath, parts...)

	// then I expect a 400 and nothing stored
	resp.RequireStatus(t, http.StatusBadRequest)
	requireError(t, resp, "files: at most 10 images per submission")
	requireNothingStored(t, db)
}

func TestUploadSubmission_StoresTheSniffedType_WhenDeclaredTypeIsWrong(t *testing.T) {
	// given a guild member with a PNG the browser labelled as a GIF
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	member := loginMember(t, srv)
	data := solidPNG(t, 8, 8, 1)
	mislabelled := testutil.MultipartPart{Field: "files", Filename: "shot.gif", ContentType: "image/gif", Data: data}

	// when I submit it
	resp := srv.PostMultipart(t, uploadPath, acceptedMetadata(t), mislabelled)

	// then I expect it stored as the PNG it really is
	resp.RequireStatus(t, http.StatusCreated)
	if diff := cmp.Diff([]storedRow{storedAt(member, "shot.gif", 8, 8, len(data), "")}, storedRows(t, db)); diff != "" {
		t.Errorf("unexpected stored submissions (-want +got):\n%s", diff)
	}
}

func TestUploadSubmission_ReturnsBadRequest_WhenFileIsNotAnImage(t *testing.T) {
	// given a guild member with an HTML file named like a PNG
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	loginMember(t, srv)

	// when I submit it
	resp := srv.PostMultipart(t, uploadPath, acceptedMetadata(t), filePart("shot.png", []byte("<html><script>alert(1)</script></html>")))

	// then I expect a 400 with the reason against the file, and nothing stored
	resp.RequireStatus(t, http.StatusBadRequest)
	var got uploadResponseJSON
	resp.DecodeJSON(t, &got)
	want := uploadResponseJSON{Results: []fileResultJSON{{Filename: "shot.png", Error: "must be a PNG, JPEG, WebP or GIF image"}}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected response (-want +got):\n%s", diff)
	}
	requireNothingStored(t, db)
}

func TestUploadSubmission_ReturnsBadRequest_WhenImageExceedsPixelLimit(t *testing.T) {
	// given a guild member with a PNG whose header claims 10000×10000 pixels
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	loginMember(t, srv)

	// when I submit it
	resp := srv.PostMultipart(t, uploadPath, acceptedMetadata(t), filePart("huge.png", pngHeaderOnly(10000, 10000)))

	// then I expect a 400 before any full decode, and nothing stored
	resp.RequireStatus(t, http.StatusBadRequest)
	var got uploadResponseJSON
	resp.DecodeJSON(t, &got)
	want := uploadResponseJSON{Results: []fileResultJSON{{Filename: "huge.png", Error: "is 10000×10000 pixels; images must be at most 50 megapixels"}}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected response (-want +got):\n%s", diff)
	}
	requireNothingStored(t, db)
}

func TestUploadSubmission_RejectsOnlyTheOversizedFile_WhenOneFileIsOverTheSizeCap(t *testing.T) {
	// given a guild member with a small screenshot and a file just over 20 MiB
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	member := loginMember(t, srv)
	small := solidPNG(t, 8, 8, 1)
	oversized := append(solidPNG(t, 8, 8, 9), make([]byte, submissions.MaxFileBytes)...)

	// when I submit both
	resp := srv.PostMultipart(t, uploadPath, acceptedMetadata(t), filePart("small.png", small), filePart("big.png", oversized))

	// then I expect a 201 that stores the small one and explains the big one
	resp.RequireStatus(t, http.StatusCreated)
	var got uploadResponseJSON
	resp.DecodeJSON(t, &got)
	if len(got.Results) != 2 || got.Results[0].Submission == nil {
		t.Fatalf("expected the first file stored, got %+v", got)
	}
	want := uploadResponseJSON{Results: []fileResultJSON{
		{Filename: "small.png", Submission: ptr(expectedSubmission(got.Results[0].Submission.ID, member, "small.png", "", 8, 8, len(small)))},
		{Filename: "big.png", Error: "must be at most 20 MiB"},
	}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected response (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]storedRow{storedAt(member, "small.png", 8, 8, len(small), "")}, storedRows(t, db)); diff != "" {
		t.Errorf("unexpected stored submissions (-want +got):\n%s", diff)
	}
}

func TestUploadSubmission_ReturnsBadRequest_WhenDescriptionIsTooLong(t *testing.T) {
	// given a guild member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	loginMember(t, srv)

	// when I submit a screenshot with a 501-character description
	resp := srv.PostMultipart(t, uploadPath, acceptedMetadata(t, strings.Repeat("a", 501)), filePart("shot.png", solidPNG(t, 8, 8, 1)))

	// then I expect a 400 naming the description, and nothing stored
	resp.RequireStatus(t, http.StatusBadRequest)
	requireError(t, resp, "descriptions[0]: must be at most 500 characters")
	requireNothingStored(t, db)
}

func TestUploadSubmission_ReturnsConflict_WhenSameUserUploadsDuplicateFile(t *testing.T) {
	// given a guild member who already submitted a screenshot
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	member := loginMember(t, srv)
	shot := solidPNG(t, 8, 8, 1)
	srv.PostMultipart(t, uploadPath, acceptedMetadata(t), filePart("shot.png", shot)).RequireStatus(t, http.StatusCreated)

	// when I submit the same file again under another name
	resp := srv.PostMultipart(t, uploadPath, acceptedMetadata(t), filePart("again.png", shot))

	// then I expect a 409 and still only the first copy stored
	resp.RequireStatus(t, http.StatusConflict)
	var got uploadResponseJSON
	resp.DecodeJSON(t, &got)
	want := uploadResponseJSON{Results: []fileResultJSON{{Filename: "again.png", Error: "you have already submitted this image"}}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected response (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]storedRow{storedAt(member, "shot.png", 8, 8, len(shot), "")}, storedRows(t, db)); diff != "" {
		t.Errorf("unexpected stored submissions (-want +got):\n%s", diff)
	}
}

func TestUploadSubmission_AcceptsTheSameImage_WhenADifferentMemberSubmitsIt(t *testing.T) {
	// given another member already submitted a screenshot
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	shot := solidPNG(t, 8, 8, 1)
	other := srv.LoginAs(t, "22222222222222222", "othermember", testutil.InDiscordServer())
	srv.PostMultipart(t, uploadPath, acceptedMetadata(t), filePart("theirs.png", shot)).RequireStatus(t, http.StatusCreated)
	member := loginMember(t, srv)

	// when I submit the same file
	resp := srv.PostMultipart(t, uploadPath, acceptedMetadata(t), filePart("mine.png", shot))

	// then I expect it stored for me too, since duplicates are refused only per member
	resp.RequireStatus(t, http.StatusCreated)
	want := []storedRow{storedAt(member, "mine.png", 8, 8, len(shot), ""), storedAt(other, "theirs.png", 8, 8, len(shot), "")}
	if diff := cmp.Diff(want, storedRows(t, db)); diff != "" {
		t.Errorf("unexpected stored submissions (-want +got):\n%s", diff)
	}
}

func TestUploadSubmission_ReturnsTooManyRequests_WhenDailyFileQuotaExceeded(t *testing.T) {
	// given a guild member who submitted 20 images in the last 24 hours
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	member := loginMember(t, srv)
	for i := range submissions.DailyFileQuota {
		insertSubmission(t, db, member.ID, "earlier.png", []byte{byte(i)}, 1, testutil.FixedNow.Add(-time.Hour))
	}
	before := storedRows(t, db)

	// when I submit one more
	resp := srv.PostMultipart(t, uploadPath, acceptedMetadata(t), filePart("one-more.png", solidPNG(t, 8, 8, 1)))

	// then I expect a 429 and nothing new stored
	resp.RequireStatus(t, http.StatusTooManyRequests)
	requireError(t, resp, "you can submit at most 20 images per 24 hours; 0 left")
	if diff := cmp.Diff(before, storedRows(t, db)); diff != "" {
		t.Errorf("submissions changed (-want +got):\n%s", diff)
	}
}

func TestUploadSubmission_ReturnsTooManyRequests_WhenDailyByteQuotaExceeded(t *testing.T) {
	// given a guild member who submitted 100 MiB in the last 24 hours
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	member := loginMember(t, srv)
	insertSubmission(t, db, member.ID, "earlier.png", []byte{1}, submissions.DailyByteQuota, testutil.FixedNow.Add(-time.Hour))
	before := storedRows(t, db)

	// when I submit one more small image
	resp := srv.PostMultipart(t, uploadPath, acceptedMetadata(t), filePart("one-more.png", solidPNG(t, 8, 8, 1)))

	// then I expect a 429 and nothing new stored
	resp.RequireStatus(t, http.StatusTooManyRequests)
	requireError(t, resp, "you can submit at most 100 MiB per 24 hours")
	if diff := cmp.Diff(before, storedRows(t, db)); diff != "" {
		t.Errorf("submissions changed (-want +got):\n%s", diff)
	}
}

func TestUploadSubmission_IgnoresOlderSubmissions_WhenCountingTheDailyQuota(t *testing.T) {
	// given a guild member who submitted 20 images just over 24 hours ago
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	member := loginMember(t, srv)
	for i := range submissions.DailyFileQuota {
		insertSubmission(t, db, member.ID, "yesterday.png", []byte{byte(i)}, 1, testutil.FixedNow.Add(-25*time.Hour))
	}

	// when I submit a new image
	resp := srv.PostMultipart(t, uploadPath, acceptedMetadata(t), filePart("today.png", solidPNG(t, 8, 8, 1)))

	// then I expect it accepted, since the window is a rolling 24 hours
	resp.RequireStatus(t, http.StatusCreated)
}

func TestUploadSubmission_StopsCountingDeletedSubmissions_WhenCheckingTheDailyQuota(t *testing.T) {
	// given a guild member at the daily quota, one of whose submissions an officer deleted
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	member := loginMember(t, srv)
	spam := insertSubmission(t, db, member.ID, "spam.png", []byte{0}, 1, testutil.FixedNow.Add(-time.Hour))
	for i := 1; i < submissions.DailyFileQuota; i++ {
		insertSubmission(t, db, member.ID, "earlier.png", []byte{byte(i)}, 1, testutil.FixedNow.Add(-time.Hour))
	}
	loginOfficer(t, srv)
	srv.Delete(t, "/api/submissions/"+spam.String()).RequireStatus(t, http.StatusNoContent)
	useSessionOf(t, srv, member)

	// when I submit one image
	resp := srv.PostMultipart(t, uploadPath, acceptedMetadata(t), filePart("new.png", solidPNG(t, 8, 8, 1)))

	// then I expect it accepted, since the deleted row no longer counts
	resp.RequireStatus(t, http.StatusCreated)
}

func TestUploadSubmission_ReturnsServiceUnavailable_WhenSubmissionsAreDisabled(t *testing.T) {
	// given a deploy with SUBMISSIONS_ENABLED=false and a guild member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db, testutil.WithSubmissionsDisabled())
	loginMember(t, srv)

	// when I submit a screenshot
	resp := srv.PostMultipart(t, uploadPath, acceptedMetadata(t), filePart("shot.png", solidPNG(t, 8, 8, 1)))

	// then I expect a 503 with the closed message, and nothing stored
	resp.RequireStatus(t, http.StatusServiceUnavailable)
	requireError(t, resp, "submissions are temporarily closed")
	requireNothingStored(t, db)
}

func TestUploadSubmission_ReturnsServiceUnavailable_WhenStorageLimitIsReached(t *testing.T) {
	// given a 1 KiB storage ceiling that existing submissions already fill
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db, testutil.WithSubmissionStorageLimit(1024))
	member := loginMember(t, srv)
	insertSubmission(t, db, member.ID, "earlier.png", []byte{1}, 1024, testutil.FixedNow.Add(-48*time.Hour))
	before := storedRows(t, db)

	// when I submit a screenshot
	resp := srv.PostMultipart(t, uploadPath, acceptedMetadata(t), filePart("shot.png", solidPNG(t, 8, 8, 1)))

	// then I expect a 503 saying storage is full, and nothing new stored
	resp.RequireStatus(t, http.StatusServiceUnavailable)
	requireError(t, resp, "submissions are temporarily closed: storage is full")
	if diff := cmp.Diff(before, storedRows(t, db)); diff != "" {
		t.Errorf("submissions changed (-want +got):\n%s", diff)
	}
}

func TestSubmissionStatus_ReportsOpen_WhenSubmissionsAreEnabled(t *testing.T) {
	// given a default deploy and nobody logged in
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I ask whether submissions are open
	resp := srv.Get(t, "/api/submissions/status")

	// then I expect them open, with the attestation wording's version
	resp.RequireStatus(t, http.StatusOK)
	var got map[string]any
	resp.DecodeJSON(t, &got)
	if diff := cmp.Diff(map[string]any{"open": true, "attestationVersion": submissions.AttestationVersion}, got); diff != "" {
		t.Errorf("unexpected status (-want +got):\n%s", diff)
	}
}

func TestSubmissionStatus_ReportsClosed_WhenSubmissionsAreDisabled(t *testing.T) {
	// given a deploy with SUBMISSIONS_ENABLED=false
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db, testutil.WithSubmissionsDisabled())

	// when I ask whether submissions are open
	resp := srv.Get(t, "/api/submissions/status")

	// then I expect them closed
	resp.RequireStatus(t, http.StatusOK)
	var got map[string]any
	resp.DecodeJSON(t, &got)
	if diff := cmp.Diff(map[string]any{"open": false, "attestationVersion": submissions.AttestationVersion}, got); diff != "" {
		t.Errorf("unexpected status (-want +got):\n%s", diff)
	}
}

func TestSubmissionStatus_ReportsClosed_WhenStorageIsFull(t *testing.T) {
	// given a 1 KiB storage ceiling that existing submissions already fill
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db, testutil.WithSubmissionStorageLimit(1024))
	member := loginMember(t, srv)
	insertSubmission(t, db, member.ID, "earlier.png", []byte{1}, 1024, testutil.FixedNow)

	// when I ask whether submissions are open
	resp := srv.Get(t, "/api/submissions/status")

	// then I expect them closed
	resp.RequireStatus(t, http.StatusOK)
	var got map[string]any
	resp.DecodeJSON(t, &got)
	if diff := cmp.Diff(map[string]any{"open": false, "attestationVersion": submissions.AttestationVersion}, got); diff != "" {
		t.Errorf("unexpected status (-want +got):\n%s", diff)
	}
}
