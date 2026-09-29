package raidprogress

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var (
	ErrNoCurrentTier = errors.New("no current raid tier")
	ErrTierNotFound  = errors.New("raid tier not found")
	ErrBossNotFound  = errors.New("raid boss not found")
	ErrTierIsCurrent = errors.New("raid tier is current; clear its current flag before deleting it")
)

type Boss struct {
	ID       uuid.UUID  `db:"id"        json:"id"`
	Name     string     `db:"name"      json:"name"`
	KilledAt *time.Time `db:"killed_at" json:"killedAt"`
}

type Tier struct {
	ID        uuid.UUID `db:"id"         json:"id"`
	Name      string    `db:"name"       json:"name"`
	IsCurrent bool      `db:"is_current" json:"isCurrent"`
	SortOrder int       `db:"sort_order" json:"sortOrder"`
	Bosses    []Boss    `json:"bosses"`
}

type Progress struct {
	Tier struct {
		Name string `json:"name"`
	} `json:"tier"`
	Bosses []Boss `json:"bosses"`
	Killed int    `json:"killed"`
	Total  int    `json:"total"`
}

type Repo struct {
	db *sqlx.DB
}

func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

// CurrentProgress returns every tier currently being raided. Forever ships at
// least two raids per tier (see product-spec.md), so "current" is not
// exclusive to one row the way an earlier version of this schema assumed.
func (r *Repo) CurrentProgress(ctx context.Context) ([]Progress, error) {
	var tiers []struct {
		ID   uuid.UUID `db:"id"`
		Name string    `db:"name"`
	}
	const tierQuery = `SELECT id, name FROM raid_tiers WHERE is_current = true ORDER BY sort_order DESC, created_at DESC`
	if err := r.db.SelectContext(ctx, &tiers, tierQuery); err != nil {
		return nil, fmt.Errorf("select current raid tiers: %w", err)
	}
	if len(tiers) == 0 {
		return nil, ErrNoCurrentTier
	}

	const bossQuery = `SELECT id, name, killed_at FROM raid_bosses WHERE tier_id = $1 ORDER BY sort_order ASC`
	progress := make([]Progress, len(tiers))
	for i, tier := range tiers {
		bosses := []Boss{}
		if err := r.db.SelectContext(ctx, &bosses, bossQuery, tier.ID); err != nil {
			return nil, fmt.Errorf("select bosses for tier %s: %w", tier.ID, err)
		}
		for j := range bosses {
			normalizeKilledAt(&bosses[j])
		}

		killed := 0
		for _, b := range bosses {
			if b.KilledAt != nil {
				killed++
			}
		}

		progress[i] = Progress{Bosses: bosses, Killed: killed, Total: len(bosses)}
		progress[i].Tier.Name = tier.Name
	}
	return progress, nil
}

func (r *Repo) ListTiers(ctx context.Context) ([]Tier, error) {
	const tierQuery = `SELECT id, name, is_current, sort_order FROM raid_tiers ORDER BY sort_order DESC, created_at DESC`
	tiers := []Tier{}
	if err := r.db.SelectContext(ctx, &tiers, tierQuery); err != nil {
		return nil, fmt.Errorf("select raid tiers: %w", err)
	}

	type bossRow struct {
		ID       uuid.UUID  `db:"id"`
		TierID   uuid.UUID  `db:"tier_id"`
		Name     string     `db:"name"`
		KilledAt *time.Time `db:"killed_at"`
	}
	const bossQuery = `SELECT id, tier_id, name, killed_at FROM raid_bosses ORDER BY tier_id, sort_order ASC`
	var rows []bossRow
	if err := r.db.SelectContext(ctx, &rows, bossQuery); err != nil {
		return nil, fmt.Errorf("select raid bosses: %w", err)
	}

	bossesByTier := make(map[uuid.UUID][]Boss, len(tiers))
	for _, row := range rows {
		boss := Boss{ID: row.ID, Name: row.Name, KilledAt: row.KilledAt}
		normalizeKilledAt(&boss)
		bossesByTier[row.TierID] = append(bossesByTier[row.TierID], boss)
	}

	for i := range tiers {
		if bosses, ok := bossesByTier[tiers[i].ID]; ok {
			tiers[i].Bosses = bosses
		} else {
			tiers[i].Bosses = []Boss{}
		}
	}
	return tiers, nil
}

