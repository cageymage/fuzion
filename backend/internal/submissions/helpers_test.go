package submissions_test

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/cageymage/fuzion/backend/internal/auth"
	"github.com/cageymage/fuzion/backend/internal/submissions"
	"github.com/cageymage/fuzion/backend/internal/testutil"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.Run(m))
}

const uploadPath = "/api/submissions"

// solidPNG returns a small valid PNG; seed varies the pixels so each seed hashes differently.
func solidPNG(t *testing.T, width, height int, seed uint8) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: seed, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode test png: %v", err)
	}
	return buf.Bytes()
}

// pngHeaderOnly is a PNG whose header claims width×height but carries no pixel data,
// so a pixel limit can be tested without allocating the image.
func pngHeaderOnly(width, height uint32) []byte {
	chunk := func(kind string, data []byte) []byte {
		var b bytes.Buffer
		_ = binary.Write(&b, binary.BigEndian, uint32(len(data)))
		b.WriteString(kind)
		b.Write(data)
		_ = binary.Write(&b, binary.BigEndian, crc32.ChecksumIEEE(append([]byte(kind), data...)))
		return b.Bytes()
	}
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:4], width)
	binary.BigEndian.PutUint32(ihdr[4:8], height)
	ihdr[8], ihdr[9] = 8, 6 // 8-bit RGBA
	var b bytes.Buffer
	b.WriteString("\x89PNG\r\n\x1a\n")
	b.Write(chunk("IHDR", ihdr))
	b.Write(chunk("IEND", nil))
	return b.Bytes()
}

// gifHeaderOnly is a GIF whose logical screen is width×height and that has no frames,
// so a pixel limit can be tested without allocating the image.
func gifHeaderOnly(width, height uint16) []byte {
	var b bytes.Buffer
	b.WriteString("GIF89a")
	_ = binary.Write(&b, binary.LittleEndian, width)
	_ = binary.Write(&b, binary.LittleEndian, height)
	b.Write([]byte{0, 0, 0, 0x3b}) // no color table, background, aspect, trailer
	return b.Bytes()
}

func metadataPart(t *testing.T, accepted bool, version string, descriptions ...string) testutil.MultipartPart {
	t.Helper()
	data, err := json.Marshal(map[string]any{
		"attestationAccepted": accepted,
		"attestationVersion":  version,
		"descriptions":        descriptions,
	})
	if err != nil {
		t.Fatalf("encode metadata: %v", err)
	}
	return testutil.MultipartPart{Field: "metadata", ContentType: "application/json", Data: data}
}

func acceptedMetadata(t *testing.T, descriptions ...string) testutil.MultipartPart {
	t.Helper()
	return metadataPart(t, true, submissions.AttestationVersion, descriptions...)
}

func filePart(filename string, data []byte) testutil.MultipartPart {
	return testutil.MultipartPart{Field: "files", Filename: filename, ContentType: "image/png", Data: data}
}

func loginMember(t *testing.T, srv *testutil.Server) auth.User {
	t.Helper()
	return srv.LoginAs(t, "80351110224678912", "thundermane", testutil.InDiscordServer())
}

func loginOfficer(t *testing.T, srv *testutil.Server) auth.User {
	t.Helper()
	return srv.LoginAs(t, "11111111111111111", "officerone", testutil.AsOfficer(), testutil.InDiscordServer())
}

type submissionJSON struct {
	ID               string        `json:"id"`
	Submitter        submitterJSON `json:"submitter"`
	Description      string        `json:"description"`
	OriginalFilename string        `json:"originalFilename"`
	ContentType      string        `json:"contentType"`
	Width            int           `json:"width"`
	Height           int           `json:"height"`
	ByteSize         int           `json:"byteSize"`
	CreatedAt        time.Time     `json:"createdAt"`
	ThumbURL         string        `json:"thumbUrl"`
	FullURL          string        `json:"fullUrl"`
	DownloadURL      string        `json:"downloadUrl"`
}

type submitterJSON struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type fileResultJSON struct {
	Filename   string          `json:"filename"`
	Submission *submissionJSON `json:"submission"`
	Error      string          `json:"error"`
}

type uploadResponseJSON struct {
	Results []fileResultJSON `json:"results"`
}

type storedRow struct {
	UserID                string    `db:"user_id"`
	OriginalFilename      string    `db:"original_filename"`
	ContentType           string    `db:"content_type"`
	Width                 int       `db:"width"`
	Height                int       `db:"height"`
	ByteSize              int       `db:"byte_size"`
	Description           string    `db:"description"`
	AttestationAcceptedAt time.Time `db:"attestation_accepted_at"`
	AttestationVersion    string    `db:"attestation_version"`
	CreatedAt             time.Time `db:"created_at"`
}

func storedRows(t *testing.T, db *sqlx.DB) []storedRow {
	t.Helper()
	var rows []storedRow
	err := db.Select(&rows, `
		SELECT user_id, original_filename, content_type, width, height, byte_size, description,
		       attestation_accepted_at, attestation_version, created_at
		FROM submissions ORDER BY original_filename`)
	if err != nil {
		t.Fatalf("select submissions: %v", err)
	}
	for i := range rows {
		rows[i].AttestationAcceptedAt = rows[i].AttestationAcceptedAt.UTC()
		rows[i].CreatedAt = rows[i].CreatedAt.UTC()
	}
	return rows
}

// insertSubmission stores a row directly, for tests that need a given age, size or
// file name rather than the upload path.
func insertSubmission(t *testing.T, db *sqlx.DB, userID uuid.UUID, filename string, original []byte, byteSize int, createdAt time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	db.MustExec(`
		INSERT INTO submissions (
			id, user_id, original_bytes, byte_size, content_type, original_filename, thumb_bytes,
			width, height, description, attestation_accepted_at, attestation_version, content_hash, created_at
		)
		VALUES ($1, $2, $3, $4, 'image/png', $5, '\x00', 4, 4, '', $6, $7, $8, $6)`,
		id, userID, original, byteSize, filename, createdAt, submissions.AttestationVersion, id.String())
	return id
}
