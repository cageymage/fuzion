package submissions

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	_ "golang.org/x/image/webp"

	"github.com/cageymage/fuzion/backend/internal/clock"
	"github.com/cageymage/fuzion/backend/internal/images"
)

// AttestationVersion names the wording of the license grant and attestation the
// submission form shows. Change it whenever that wording changes, so each stored
// submission records which wording its submitter accepted.
const AttestationVersion = "2026-10-26"

const (
	MaxFileBytes     = 20 << 20
	MaxBatchFiles    = 10
	MaxBatchBytes    = 50 << 20
	MaxDescription   = 500
	DailyFileQuota   = 20
	DailyByteQuota   = 100 << 20
	quotaWindow      = 24 * time.Hour
	maxFilenameRunes = 200

	// Rejects decompression bombs before a full decode: a decoded image costs about
	// 4 bytes per pixel, so this is roughly 200 MB. Files are decoded one at a time.
	MaxPixels = 50_000_000
)

var allowedTypes = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/webp": true,
	"image/gif":  true,
}

var (
	ErrClosed      = errors.New("submissions are temporarily closed")
	ErrStorageFull = errors.New("submissions are temporarily closed: storage is full")
)

type ValidationError struct {
	Field   string
	Problem string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Problem
}

type QuotaError struct {
	Problem string
}

func (e *QuotaError) Error() string {
	return e.Problem
}

type UploadFile struct {
	Filename string
	Data     []byte
	// TooLarge marks a file the handler stopped reading at MaxFileBytes.
	TooLarge bool
}

type UploadRequest struct {
	AttestationAccepted bool
	AttestationVersion  string
	Descriptions        []string
	Files               []UploadFile
}

// FileResult is the outcome for one file. Exactly one of Submission and Problem is set.
type FileResult struct {
	Filename   string
	Submission *Summary
	Problem    string
	Duplicate  bool
}

type Service struct {
	repo         *Repo
	clock        clock.Clock
	enabled      bool
	storageLimit int64
}

// enabled is the SUBMISSIONS_ENABLED kill switch: off refuses uploads while officers
// keep reading and deleting. storageLimit caps the bytes all submissions may hold.
func NewService(repo *Repo, clock clock.Clock, enabled bool, storageLimit int64) *Service {
	return &Service{repo: repo, clock: clock, enabled: enabled, storageLimit: storageLimit}
}

// Open reports whether uploads are accepted right now.
func (s *Service) Open(ctx context.Context) (bool, error) {
	if !s.enabled {
		return false, nil
	}
	total, err := s.repo.totalBytes(ctx)
	if err != nil {
		return false, fmt.Errorf("check submission storage: %w", err)
	}
	return total < s.storageLimit, nil
}

// Enabled is the kill switch alone, which the handler checks before reading a body.
func (s *Service) Enabled() bool {
	return s.enabled
}

func (s *Service) Upload(ctx context.Context, userID uuid.UUID, req UploadRequest) ([]FileResult, error) {
	if !s.enabled {
		return nil, ErrClosed
	}
	if err := validateRequest(req); err != nil {
		return nil, err
	}

	var batchBytes int64
	for _, f := range req.Files {
		if !f.TooLarge {
			batchBytes += int64(len(f.Data))
		}
	}

	total, err := s.repo.totalBytes(ctx)
	if err != nil {
		return nil, fmt.Errorf("check submission storage: %w", err)
	}
	if total+batchBytes > s.storageLimit {
		return nil, ErrStorageFull
	}

	now := s.clock.Now().UTC()
	files, usedBytes, err := s.repo.usageSince(ctx, userID, now.Add(-quotaWindow))
	if err != nil {
		return nil, err
	}
	if files+len(req.Files) > DailyFileQuota {
		return nil, &QuotaError{Problem: fmt.Sprintf("you can submit at most %d images per 24 hours; %d left", DailyFileQuota, max(0, DailyFileQuota-files))}
	}
	if usedBytes+batchBytes > DailyByteQuota {
		return nil, &QuotaError{Problem: fmt.Sprintf("you can submit at most %d MiB per 24 hours", DailyByteQuota>>20)}
	}

	results := make([]FileResult, len(req.Files))
	for i, f := range req.Files {
		description := ""
		if i < len(req.Descriptions) {
			description = strings.TrimSpace(req.Descriptions[i])
		}
		results[i] = s.store(ctx, userID, f, description, req.AttestationVersion, now)
	}
	return results, nil
}

