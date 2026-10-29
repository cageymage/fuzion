package submissions_test

import (
	"bytes"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/cageymage/fuzion/backend/internal/auth"
	"github.com/cageymage/fuzion/backend/internal/submissions"
	"github.com/cageymage/fuzion/backend/internal/testutil"
)

func errorBody(t *testing.T, resp testutil.Response) map[string]string {
	t.Helper()
	var body map[string]string
	resp.DecodeJSON(t, &body)
	return body
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
	firstID, secondID := got.Results[0].Submission.ID, got.Results[1].Submission.ID
	want := uploadResponseJSON{Results: []fileResultJSON{
		{Filename: "onyxia.png", Submission: &submissionJSON{
			ID: firstID, Submitter: submitterJSON{ID: member.ID.String(), Username: "thundermane"},
			Description: "Onyxia down!", OriginalFilename: "onyxia.png", ContentType: "image/png",
			Width: 64, Height: 32, ByteSize: len(first), CreatedAt: testutil.FixedNow,
			ThumbURL: "/api/submissions/" + firstID + "/thumb", FullURL: "/api/submissions/" + firstID + "/full",
			DownloadURL: "/api/submissions/" + firstID + "/download",
		}},
		{Filename: "raid.png", Submission: &submissionJSON{
			ID: secondID, Submitter: submitterJSON{ID: member.ID.String(), Username: "thundermane"},
			Description: "", OriginalFilename: "raid.png", ContentType: "image/png",
			Width: 20, Height: 40, ByteSize: len(second), CreatedAt: testutil.FixedNow,
			ThumbURL: "/api/submissions/" + secondID + "/thumb", FullURL: "/api/submissions/" + secondID + "/full",
			DownloadURL: "/api/submissions/" + secondID + "/download",
		}},
	}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected response (-want +got):\n%s", diff)
	}
	wantRows := []storedRow{
		{UserID: member.ID.String(), OriginalFilename: "onyxia.png", ContentType: "image/png", Width: 64, Height: 32, ByteSize: len(first), Description: "Onyxia down!", AttestationAcceptedAt: testutil.FixedNow, AttestationVersion: "2026-10-26", CreatedAt: testutil.FixedNow},
		{UserID: member.ID.String(), OriginalFilename: "raid.png", ContentType: "image/png", Width: 20, Height: 40, ByteSize: len(second), Description: "", AttestationAcceptedAt: testutil.FixedNow, AttestationVersion: "2026-10-26", CreatedAt: testutil.FixedNow},
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
	if diff := cmp.Diff([]storedRow(nil), storedRows(t, db)); diff != "" {
		t.Errorf("expected no submissions (-want +got):\n%s", diff)
	}
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
	if diff := cmp.Diff(map[string]string{"error": "guild discord server membership required"}, errorBody(t, resp)); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]storedRow(nil), storedRows(t, db)); diff != "" {
		t.Errorf("expected no submissions (-want +got):\n%s", diff)
	}
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
	if diff := cmp.Diff(map[string]string{"error": "attestationAccepted: must be accepted"}, errorBody(t, resp)); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]storedRow(nil), storedRows(t, db)); diff != "" {
		t.Errorf("expected no submissions (-want +got):\n%s", diff)
	}
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
	want := map[string]string{"error": "attestationVersion: is out of date; reload the page and accept the current wording"}
	if diff := cmp.Diff(want, errorBody(t, resp)); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]storedRow(nil), storedRows(t, db)); diff != "" {
		t.Errorf("expected no submissions (-want +got):\n%s", diff)
	}
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
	if diff := cmp.Diff(map[string]string{"error": "files: at most 10 images per submission"}, errorBody(t, resp)); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]storedRow(nil), storedRows(t, db)); diff != "" {
		t.Errorf("expected no submissions (-want +got):\n%s", diff)
	}
}

