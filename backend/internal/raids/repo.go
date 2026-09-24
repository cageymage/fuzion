package raids

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Raid struct {
	ID              uuid.UUID `db:"id"               json:"id"`
	Difficulty      string    `db:"difficulty"       json:"difficulty"`
	InstanceName    string    `db:"instance_name"    json:"instanceName"`
	StartsAt        time.Time `db:"starts_at"        json:"startsAt"`
	ProgressSummary string    `db:"progress_summary" json:"progressSummary"`
}

type Repo struct {
	db *sqlx.DB
}

func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) NextRaid(ctx context.Context) (*Raid, error) {
	const query = `
		SELECT id, difficulty, instance_name, starts_at, progress_summary
		FROM raids
		WHERE starts_at > now()
		ORDER BY starts_at ASC
		LIMIT 1`

	var raid Raid
	if err := r.db.GetContext(ctx, &raid, query); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("select next raid: %w", err)
	}
	// pgx hands back timestamptz in the process-local zone; the API always emits UTC.
	raid.StartsAt = raid.StartsAt.UTC()
	return &raid, nil
}

func (r *Repo) ListUpcoming(ctx context.Context) ([]Raid, error) {
	const query = `
		SELECT id, difficulty, instance_name, starts_at, progress_summary
		FROM raids
		WHERE starts_at > now()
		ORDER BY starts_at ASC`

	raids := []Raid{}
	if err := r.db.SelectContext(ctx, &raids, query); err != nil {
		return nil, fmt.Errorf("select upcoming raids: %w", err)
	}
	// pgx hands back timestamptz in the process-local zone; the API always emits UTC.
	for i := range raids {
		raids[i].StartsAt = raids[i].StartsAt.UTC()
	}
	return raids, nil
}
