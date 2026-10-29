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
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
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

	// Rejects decompression bombs before a full decode. A decoded image costs about
	// 4 bytes per pixel, so this is roughly 100 MB on a 512 MB instance, and still
	// fits a 6000×4000 photo. Files are decoded one at a time, and the handler lets
	// only a few uploads run at once.
	MaxPixels = 25_000_000

	// The status endpoint is public, so it reads the storage total from a short cache
	// rather than summing the table on every page load. Uploads use the exact total.
	statusCacheTTL = 30 * time.Second
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

	mu            sync.Mutex
	cachedTotal   int64
	cachedTotalAt time.Time
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
	total, err := s.statusTotal(ctx)
	if err != nil {
		return false, fmt.Errorf("check submission storage: %w", err)
	}
	return total < s.storageLimit, nil
}

func (s *Service) statusTotal(ctx context.Context) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.clock.Now()
	if !s.cachedTotalAt.IsZero() && now.Sub(s.cachedTotalAt) < statusCacheTTL {
		return s.cachedTotal, nil
	}
	total, err := s.repo.totalBytes(ctx, s.repo.db)
	if err != nil {
		return 0, err
	}
	s.cachedTotal, s.cachedTotalAt = total, now
	return total, nil
}

// forgetTotal drops the cached total after an upload or delete, so the status
// reflects it on the next call instead of up to statusCacheTTL later.
func (s *Service) forgetTotal() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cachedTotalAt = time.Time{}
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
	now := s.clock.Now().UTC()

	// Checked once before decoding, so an over-quota member costs no CPU, and again
	// under the upload lock, where it decides.
	if err := s.checkLimits(ctx, s.repo.db, userID, req, now); err != nil {
		return nil, err
	}

	results := make([]FileResult, len(req.Files))
	prepared := make([]*newSubmission, len(req.Files))
	for i, f := range req.Files {
		// A client that has gone away gets no more decoding or inserts.
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		description := ""
		if i < len(req.Descriptions) {
			description = strings.TrimSpace(req.Descriptions[i])
		}
		results[i], prepared[i] = s.prepare(ctx, userID, f, description, req.AttestationVersion, now)
	}

	created := 0
	err := s.repo.withUploadLock(ctx, func(q sqlx.ExtContext) error {
		if err := s.checkLimits(ctx, q, userID, req, now); err != nil {
			return err
		}
		for i, submission := range prepared {
			if submission == nil {
				continue
			}
			stored, err := s.repo.create(ctx, q, *submission)
			if errors.Is(err, errDuplicate) {
				results[i].Problem = "you have already submitted this image"
				results[i].Duplicate = true
				continue
			}
			if err != nil {
				return err
			}
			results[i].Submission = &stored
			created++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if created > 0 {
		s.forgetTotal()
	}
	return results, nil
}

// checkLimits applies the storage ceiling and the member's daily quota to the whole
// batch, counting every file in it, even ones that will fail validation.
func (s *Service) checkLimits(ctx context.Context, q sqlx.QueryerContext, userID uuid.UUID, req UploadRequest, now time.Time) error {
	var batchBytes int64
	for _, f := range req.Files {
		if !f.TooLarge {
			batchBytes += int64(len(f.Data))
		}
	}

	total, err := s.repo.totalBytes(ctx, q)
	if err != nil {
		return fmt.Errorf("check submission storage: %w", err)
	}
	if total+batchBytes > s.storageLimit {
		return ErrStorageFull
	}

	files, usedBytes, err := s.repo.usageSince(ctx, q, userID, now.Add(-quotaWindow))
	if err != nil {
		return err
	}
	if files+len(req.Files) > DailyFileQuota {
		return &QuotaError{Problem: fmt.Sprintf("you can submit at most %d images per 24 hours; %d left", DailyFileQuota, max(0, DailyFileQuota-files))}
	}
	if usedBytes+batchBytes > DailyByteQuota {
		return &QuotaError{Problem: fmt.Sprintf("you can submit at most %d MiB per 24 hours", DailyByteQuota>>20)}
	}
	return nil
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

// prepare validates, decodes and thumbnails one file without touching the database.
// It returns the submission to insert, or nil with the reason in the result.
func (s *Service) prepare(ctx context.Context, userID uuid.UUID, f UploadFile, description, version string, now time.Time) (FileResult, *newSubmission) {
	result := FileResult{Filename: SafeFilename(f.Filename)}
	if f.TooLarge {
		result.Problem = fmt.Sprintf("must be at most %d MiB", MaxFileBytes>>20)
		return result, nil
	}

	// The declared type is the browser's guess from the file name, so only the
	// bytes decide what the file is.
	contentType := http.DetectContentType(f.Data)
	if !allowedTypes[contentType] {
		result.Problem = "must be a PNG, JPEG, WebP or GIF image"
		return result, nil
	}
	// For a GIF this is the logical screen. image.Decode keeps only the first frame,
	// and the decoder rejects frames larger than the screen, so this also bounds a GIF
	// with many frames.
	config, _, err := image.DecodeConfig(bytes.NewReader(f.Data))
	if err != nil || config.Width < 1 || config.Height < 1 {
		result.Problem = "must be a PNG, JPEG, WebP or GIF image"
		return result, nil
	}
	if config.Width*config.Height > MaxPixels {
		result.Problem = fmt.Sprintf("is %d×%d pixels; images must be at most %d megapixels", config.Width, config.Height, MaxPixels/1_000_000)
		return result, nil
	}
	src, _, err := image.Decode(bytes.NewReader(f.Data))
	if err != nil {
		result.Problem = "must be a PNG, JPEG, WebP or GIF image"
		return result, nil
	}
	thumb, err := images.Thumbnail(src)
	if err != nil {
		slog.ErrorContext(ctx, "make submission thumbnail", "error", err)
		result.Problem = "could not be processed"
		return result, nil
	}

	hash := sha256.Sum256(f.Data)
	return result, &newSubmission{
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
	}
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
		s.forgetTotal()
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