func validateRequest(req UploadRequest) error {
	if !req.AttestationAccepted {
		return &ValidationError{Field: "attestationAccepted", Problem: "must be accepted"}
	}
	if req.AttestationVersion != AttestationVersion {
		return &ValidationError{Field: "attestationVersion", Problem: "is out of date; reload the page and accept the current wording"}
	}
	if len(req.Files) == 0 {
		return &ValidationError{Field: "files", Problem: "at least one image is required"}
	}
	if len(req.Files) > MaxBatchFiles {
		return &ValidationError{Field: "files", Problem: fmt.Sprintf("at most %d images per submission", MaxBatchFiles)}
	}
	if len(req.Descriptions) > len(req.Files) {
		return &ValidationError{Field: "descriptions", Problem: "has more entries than there are images"}
	}
	for i, d := range req.Descriptions {
		if utf8.RuneCountInString(strings.TrimSpace(d)) > MaxDescription {
			return &ValidationError{Field: fmt.Sprintf("descriptions[%d]", i), Problem: fmt.Sprintf("must be at most %d characters", MaxDescription)}
		}
	}
	return nil
}

func (s *Service) store(ctx context.Context, userID uuid.UUID, f UploadFile, description, version string, now time.Time) FileResult {
	result := FileResult{Filename: SafeFilename(f.Filename)}
	if f.TooLarge {
		result.Problem = fmt.Sprintf("must be at most %d MiB", MaxFileBytes>>20)
		return result
	}

	// The declared type is the browser's guess from the file name, so only the
	// bytes decide what the file is.
	contentType := http.DetectContentType(f.Data)
	if !allowedTypes[contentType] {
		result.Problem = "must be a PNG, JPEG, WebP or GIF image"
		return result
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(f.Data))
	if err != nil || config.Width < 1 || config.Height < 1 {
		result.Problem = "must be a PNG, JPEG, WebP or GIF image"
		return result
	}
	if config.Width*config.Height > MaxPixels {
		result.Problem = fmt.Sprintf("is %d×%d pixels; images must be at most %d megapixels", config.Width, config.Height, MaxPixels/1_000_000)
		return result
	}
	src, _, err := image.Decode(bytes.NewReader(f.Data))
	if err != nil {
		result.Problem = "must be a PNG, JPEG, WebP or GIF image"
		return result
	}
	thumb, err := images.Thumbnail(src)
	if err != nil {
		slog.ErrorContext(ctx, "make submission thumbnail", "error", err)
		result.Problem = "could not be processed"
		return result
	}

	hash := sha256.Sum256(f.Data)
	created, err := s.repo.create(ctx, newSubmission{
		UserID:                userID,
		Original:              f.Data,
		ContentType:           contentType,
		OriginalFilename:      result.Filename,
		Thumb:                 thumb,
		Width:                 config.Width,
		Height:                config.Height,
		Description:           description,
		AttestationAcceptedAt: now,
		AttestationVersion:    version,
		ContentHash:           hex.EncodeToString(hash[:]),
		CreatedAt:             now,
	})
	if errors.Is(err, errDuplicate) {
		result.Problem = "you have already submitted this image"
		result.Duplicate = true
		return result
	}
	if err != nil {
		slog.ErrorContext(ctx, "store submission", "error", err)
		result.Problem = "could not be stored"
		return result
	}
	result.Submission = &created
	return result
}

func (s *Service) List(ctx context.Context, limit, offset int) ([]Summary, int, error) {
	summaries, total, err := s.repo.list(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list submissions: %w", err)
	}
	return summaries, total, nil
}

func (s *Service) Thumb(ctx context.Context, id uuid.UUID) ([]byte, error) {
	return s.repo.thumb(ctx, id)
}

func (s *Service) Original(ctx context.Context, id uuid.UUID) (File, error) {
	return s.repo.original(ctx, id)
}

// Delete hard-deletes the submissions, original and thumbnail both, and logs who did it.
func (s *Service) Delete(ctx context.Context, officerID uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	deleted, err := s.repo.delete(ctx, ids)
	if err != nil {
		return nil, err
	}
	if len(deleted) > 0 {
		slog.InfoContext(ctx, "submissions deleted", "officer_id", officerID, "submission_ids", deleted)
	}
	return deleted, nil
}

// SafeFilename keeps a submitted file name usable in a Content-Disposition header:
// no directories, control characters or quotes, and a bounded length.
func SafeFilename(name string) string {
	name = path.Base(strings.ReplaceAll(name, `\`, "/"))
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == '"' {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." || name == "/" {
		return "submission"
	}
	if runes := []rune(name); len(runes) > maxFilenameRunes {
		name = string(runes[:maxFilenameRunes])
	}
	return name
}
