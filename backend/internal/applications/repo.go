package applications

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

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