func TestUploadSubmission_ReturnsRequestEntityTooLarge_WhenBodyExceedsFiftyMiB(t *testing.T) {
	// given a guild member with three 18 MiB files, each under the per-file cap but 54 MiB together
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	loginMember(t, srv)
	big := func(seed uint8) []byte { return append(solidPNG(t, 8, 8, seed), make([]byte, 18<<20)...) }

	// when I submit all three at once
	resp := srv.PostMultipart(t, uploadPath, acceptedMetadata(t), filePart("a.png", big(1)), filePart("b.png", big(2)), filePart("c.png", big(3)))

	// then I expect a 413 and nothing stored
	resp.RequireStatus(t, http.StatusRequestEntityTooLarge)
	if diff := cmp.Diff(map[string]string{"error": "a submission must be at most 50 MiB in total"}, errorBody(t, resp)); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]storedRow(nil), storedRows(t, db)); diff != "" {
		t.Errorf("expected no submissions (-want +got):\n%s", diff)
	}
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
	want := []storedRow{
		{UserID: member.ID.String(), OriginalFilename: "shot.gif", ContentType: "image/png", Width: 8, Height: 8, ByteSize: len(data), AttestationAcceptedAt: testutil.FixedNow, AttestationVersion: "2026-10-26", CreatedAt: testutil.FixedNow},
	}
	if diff := cmp.Diff(want, storedRows(t, db)); diff != "" {
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
	if diff := cmp.Diff([]storedRow(nil), storedRows(t, db)); diff != "" {
		t.Errorf("expected no submissions (-want +got):\n%s", diff)
	}
}

func TestUploadSubmission_ReturnsBadRequest_WhenImageExceedsPixelLimit(t *testing.T) {
	// given a guild member with a PNG whose header claims 6000×5000 pixels (30 megapixels)
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	loginMember(t, srv)

	// when I submit it
	resp := srv.PostMultipart(t, uploadPath, acceptedMetadata(t), filePart("huge.png", pngHeaderOnly(6000, 5000)))

	// then I expect a 400 before any full decode, and nothing stored
	resp.RequireStatus(t, http.StatusBadRequest)
	var got uploadResponseJSON
	resp.DecodeJSON(t, &got)
	want := uploadResponseJSON{Results: []fileResultJSON{{Filename: "huge.png", Error: "is 6000×5000 pixels; images must be at most 25 megapixels"}}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected response (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]storedRow(nil), storedRows(t, db)); diff != "" {
		t.Errorf("expected no submissions (-want +got):\n%s", diff)
	}
}

func TestUploadSubmission_ReturnsBadRequest_WhenGifScreenExceedsPixelLimit(t *testing.T) {
	// given a guild member with a GIF whose logical screen is 6000×5000, the bound on every frame in it
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	loginMember(t, srv)

	// when I submit it
	resp := srv.PostMultipart(t, uploadPath, acceptedMetadata(t), filePart("huge.gif", gifHeaderOnly(6000, 5000)))

	// then I expect a 400 before any frame is decoded, and nothing stored
	resp.RequireStatus(t, http.StatusBadRequest)
	var got uploadResponseJSON
	resp.DecodeJSON(t, &got)
	want := uploadResponseJSON{Results: []fileResultJSON{{Filename: "huge.gif", Error: "is 6000×5000 pixels; images must be at most 25 megapixels"}}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected response (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]storedRow(nil), storedRows(t, db)); diff != "" {
		t.Errorf("expected no submissions (-want +got):\n%s", diff)
	}
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
	id := got.Results[0].Submission.ID
	want := uploadResponseJSON{Results: []fileResultJSON{
		{Filename: "small.png", Submission: &submissionJSON{
			ID: id, Submitter: submitterJSON{ID: member.ID.String(), Username: "thundermane"},
			OriginalFilename: "small.png", ContentType: "image/png", Width: 8, Height: 8, ByteSize: len(small),
			CreatedAt: testutil.FixedNow, ThumbURL: "/api/submissions/" + id + "/thumb",
			FullURL: "/api/submissions/" + id + "/full", DownloadURL: "/api/submissions/" + id + "/download",
		}},
		{Filename: "big.png", Error: "must be at most 20 MiB"},
	}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected response (-want +got):\n%s", diff)
	}
	wantRows := []storedRow{
		{UserID: member.ID.String(), OriginalFilename: "small.png", ContentType: "image/png", Width: 8, Height: 8, ByteSize: len(small), AttestationAcceptedAt: testutil.FixedNow, AttestationVersion: "2026-10-26", CreatedAt: testutil.FixedNow},
	}
	if diff := cmp.Diff(wantRows, storedRows(t, db)); diff != "" {
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
	if diff := cmp.Diff(map[string]string{"error": "descriptions[0]: must be at most 500 characters"}, errorBody(t, resp)); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]storedRow(nil), storedRows(t, db)); diff != "" {
		t.Errorf("expected no submissions (-want +got):\n%s", diff)
	}
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
	wantRows := []storedRow{
		{UserID: member.ID.String(), OriginalFilename: "shot.png", ContentType: "image/png", Width: 8, Height: 8, ByteSize: len(shot), AttestationAcceptedAt: testutil.FixedNow, AttestationVersion: "2026-10-26", CreatedAt: testutil.FixedNow},
	}
	if diff := cmp.Diff(wantRows, storedRows(t, db)); diff != "" {
		t.Errorf("unexpected stored submissions (-want +got):\n%s", diff)
	}
}

