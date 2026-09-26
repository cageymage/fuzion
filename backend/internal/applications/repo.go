package applications

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var ErrNotFound = errors.New("application not found")

type Application struct {
	ID            uuid.UUID `db:"id"`
	ApplicantName string    `db:"applicant_name"`
	CharacterName string    `db:"character_name"`
	Class         string    `db:"class"`
	Role          string    `db:"role"`
	Availability  string    `db:"availability"`
	DiscordHandle string    `db:"discord_handle"`
	Notes         string    `db:"notes"`
}

type Submitted struct {
	ID          uuid.UUID `db:"id"           json:"id"`
	Status      string    `db:"status"       json:"status"`
	SubmittedAt time.Time `db:"submitted_at" json:"submittedAt"`
}

type Reviewable struct {
	ID            uuid.UUID     `db:"id"             json:"id"`
	ApplicantName string        `db:"applicant_name" json:"applicantName"`
	CharacterName string        `db:"character_name" json:"characterName"`
	Class         string        `db:"class"          json:"class"`
	Role          string        `db:"role"           json:"role"`
	Availability  string        `db:"availability"   json:"availability"`
	DiscordHandle string        `db:"discord_handle" json:"discordHandle"`
	Notes         string        `db:"notes"          json:"notes"`
	Status        string        `db:"status"         json:"status"`
	SubmittedAt   time.Time     `db:"submitted_at"   json:"submittedAt"`
	ReviewedBy    uuid.NullUUID `db:"reviewed_by"    json:"reviewedBy"`
	ReviewedAt    *time.Time    `db:"reviewed_at"    json:"reviewedAt"`
	ReviewNote    string        `db:"review_note"    json:"reviewNote"`
}

const reviewableColumns = `id, applicant_name, character_name, class, role, availability, discord_handle, notes,
	status, submitted_at, reviewed_by, reviewed_at, review_note`

type Repo struct {
	db *sqlx.DB
}

func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) Create(ctx context.Context, app Application) (Submitted, error) {
	const query = `
		INSERT INTO applications (id, applicant_name, character_name, class, role, availability, discord_handle, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, status, submitted_at`

	var submitted Submitted
	err := r.db.GetContext(ctx, &submitted, query,
		app.ID, app.ApplicantName, app.CharacterName, app.Class, app.Role, app.Availability, app.DiscordHandle, app.Notes)
	if err != nil {
		return Submitted{}, fmt.Errorf("insert application: %w", err)
	}
	// pgx hands back timestamptz in the process-local zone; the API always emits UTC.
	submitted.SubmittedAt = submitted.SubmittedAt.UTC()
	return submitted, nil
}

// List returns applications newest first; an empty status means every status.
func (r *Repo) List(ctx context.Context, status string) ([]Reviewable, error) {
	const query = `
		SELECT ` + reviewableColumns + `
		FROM applications
		WHERE $1::text = '' OR status = $1::text
		ORDER BY submitted_at DESC, id`

	apps := []Reviewable{}
	if err := r.db.SelectContext(ctx, &apps, query, status); err != nil {
		return nil, fmt.Errorf("select applications: %w", err)
	}
	for i := range apps {
		apps[i].normalizeTimes()
	}
	return apps, nil
}

func (r *Repo) Get(ctx context.Context, id uuid.UUID) (Reviewable, error) {
	const query = `SELECT ` + reviewableColumns + ` FROM applications WHERE id = $1`

	var app Reviewable
	err := r.db.GetContext(ctx, &app, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Reviewable{}, ErrNotFound
	}
	if err != nil {
		return Reviewable{}, fmt.Errorf("select application %s: %w", id, err)
	}
	app.normalizeTimes()
	return app, nil
}

func (r *Repo) Review(ctx context.Context, id uuid.UUID, status, note string, reviewer uuid.UUID, reviewedAt time.Time) (Reviewable, error) {
	const query = `
		UPDATE applications
		SET status = $2, review_note = $3, reviewed_by = $4, reviewed_at = $5
		WHERE id = $1
		RETURNING ` + reviewableColumns

	var app Reviewable
	err := r.db.GetContext(ctx, &app, query, id, status, note, reviewer, reviewedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Reviewable{}, ErrNotFound
	}
	if err != nil {
		return Reviewable{}, fmt.Errorf("update application %s: %w", id, err)
	}
	app.normalizeTimes()
	return app, nil
}

func (a *Reviewable) normalizeTimes() {
	a.SubmittedAt = a.SubmittedAt.UTC()
	if a.ReviewedAt != nil {
		utc := a.ReviewedAt.UTC()
		a.ReviewedAt = &utc
	}
}
