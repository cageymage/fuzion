package images_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/image/webp"

	"github.com/cageymage/fuzion/backend/internal/testutil"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.Run(m))
}

type uploadedImageJSON struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

func solidPNG(t *testing.T, width, height int) []byte {
	t.Helper()

	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode test png: %v", err)
	}
	return buf.Bytes()
}

func uploadAsOfficer(t *testing.T, srv *testutil.Server, data []byte) uploadedImageJSON {
	t.Helper()

	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	resp := srv.PostFile(t, "/api/images", "file", "kill.png", "image/png", data)
	resp.RequireStatus(t, 201)
	var uploaded uploadedImageJSON
	resp.DecodeJSON(t, &uploaded)
	return uploaded
}

func TestUploadImage_ReturnsCreatedWithImageUrl_WhenOfficerUploadsAPng(t *testing.T) {
	// given a signed-in officer and a png
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	data := solidPNG(t, 64, 32)

	// when I upload the png
	resp := srv.PostFile(t, "/api/images", "file", "kill.png", "image/png", data)

	// then I expect a 201 with the id and the url the markdown should hold
	resp.RequireStatus(t, 201)
	var uploaded uploadedImageJSON
	resp.DecodeJSON(t, &uploaded)
	want := uploadedImageJSON{ID: uploaded.ID, URL: "/api/images/" + uploaded.ID}
	if uploaded.ID == "" {
		t.Fatalf("expected an image id, got %+v", uploaded)
	}
	if diff := cmp.Diff(want, uploaded); diff != "" {
		t.Errorf("response mismatch (-want +got):\n%s", diff)
	}
}

func TestUploadImage_ReturnsUnauthorized_WhenUserIsAnonymous(t *testing.T) {
	// given nobody is signed in
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I upload a png
	resp := srv.PostFile(t, "/api/images", "file", "kill.png", "image/png", solidPNG(t, 8, 8))

	// then I expect a 401
	resp.RequireStatus(t, 401)
}

func TestUploadImage_ReturnsForbidden_WhenUserIsNotAnOfficer(t *testing.T) {
	// given a signed-in member who is not an officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "member-1", "Member")

	// when I upload a png
	resp := srv.PostFile(t, "/api/images", "file", "kill.png", "image/png", solidPNG(t, 8, 8))

	// then I expect a 403
	resp.RequireStatus(t, 403)
}

func TestUploadImage_ReturnsBadRequest_WhenFileIsNotAnImage(t *testing.T) {
	// given a signed-in officer and a text file that claims to be a png
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I upload the text file
	resp := srv.PostFile(t, "/api/images", "file", "notes.png", "image/png", []byte("this is not an image"))

	// then I expect a 400 explaining the file is not a supported image
	resp.RequireStatus(t, 400)
	var body map[string]string
	resp.DecodeJSON(t, &body)
	want := map[string]string{"error": "file: must be a PNG, JPEG, WebP or GIF image"}
	if diff := cmp.Diff(want, body); diff != "" {
		t.Errorf("response mismatch (-want +got):\n%s", diff)
	}
}

func TestUploadImage_ReturnsBadRequest_WhenDeclaredTypeDoesNotMatchTheBytes(t *testing.T) {
	// given a signed-in officer and png bytes declared as jpeg
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I upload the mislabeled file
	resp := srv.PostFile(t, "/api/images", "file", "kill.jpg", "image/jpeg", solidPNG(t, 8, 8))

	// then I expect a 400
	resp.RequireStatus(t, 400)
}

func TestUploadImage_ReturnsBadRequest_WhenFileFieldIsMissing(t *testing.T) {
	// given a signed-in officer and a form whose file is under the wrong field
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())

	// when I upload it
	resp := srv.PostFile(t, "/api/images", "attachment", "kill.png", "image/png", solidPNG(t, 8, 8))

	// then I expect a 400
	resp.RequireStatus(t, 400)
}

