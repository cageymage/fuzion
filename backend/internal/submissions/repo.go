package submissions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

const uniqueViolationCode = "23505"

var (
	ErrNotFound  = errors.New("submission not found")
	errDuplicate = errors.New("duplicate submission")
)

type newSubmission struct {
	UserID                uuid.UUID
	Original              []byte
	ContentType           string
	OriginalFilename      string
	Thumb                 []byte
	Width                 int
	Height                int
	Description           string
	AttestationAcceptedAt time.Time
	AttestationVersion    string
	ContentHash           string
	CreatedAt             time.Time
}

// Summary is everything about a submission except its image bytes.
type Summary struct {
	ID                uuid.UUID `db:"id"`
	SubmitterID       uuid.UUID `db:"user_id"`
	SubmitterUsername string    `db:"username"`
	Description       string    `db:"description"`
	OriginalFilename  string    `db:"original_filename"`
	ContentType       string    `db:"content_type"`
	Width             int       `db:"width"`
	Height            int       `db:"height"`
	ByteSize          int       `db:"byte_size"`
	CreatedAt         time.Time `db:"created_at"`
}

type File struct {
	Bytes       []byte `db:"original_bytes"`
	ContentType string `db:"content_type"`
	Filename    string `db:"original_filename"`
}

type Repo struct {
	db *sqlx.DB
}

func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

const summaryColumns = `
	s.id, s.user_id, u.username, s.description, s.original_filename, s.content_type,
	s.width, s.height, s.byte_size, s.created_at`

func (r *Repo) create(ctx context.Context, s newSubmission) (Summary, error) {
	const query = `
		WITH s AS (
			INSERT INTO submissions (
				user_id, original_bytes, byte_size, content_type, original_filename, thumb_bytes,
				width, height, description, attestation_accepted_at, attestation_version, content_hash, created_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			RETURNING *
		)
		SELECT ` + summaryColumns + `
		FROM s JOIN users u ON u.id = s.user_id`

	var created Summary
	err := r.db.GetContext(ctx, &created, query,
		s.UserID, s.Original, len(s.Original), s.ContentType, s.OriginalFilename, s.Thumb,
		s.Width, s.Height, s.Description, s.AttestationAcceptedAt, s.AttestationVersion, s.ContentHash, s.CreatedAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
		return Summary{}, errDuplicate
	}
	if err != nil {
		return Summary{}, fmt.Errorf("insert submission: %w", err)
	}
	created.CreatedAt = created.CreatedAt.UTC()
	return created, nil
}

// usageSince counts the user's submissions and their bytes created after since.
func (r *Repo) usageSince(ctx context.Context, userID uuid.UUID, since time.Time) (files int, bytes int64, err error) {
	const query = `
		SELECT count(*) AS files, COALESCE(sum(byte_size), 0) AS bytes
		FROM submissions
		WHERE user_id = $1 AND created_at > $2`

	var usage struct {
		Files int   `db:"files"`
		Bytes int64 `db:"bytes"`
	}
	if err := r.db.GetContext(ctx, &usage, query, userID, since); err != nil {
		return 0, 0, fmt.Errorf("select submission usage: %w", err)
	}
	return usage.Files, usage.Bytes, nil
}

func (r *Repo) totalBytes(ctx context.Context) (int64, error) {
	var total int64
	if err := r.db.GetContext(ctx, &total, `SELECT COALESCE(sum(byte_size), 0) FROM submissions`); err != nil {
		return 0, fmt.Errorf("select submission storage: %w", err)
	}
	return total, nil
}

func (r *Repo) list(ctx context.Context, limit, offset int) ([]Summary, int, error) {
	const query = `
		SELECT ` + summaryColumns + `
		FROM submissions s JOIN users u ON u.id = s.user_id
		ORDER BY s.created_at DESC, s.id
		LIMIT $1 OFFSET $2`

	summaries := []Summary{}
	if err := r.db.SelectContext(ctx, &summaries, query, limit, offset); err != nil {
		return nil, 0, fmt.Errorf("select submissions: %w", err)
	}
	for i := range summaries {
		summaries[i].CreatedAt = summaries[i].CreatedAt.UTC()
	}
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT count(*) FROM submissions`); err != nil {
		return nil, 0, fmt.Errorf("count submissions: %w", err)
	}
	return summaries, total, nil
}

func (r *Repo) thumb(ctx context.Context, id uuid.UUID) ([]byte, error) {
	var data []byte
	err := r.db.GetContext(ctx, &data, `SELECT thumb_bytes FROM submissions WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("select submission thumbnail: %w", err)
	}
	return data, nil
}

func (r *Repo) original(ctx context.Context, id uuid.UUID) (File, error) {
	const query = `SELECT original_bytes, content_type, original_filename FROM submissions WHERE id = $1`

	var o File
	err := r.db.GetContext(ctx, &o, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return File{}, ErrNotFound
	}
	if err != nil {
		return File{}, fmt.Errorf("select submission: %w", err)
	}
	return o, nil
}

// delete removes the given submissions and returns the ids that existed.
func (r *Repo) delete(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	idStrings := make([]string, len(ids))
	for i, id := range ids {
		idStrings[i] = id.String()
	}
	deleted := []uuid.UUID{}
	if err := r.db.SelectContext(ctx, &deleted, `DELETE FROM submissions WHERE id = ANY($1::uuid[]) RETURNING id`, idStrings); err != nil {
		return nil, fmt.Errorf("delete submissions: %w", err)
	}
	return deleted, nil
}