func TestUploadSubmission_StoresOneCopy_WhenTheSameFileAppearsTwiceInOneBatch(t *testing.T) {
	// given a guild member who picked the same screenshot twice
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	member := loginMember(t, srv)
	shot := solidPNG(t, 8, 8, 1)

	// when I submit both copies in one request
	resp := srv.PostMultipart(t, uploadPath, acceptedMetadata(t), filePart("shot.png", shot), filePart("copy.png", shot))

	// then I expect the first stored and the second reported as a duplicate, without failing the batch
	resp.RequireStatus(t, http.StatusCreated)
	var got uploadResponseJSON
	resp.DecodeJSON(t, &got)
	if len(got.Results) != 2 || got.Results[0].Submission == nil {
		t.Fatalf("expected the first copy stored, got %+v", got)
	}
	if diff := cmp.Diff(fileResultJSON{Filename: "copy.png", Error: "you have already submitted this image"}, got.Results[1]); diff != "" {
		t.Errorf("unexpected result for the second copy (-want +got):\n%s", diff)
	}
	wantRows := []storedRow{
		{UserID: member.ID.String(), OriginalFilename: "shot.png", ContentType: "image/png", Width: 8, Height: 8, ByteSize: len(shot), AttestationAcceptedAt: testutil.FixedNow, AttestationVersion: "2026-10-26", CreatedAt: testutil.FixedNow},
	}
	if diff := cmp.Diff(wantRows, storedRows(t, db)); diff != "" {
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
	want := []storedRow{
		{UserID: member.ID.String(), OriginalFilename: "mine.png", ContentType: "image/png", Width: 8, Height: 8, ByteSize: len(shot), AttestationAcceptedAt: testutil.FixedNow, AttestationVersion: "2026-10-26", CreatedAt: testutil.FixedNow},
		{UserID: other.ID.String(), OriginalFilename: "theirs.png", ContentType: "image/png", Width: 8, Height: 8, ByteSize: len(shot), AttestationAcceptedAt: testutil.FixedNow, AttestationVersion: "2026-10-26", CreatedAt: testutil.FixedNow},
	}
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
	if diff := cmp.Diff(map[string]string{"error": "you can submit at most 20 images per 24 hours; 0 left"}, errorBody(t, resp)); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
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
	if diff := cmp.Diff(map[string]string{"error": "you can submit at most 100 MiB per 24 hours"}, errorBody(t, resp)); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(before, storedRows(t, db)); diff != "" {
		t.Errorf("submissions changed (-want +got):\n%s", diff)
	}
}

func TestUploadSubmission_KeepsTheDailyQuota_WhenOneMemberSendsUploadsInParallel(t *testing.T) {
	// given a guild member with room for two more images today
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	member := loginMember(t, srv)
	for i := range submissions.DailyFileQuota - 2 {
		insertSubmission(t, db, member.ID, "earlier.png", []byte{byte(i)}, 1, testutil.FixedNow.Add(-time.Hour))
	}

	// when I send two uploads of two images each at the same time
	statuses := make([]int, 2)
	var wg sync.WaitGroup
	for i := range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp := srv.PostMultipart(t, uploadPath, acceptedMetadata(t),
				filePart("parallel.png", solidPNG(t, 8, 8, uint8(10+2*i))),
				filePart("parallel.png", solidPNG(t, 8, 8, uint8(11+2*i))))
			statuses[i] = resp.StatusCode
		}()
	}
	wg.Wait()

	// then I expect one to be stored and the other refused, never both past the quota
	slices.Sort(statuses)
	if diff := cmp.Diff([]int{http.StatusCreated, http.StatusTooManyRequests}, statuses); diff != "" {
		t.Errorf("unexpected statuses (-want +got):\n%s", diff)
	}
	var stored int
	if err := db.Get(&stored, `SELECT count(*) FROM submissions WHERE user_id = $1`, member.ID); err != nil {
		t.Fatalf("count submissions: %v", err)
	}
	if stored != submissions.DailyFileQuota {
		t.Errorf("stored %d submissions, want exactly the quota of %d", stored, submissions.DailyFileQuota)
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
	srv.SetCookie(t, &http.Cookie{Name: auth.SessionCookieName, Value: "session-for-" + member.DiscordID, Path: "/"})

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
	if diff := cmp.Diff(map[string]string{"error": "submissions are temporarily closed"}, errorBody(t, resp)); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]storedRow(nil), storedRows(t, db)); diff != "" {
		t.Errorf("expected no submissions (-want +got):\n%s", diff)
	}
}