func TestUploadImage_ReturnsRequestEntityTooLarge_WhenFileExceedsTheLimit(t *testing.T) {
	// given a signed-in officer and a file larger than 10 MiB
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "officer-1", "Officer", testutil.AsOfficer())
	oversized := make([]byte, 10<<20+1)

	// when I upload the oversized file
	resp := srv.PostFile(t, "/api/images", "file", "huge.png", "image/png", oversized)

	// then I expect a 413
	resp.RequireStatus(t, 413)
}

func TestGetImage_ReturnsWebpWithCacheHeaders_WhenImageExists(t *testing.T) {
	// given an uploaded png
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	uploaded := uploadAsOfficer(t, srv, solidPNG(t, 64, 32))

	// when I fetch the image anonymously
	resp := srv.Get(t, uploaded.URL)

	// then I expect a 200 webp with the original dimensions and an immutable cache policy
	resp.RequireStatus(t, 200)
	if got := resp.Header.Get("Content-Type"); got != "image/webp" {
		t.Errorf("Content-Type: want image/webp, got %q", got)
	}
	if got := resp.Header.Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Errorf("Cache-Control: want public, max-age=31536000, immutable, got %q", got)
	}
	config, err := webp.DecodeConfig(bytes.NewReader(resp.Body))
	if err != nil {
		t.Fatalf("response is not a webp: %v", err)
	}
	if config.Width != 64 || config.Height != 32 {
		t.Errorf("dimensions: want 64x32, got %dx%d", config.Width, config.Height)
	}
}

func TestGetImage_ReturnsDownscaledWebp_WhenImageIsLargerThanTheFullSizeLimit(t *testing.T) {
	// given an uploaded png wider than 2560 px
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	uploaded := uploadAsOfficer(t, srv, solidPNG(t, 3000, 1000))

	// when I fetch the full-size image
	resp := srv.Get(t, uploaded.URL)

	// then I expect a webp whose long edge is 2560 px with the aspect ratio kept
	resp.RequireStatus(t, 200)
	config, err := webp.DecodeConfig(bytes.NewReader(resp.Body))
	if err != nil {
		t.Fatalf("response is not a webp: %v", err)
	}
	if config.Width != 2560 || config.Height != 853 {
		t.Errorf("dimensions: want 2560x853, got %dx%d", config.Width, config.Height)
	}
}

func TestGetImageThumb_ReturnsDownscaledWebp_WhenImageIsLargerThanThumbWidth(t *testing.T) {
	// given an uploaded png wider than 800 px
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	uploaded := uploadAsOfficer(t, srv, solidPNG(t, 1600, 900))

	// when I fetch the thumbnail
	resp := srv.Get(t, uploaded.URL+"/thumb")

	// then I expect a webp whose long edge is 800 px with the cache policy
	resp.RequireStatus(t, 200)
	if got := resp.Header.Get("Content-Type"); got != "image/webp" {
		t.Errorf("Content-Type: want image/webp, got %q", got)
	}
	if got := resp.Header.Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Errorf("Cache-Control: want public, max-age=31536000, immutable, got %q", got)
	}
	config, err := webp.DecodeConfig(bytes.NewReader(resp.Body))
	if err != nil {
		t.Fatalf("response is not a webp: %v", err)
	}
	if config.Width != 800 || config.Height != 450 {
		t.Errorf("dimensions: want 800x450, got %dx%d", config.Width, config.Height)
	}
}

func TestGetImage_ReturnsNotFound_WhenIdIsUnknown(t *testing.T) {
	// given no uploaded images
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I fetch an id that does not exist
	resp := srv.Get(t, "/api/images/11111111-1111-1111-1111-111111111111")

	// then I expect a 404
	resp.RequireStatus(t, 404)
}

func TestGetImage_ReturnsNotFound_WhenIdIsNotAUuid(t *testing.T) {
	// given no uploaded images
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I fetch an id that is not a uuid
	resp := srv.Get(t, "/api/images/not-a-uuid")

	// then I expect a 404
	resp.RequireStatus(t, 404)
}
