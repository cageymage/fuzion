package submissions_test

import (
	"bytes"
	"encoding/json"
	"image"
	"net/http"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "golang.org/x/image/webp"

	"github.com/cageymage/fuzion/backend/internal/testutil"
)

type pageJSON struct {
	Total  int              `json:"total"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
	Items  []submissionJSON `json:"items"`
}

// uploadAsMember submits one image as the member and returns its id.
func uploadAsMember(t *testing.T, srv *testutil.Server, filename string, data []byte) string {
	t.Helper()
	loginMember(t, srv)
	resp := srv.PostMultipart(t, uploadPath, acceptedMetadata(t), filePart(filename, data))
	resp.RequireStatus(t, http.StatusCreated)
	var got uploadResponseJSON
	resp.DecodeJSON(t, &got)
	return got.Results[0].Submission.ID
}

func storedFilenames(t *testing.T, db *sqlx.DB) []string {
	t.Helper()
	var names []string
	if err := db.Select(&names, `SELECT original_filename FROM submissions ORDER BY original_filename`); err != nil {
		t.Fatalf("select submission names: %v", err)
	}
	return names
}

func headers(resp testutil.Response, names ...string) map[string]string {
	got := map[string]string{}
	for _, name := range names {
		got[name] = resp.Header.Get(name)
	}
	return got
}

func TestListSubmissions_ReturnsPagedMetadata_WithoutImageBytes(t *testing.T) {
	// given three submissions made an hour apart
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	member := loginMember(t, srv)
	insertSubmission(t, db, member.ID, "oldest.png", []byte{1}, 10, testutil.FixedNow.Add(-2*time.Hour))
	middle := insertSubmission(t, db, member.ID, "middle.png", []byte{2}, 20, testutil.FixedNow.Add(-time.Hour)).String()
	newest := insertSubmission(t, db, member.ID, "newest.png", []byte{3}, 30, testutil.FixedNow).String()
	loginOfficer(t, srv)

	// when I list the first page of two
	resp := srv.Get(t, "/api/submissions?limit=2")

	// then I expect the two newest, the total, URLs instead of image bytes, and private headers
	resp.RequireStatus(t, http.StatusOK)
	if diff := cmp.Diff(map[string]string{"X-Content-Type-Options": "nosniff", "Cache-Control": "private"}, headers(resp, "X-Content-Type-Options", "Cache-Control")); diff != "" {
		t.Errorf("unexpected headers (-want +got):\n%s", diff)
	}
	var got pageJSON
	resp.DecodeJSON(t, &got)
	submitter := submitterJSON{ID: member.ID.String(), Username: "thundermane"}
	want := pageJSON{Total: 3, Limit: 2, Offset: 0, Items: []submissionJSON{
		{
			ID: newest, Submitter: submitter, OriginalFilename: "newest.png", ContentType: "image/png",
			Width: 4, Height: 4, ByteSize: 30, CreatedAt: testutil.FixedNow,
			ThumbURL: "/api/submissions/" + newest + "/thumb", FullURL: "/api/submissions/" + newest + "/full",
			DownloadURL: "/api/submissions/" + newest + "/download",
		},
		{
			ID: middle, Submitter: submitter, OriginalFilename: "middle.png", ContentType: "image/png",
			Width: 4, Height: 4, ByteSize: 20, CreatedAt: testutil.FixedNow.Add(-time.Hour),
			ThumbURL: "/api/submissions/" + middle + "/thumb", FullURL: "/api/submissions/" + middle + "/full",
			DownloadURL: "/api/submissions/" + middle + "/download",
		},
	}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected page (-want +got):\n%s", diff)
	}
	var raw struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(resp.Body, &raw); err != nil {
		t.Fatalf("decode raw page: %v", err)
	}
	wantKeys := []string{"byteSize", "contentType", "createdAt", "description", "downloadUrl", "fullUrl", "height", "id", "originalFilename", "submitter", "thumbUrl", "width"}
	var gotKeys []string
	for key := range raw.Items[0] {
		gotKeys = append(gotKeys, key)
	}
	if diff := cmp.Diff(wantKeys, gotKeys, cmpopts.SortSlices(func(a, b string) bool { return a < b })); diff != "" {
		t.Errorf("unexpected list item fields (-want +got):\n%s", diff)
	}
}

func TestListSubmissions_ReturnsTheNextPage_WhenOffsetIsGiven(t *testing.T) {
	// given three submissions made an hour apart
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	member := loginMember(t, srv)
	oldest := insertSubmission(t, db, member.ID, "oldest.png", []byte{1}, 10, testutil.FixedNow.Add(-2*time.Hour)).String()
	insertSubmission(t, db, member.ID, "middle.png", []byte{2}, 20, testutil.FixedNow.Add(-time.Hour))
	insertSubmission(t, db, member.ID, "newest.png", []byte{3}, 30, testutil.FixedNow)
	loginOfficer(t, srv)

	// when I ask for the second page of two
	resp := srv.Get(t, "/api/submissions?limit=2&offset=2")

	// then I expect only the oldest
	resp.RequireStatus(t, http.StatusOK)
	var got pageJSON
	resp.DecodeJSON(t, &got)
	want := pageJSON{Total: 3, Limit: 2, Offset: 2, Items: []submissionJSON{{
		ID: oldest, Submitter: submitterJSON{ID: member.ID.String(), Username: "thundermane"},
		OriginalFilename: "oldest.png", ContentType: "image/png", Width: 4, Height: 4, ByteSize: 10,
		CreatedAt: testutil.FixedNow.Add(-2 * time.Hour), ThumbURL: "/api/submissions/" + oldest + "/thumb",
		FullURL: "/api/submissions/" + oldest + "/full", DownloadURL: "/api/submissions/" + oldest + "/download",
	}}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected page (-want +got):\n%s", diff)
	}
}

func TestListSubmissions_ReturnsBadRequest_WhenLimitIsTooLarge(t *testing.T) {
	// given an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	loginOfficer(t, srv)

	// when I ask for 101 submissions at once
	resp := srv.Get(t, "/api/submissions?limit=101")

	// then I expect a 400
	resp.RequireStatus(t, http.StatusBadRequest)
	if diff := cmp.Diff(map[string]string{"error": "limit: must be between 1 and 100"}, errorBody(t, resp)); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
}

func TestListSubmissions_ReturnsForbidden_WhenUserIsNotOfficer(t *testing.T) {
	// given a guild member who is not an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	loginMember(t, srv)

	// when I list submissions
	resp := srv.Get(t, "/api/submissions")

	// then I expect a 403
	resp.RequireStatus(t, http.StatusForbidden)
}

func TestListSubmissions_ReturnsUnauthorized_WhenNotLoggedIn(t *testing.T) {
	// given nobody is logged in
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I list submissions
	resp := srv.Get(t, "/api/submissions")

	// then I expect a 401
	resp.RequireStatus(t, http.StatusUnauthorized)
}

func TestGetSubmissionFull_ReturnsTheOriginal_WhenOfficerAsks(t *testing.T) {
	// given a member's screenshot
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	original := solidPNG(t, 30, 20, 1)
	id := uploadAsMember(t, srv, "shot.png", original)
	loginOfficer(t, srv)

	// when I open the full image
	resp := srv.Get(t, "/api/submissions/"+id+"/full")

	// then I expect the exact original bytes, served as the sniffed type with private headers
	resp.RequireStatus(t, http.StatusOK)
	want := map[string]string{"Content-Type": "image/png", "X-Content-Type-Options": "nosniff", "Cache-Control": "private"}
	if diff := cmp.Diff(want, headers(resp, "Content-Type", "X-Content-Type-Options", "Cache-Control")); diff != "" {
		t.Errorf("unexpected headers (-want +got):\n%s", diff)
	}
	if !bytes.Equal(original, resp.Body) {
		t.Errorf("full image differs from the original (%d vs %d bytes)", len(resp.Body), len(original))
	}
}

func TestGetSubmissionFull_ReturnsNotFound_WhenSubmissionDoesNotExist(t *testing.T) {
	// given an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	loginOfficer(t, srv)

	// when I open the full image for an id that does not exist
	resp := srv.Get(t, "/api/submissions/"+uuid.NewString()+"/full")

	// then I expect a 404
	resp.RequireStatus(t, http.StatusNotFound)
	if diff := cmp.Diff(map[string]string{"error": "submission not found"}, errorBody(t, resp)); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
}

func TestGetSubmissionFull_ReturnsForbidden_WhenUserIsNotOfficer(t *testing.T) {
	// given my own screenshot
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	id := uploadAsMember(t, srv, "shot.png", solidPNG(t, 8, 8, 1))

	// when I, its submitter but not an officer, open it
	resp := srv.Get(t, "/api/submissions/"+id+"/full")

	// then I expect a 403, since submissions are never publicly readable
	resp.RequireStatus(t, http.StatusForbidden)
}

func TestGetSubmissionThumb_ReturnsAWebPPreview_WhenOfficerAsks(t *testing.T) {
	// given a member's 1600×900 screenshot
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	id := uploadAsMember(t, srv, "wide.png", solidPNG(t, 1600, 900, 1))
	loginOfficer(t, srv)

	// when I load its thumbnail
	resp := srv.Get(t, "/api/submissions/"+id+"/thumb")

	// then I expect a WebP at most 800 pixels on its long edge, with private headers
	resp.RequireStatus(t, http.StatusOK)
	wantHeaders := map[string]string{"Content-Type": "image/webp", "X-Content-Type-Options": "nosniff", "Cache-Control": "private"}
	if diff := cmp.Diff(wantHeaders, headers(resp, "Content-Type", "X-Content-Type-Options", "Cache-Control")); diff != "" {
		t.Errorf("unexpected headers (-want +got):\n%s", diff)
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(resp.Body))
	if err != nil {
		t.Fatalf("decode thumbnail: %v", err)
	}
	got := map[string]any{"format": format, "width": config.Width, "height": config.Height}
	if diff := cmp.Diff(map[string]any{"format": "webp", "width": 800, "height": 450}, got); diff != "" {
		t.Errorf("unexpected thumbnail (-want +got):\n%s", diff)
	}
}

func TestGetSubmissionThumb_ReturnsNotFound_WhenSubmissionDoesNotExist(t *testing.T) {
	// given an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	loginOfficer(t, srv)

	// when I load a thumbnail for an id that does not exist
	resp := srv.Get(t, "/api/submissions/"+uuid.NewString()+"/thumb")

	// then I expect a 404
	resp.RequireStatus(t, http.StatusNotFound)
	if diff := cmp.Diff(map[string]string{"error": "submission not found"}, errorBody(t, resp)); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
}

func TestDownloadSubmission_ReturnsTheOriginalAsAnAttachment_WithItsOriginalFilename(t *testing.T) {
	// given a member's screenshot
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	original := solidPNG(t, 8, 8, 1)
	id := uploadAsMember(t, srv, "Onyxia kill.png", original)
	loginOfficer(t, srv)

	// when I download it
	resp := srv.Get(t, "/api/submissions/"+id+"/download")

	// then I expect the original bytes as an attachment named as uploaded, with private headers
	resp.RequireStatus(t, http.StatusOK)
	want := map[string]string{
		"Content-Disposition":    `attachment; filename="Onyxia kill.png"`,
		"Content-Type":           "image/png",
		"X-Content-Type-Options": "nosniff",
		"Cache-Control":          "private",
	}
	if diff := cmp.Diff(want, headers(resp, "Content-Disposition", "Content-Type", "X-Content-Type-Options", "Cache-Control")); diff != "" {
		t.Errorf("unexpected headers (-want +got):\n%s", diff)
	}
	if !bytes.Equal(original, resp.Body) {
		t.Errorf("download differs from the original")
	}
}

func TestDownloadSubmission_SetsAttachmentDisposition_WhenFilenameContainsHeaderCharacters(t *testing.T) {
	// given a submission whose stored name carries quotes, a line break and a path
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	member := loginMember(t, srv)
	id := insertSubmission(t, db, member.ID, "..\\evil\"\r\nSet-Cookie: x=1.png", []byte{1}, 1, testutil.FixedNow)
	loginOfficer(t, srv)

	// when I download it
	resp := srv.Get(t, "/api/submissions/"+id.String()+"/download")

	// then I expect a single safe attachment header and no injected header
	resp.RequireStatus(t, http.StatusOK)
	if got := resp.Header.Get("Content-Disposition"); got != `attachment; filename="evilSet-Cookie: x=1.png"` {
		t.Errorf("Content-Disposition = %q", got)
	}
	if got := resp.Header.Values("Set-Cookie"); len(got) != 0 {
		t.Errorf("expected no Set-Cookie header, got %q", got)
	}
}

func TestDownloadSubmission_ReturnsNotFound_WhenSubmissionDoesNotExist(t *testing.T) {
	// given an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	loginOfficer(t, srv)

	// when I download an id that does not exist
	resp := srv.Get(t, "/api/submissions/"+uuid.NewString()+"/download")

	// then I expect a 404 and no attachment header
	resp.RequireStatus(t, http.StatusNotFound)
	if diff := cmp.Diff(map[string]string{"error": "submission not found"}, errorBody(t, resp)); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
	if got := resp.Header.Get("Content-Disposition"); got != "" {
		t.Errorf("Content-Disposition = %q, want none", got)
	}
}

func TestDeleteSubmission_RemovesIt_WhenOfficerDeletes(t *testing.T) {
	// given two member submissions
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	member := loginMember(t, srv)
	doomed := insertSubmission(t, db, member.ID, "doomed.png", []byte{1}, 1, testutil.FixedNow)
	insertSubmission(t, db, member.ID, "kept.png", []byte{2}, 1, testutil.FixedNow)
	loginOfficer(t, srv)

	// when I delete one
	resp := srv.Delete(t, "/api/submissions/"+doomed.String())

	// then I expect a 204 and only the other left
	resp.RequireStatus(t, http.StatusNoContent)
	if diff := cmp.Diff([]string{"kept.png"}, storedFilenames(t, db)); diff != "" {
		t.Errorf("unexpected remaining submissions (-want +got):\n%s", diff)
	}
}

func TestDeleteSubmission_ReturnsForbidden_WhenUserIsNotOfficer(t *testing.T) {
	// given my own submission
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	member := loginMember(t, srv)
	id := insertSubmission(t, db, member.ID, "mine.png", []byte{1}, 1, testutil.FixedNow)

	// when I, not an officer, delete it
	resp := srv.Delete(t, "/api/submissions/"+id.String())

	// then I expect a 403 and it still there
	resp.RequireStatus(t, http.StatusForbidden)
	if diff := cmp.Diff([]string{"mine.png"}, storedFilenames(t, db)); diff != "" {
		t.Errorf("unexpected remaining submissions (-want +got):\n%s", diff)
	}
}

func TestDeleteSubmission_ReturnsNotFound_WhenSubmissionDoesNotExist(t *testing.T) {
	// given an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	loginOfficer(t, srv)

	// when I delete an id that does not exist
	resp := srv.Delete(t, "/api/submissions/"+uuid.NewString())

	// then I expect a 404
	resp.RequireStatus(t, http.StatusNotFound)
	if diff := cmp.Diff(map[string]string{"error": "submission not found"}, errorBody(t, resp)); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
}

func TestDeleteSubmission_Succeeds_WhenSubmissionsAreDisabled(t *testing.T) {
	// given a deploy with SUBMISSIONS_ENABLED=false during a spam flood
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db, testutil.WithSubmissionsDisabled())
	member := loginMember(t, srv)
	spam := insertSubmission(t, db, member.ID, "spam.png", []byte{1}, 1, testutil.FixedNow)
	loginOfficer(t, srv)

	// when I delete the spam
	resp := srv.Delete(t, "/api/submissions/"+spam.String())

	// then I expect it deleted, since officers keep cleaning up while uploads are closed
	resp.RequireStatus(t, http.StatusNoContent)
	if diff := cmp.Diff([]string(nil), storedFilenames(t, db)); diff != "" {
		t.Errorf("unexpected remaining submissions (-want +got):\n%s", diff)
	}
}

func TestBulkDeleteSubmissions_RemovesAllSelected_WhenIdsAreValid(t *testing.T) {
	// given three member submissions
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	member := loginMember(t, srv)
	first := insertSubmission(t, db, member.ID, "spam-1.png", []byte{1}, 1, testutil.FixedNow)
	second := insertSubmission(t, db, member.ID, "spam-2.png", []byte{2}, 1, testutil.FixedNow)
	insertSubmission(t, db, member.ID, "keeper.png", []byte{3}, 1, testutil.FixedNow)
	loginOfficer(t, srv)
	unknown := uuid.New()

	// when I bulk-delete the two spam submissions and an id that does not exist
	resp := srv.Post(t, "/api/submissions/bulk-delete", map[string]any{"ids": []string{first.String(), second.String(), unknown.String()}})

	// then I expect the two that existed reported deleted, and only the keeper left
	resp.RequireStatus(t, http.StatusOK)
	var got struct {
		Deleted []string `json:"deleted"`
	}
	resp.DecodeJSON(t, &got)
	if diff := cmp.Diff([]string{first.String(), second.String()}, got.Deleted, cmpopts.SortSlices(func(a, b string) bool { return a < b })); diff != "" {
		t.Errorf("unexpected deleted ids (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"keeper.png"}, storedFilenames(t, db)); diff != "" {
		t.Errorf("unexpected remaining submissions (-want +got):\n%s", diff)
	}
}

func TestBulkDeleteSubmissions_ReturnsBadRequest_WhenAnIdIsMalformed(t *testing.T) {
	// given a member submission
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	member := loginMember(t, srv)
	id := insertSubmission(t, db, member.ID, "kept.png", []byte{1}, 1, testutil.FixedNow)
	loginOfficer(t, srv)

	// when I bulk-delete it along with a malformed id
	resp := srv.Post(t, "/api/submissions/bulk-delete", map[string]any{"ids": []string{id.String(), "not-an-id"}})

	// then I expect a 400 and nothing deleted
	resp.RequireStatus(t, http.StatusBadRequest)
	if diff := cmp.Diff(map[string]string{"error": "ids[1]: must be a submission id"}, errorBody(t, resp)); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"kept.png"}, storedFilenames(t, db)); diff != "" {
		t.Errorf("unexpected remaining submissions (-want +got):\n%s", diff)
	}
}

func TestBulkDeleteSubmissions_ReturnsBadRequest_WhenNoIdsAreGiven(t *testing.T) {
	// given an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	loginOfficer(t, srv)

	// when I bulk-delete an empty list
	resp := srv.Post(t, "/api/submissions/bulk-delete", map[string]any{"ids": []string{}})

	// then I expect a 400
	resp.RequireStatus(t, http.StatusBadRequest)
	if diff := cmp.Diff(map[string]string{"error": "ids: must list between 1 and 100 submissions"}, errorBody(t, resp)); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
}

func TestBulkDeleteSubmissions_ReturnsBadRequest_WhenMoreThanOneHundredIdsGiven(t *testing.T) {
	// given a member submission and an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	member := loginMember(t, srv)
	kept := insertSubmission(t, db, member.ID, "kept.png", []byte{1}, 1, testutil.FixedNow)
	loginOfficer(t, srv)
	ids := []string{kept.String()}
	for range 100 {
		ids = append(ids, uuid.NewString())
	}

	// when I bulk-delete 101 ids, the real one among them
	resp := srv.Post(t, "/api/submissions/bulk-delete", map[string]any{"ids": ids})

	// then I expect a 400 and nothing deleted
	resp.RequireStatus(t, http.StatusBadRequest)
	if diff := cmp.Diff(map[string]string{"error": "ids: must list between 1 and 100 submissions"}, errorBody(t, resp)); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"kept.png"}, storedFilenames(t, db)); diff != "" {
		t.Errorf("unexpected remaining submissions (-want +got):\n%s", diff)
	}
}

func TestBulkDeleteSubmissions_ReturnsForbidden_WhenUserIsNotOfficer(t *testing.T) {
	// given my own submission
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	member := loginMember(t, srv)
	id := insertSubmission(t, db, member.ID, "mine.png", []byte{1}, 1, testutil.FixedNow)

	// when I, not an officer, bulk-delete it
	resp := srv.Post(t, "/api/submissions/bulk-delete", map[string]any{"ids": []string{id.String()}})

	// then I expect a 403 and it still there
	resp.RequireStatus(t, http.StatusForbidden)
	if diff := cmp.Diff([]string{"mine.png"}, storedFilenames(t, db)); diff != "" {
		t.Errorf("unexpected remaining submissions (-want +got):\n%s", diff)
	}
}