func (r *Repo) CreateTier(ctx context.Context, name string, sortOrder int, bossNames []string) (created Tier, err error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return Tier{}, fmt.Errorf("begin create tier: %w", err)
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				err = errors.Join(err, fmt.Errorf("rollback create tier: %w", rollbackErr))
			}
		}
	}()

	const tierQuery = `INSERT INTO raid_tiers (name, sort_order) VALUES ($1, $2) RETURNING id, name, is_current, sort_order`
	if err = tx.GetContext(ctx, &created, tierQuery, name, sortOrder); err != nil {
		return Tier{}, fmt.Errorf("insert raid tier: %w", err)
	}

	created.Bosses = make([]Boss, len(bossNames))
	const bossQuery = `INSERT INTO raid_bosses (tier_id, name, sort_order) VALUES ($1, $2, $3) RETURNING id, name, killed_at`
	for i, bossName := range bossNames {
		if err = tx.GetContext(ctx, &created.Bosses[i], bossQuery, created.ID, bossName, i); err != nil {
			return Tier{}, fmt.Errorf("insert raid boss %q: %w", bossName, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return Tier{}, fmt.Errorf("commit create tier: %w", err)
	}
	return created, nil
}

// SetTierCurrent flips just this tier's own flag. Multiple tiers can be
// current at once (see CurrentProgress), so setting one current never
// touches any other tier's row.
func (r *Repo) SetTierCurrent(ctx context.Context, id uuid.UUID, isCurrent bool) (Tier, error) {
	const query = `UPDATE raid_tiers SET is_current = $2 WHERE id = $1 RETURNING id, name, is_current, sort_order`

	var updated Tier
	if err := r.db.GetContext(ctx, &updated, query, id, isCurrent); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Tier{}, ErrTierNotFound
		}
		return Tier{}, fmt.Errorf("update raid tier %s: %w", id, err)
	}
	updated.Bosses = []Boss{}
	return updated, nil
}

func (r *Repo) SetBossKilled(ctx context.Context, id uuid.UUID, killedAt *time.Time) (Boss, error) {
	const query = `UPDATE raid_bosses SET killed_at = $2 WHERE id = $1 RETURNING id, name, killed_at`

	var boss Boss
	if err := r.db.GetContext(ctx, &boss, query, id, killedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Boss{}, ErrBossNotFound
		}
		return Boss{}, fmt.Errorf("update raid boss %s: %w", id, err)
	}
	normalizeKilledAt(&boss)
	return boss, nil
}

// DeleteTier refuses a current tier so the public progress widget never loses
// a raid out from under it; the tier's bosses go with it via ON DELETE CASCADE.
func (r *Repo) DeleteTier(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM raid_tiers WHERE id = $1 AND is_current = false`, id)
	if err != nil {
		return fmt.Errorf("delete raid tier %s: %w", id, err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count deleted raid tiers: %w", err)
	}
	if deleted > 0 {
		return nil
	}

	var isCurrent bool
	if err := r.db.GetContext(ctx, &isCurrent, `SELECT is_current FROM raid_tiers WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrTierNotFound
		}
		return fmt.Errorf("check raid tier %s: %w", id, err)
	}
	return ErrTierIsCurrent
}

func (r *Repo) DeleteBoss(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM raid_bosses WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete raid boss %s: %w", id, err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count deleted raid bosses: %w", err)
	}
	if deleted == 0 {
		return ErrBossNotFound
	}
	return nil
}

// pgx hands back timestamptz in the process-local zone; the API always emits UTC.
func normalizeKilledAt(b *Boss) {
	if b.KilledAt != nil {
		utc := b.KilledAt.UTC()
		b.KilledAt = &utc
	}
}
