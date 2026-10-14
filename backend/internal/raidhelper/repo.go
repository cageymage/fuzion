package raidhelper

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type mirroredRaid struct {
	EventID     string
	Difficulty  string
	Title       string
	StartsAt    time.Time
	LastUpdated int64
}

type Repo struct {
	db *sqlx.DB
}

func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

// Upsert leaves the row alone when it is a tombstone or already holds a strictly newer
// lastUpdated, so a late or duplicate delivery cannot undo a newer one or a delete.
func (r *Repo) Upsert(ctx context.Context, raid mirroredRaid) error {
	const query = `
		INSERT INTO raids (difficulty, instance_name, starts_at, progress_summary, raid_helper_event_id, raid_helper_last_updated)
		VALUES ($1, $2, $3, '', $4, $5)
		ON CONFLICT (raid_helper_event_id) DO UPDATE
		SET difficulty = EXCLUDED.difficulty,
			instance_name = EXCLUDED.instance_name,
			starts_at = EXCLUDED.starts_at,
			raid_helper_last_updated = EXCLUDED.raid_helper_last_updated
		WHERE raids.deleted_at IS NULL
			AND raids.raid_helper_last_updated <= EXCLUDED.raid_helper_last_updated`
	if _, err := r.db.ExecContext(ctx, query, raid.Difficulty, raid.Title, raid.StartsAt, raid.EventID, raid.LastUpdated); err != nil {
		return fmt.Errorf("upsert raid-helper event %s: %w", raid.EventID, err)
	}
	return nil
}

// MarkDeleted applies regardless of lastUpdated: a delete carries the same value as the
// update before it. An event never seen before is stored as a tombstone straight away.
func (r *Repo) MarkDeleted(ctx context.Context, raid mirroredRaid) error {
	const query = `
		INSERT INTO raids (difficulty, instance_name, starts_at, progress_summary, raid_helper_event_id, raid_helper_last_updated, deleted_at)
		VALUES ($1, $2, $3, '', $4, $5, now())
		ON CONFLICT (raid_helper_event_id) DO UPDATE
		SET deleted_at = COALESCE(raids.deleted_at, now())`
	if _, err := r.db.ExecContext(ctx, query, raid.Difficulty, raid.Title, raid.StartsAt, raid.EventID, raid.LastUpdated); err != nil {
		return fmt.Errorf("mark raid-helper event %s deleted: %w", raid.EventID, err)
	}
	return nil
}