func TestUploadSubmission_ReturnsServiceUnavailable_WhenNoUploadSlotIsFree(t *testing.T) {
	// given a server whose upload slots are all taken
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db, testutil.WithSubmissionUploadSlots(0))
	loginMember(t, srv)

	// when I submit a screenshot
	resp := srv.PostMultipart(t, uploadPath, acceptedMetadata(t), filePart("shot.png", solidPNG(t, 8, 8, 1)))

	// then I expect a 503 asking me to retry shortly, before the body is read, and nothing stored
	resp.RequireStatus(t, http.StatusServiceUnavailable)
	if diff := cmp.Diff(map[string]string{"error": "too many uploads are in progress; try again in a few seconds"}, errorBody(t, resp)); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
	if got := resp.Header.Get("Retry-After"); got != "10" {
		t.Errorf("Retry-After = %q, want 10", got)
	}
	if diff := cmp.Diff([]storedRow(nil), storedRows(t, db)); diff != "" {
		t.Errorf("expected no submissions (-want +got):\n%s", diff)
	}
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
	if diff := cmp.Diff(map[string]string{"error": "submissions are temporarily closed: storage is full"}, errorBody(t, resp)); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
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
	if diff := cmp.Diff(map[string]any{"open": true, "attestationVersion": "2026-10-26"}, got); diff != "" {
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
	if diff := cmp.Diff(map[string]any{"open": false, "attestationVersion": "2026-10-26"}, got); diff != "" {
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
	if diff := cmp.Diff(map[string]any{"open": false, "attestationVersion": "2026-10-26"}, got); diff != "" {
		t.Errorf("unexpected status (-want +got):\n%s", diff)
	}
}

func TestSubmissionStatus_ReportsOpenAgain_WhenAnOfficerDeletionFreesStorage(t *testing.T) {
	// given a full 1 KiB ceiling the status has already reported, so the total is cached
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db, testutil.WithSubmissionStorageLimit(1024))
	member := loginMember(t, srv)
	big := insertSubmission(t, db, member.ID, "big.png", []byte{1}, 1024, testutil.FixedNow)
	srv.Get(t, "/api/submissions/status").RequireStatus(t, http.StatusOK)
	loginOfficer(t, srv)
	srv.Delete(t, "/api/submissions/"+big.String()).RequireStatus(t, http.StatusNoContent)

	// when I ask whether submissions are open, within the cache window
	resp := srv.Get(t, "/api/submissions/status")

	// then I expect them open, since a delete clears the cached total
	resp.RequireStatus(t, http.StatusOK)
	var got map[string]any
	resp.DecodeJSON(t, &got)
	if diff := cmp.Diff(map[string]any{"open": true, "attestationVersion": "2026-10-26"}, got); diff != "" {
		t.Errorf("unexpected status (-want +got):\n%s", diff)
	}
}
