package roster

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

type Character struct {
	ID            uuid.UUID `db:"id"             json:"id"`
	Name          string    `db:"name"           json:"name"`
	SecondaryName string    `db:"secondary_name" json:"secondaryName"`
	Realm         string    `db:"realm"          json:"realm"`
	Class         string    `db:"class"          json:"class"`
	Spec          *string   `db:"spec"           json:"spec"`
	Role          string    `db:"role"           json:"role"`
	IsMain        bool      `db:"is_main"        json:"isMain"`
	RaidTeam      *string   `db:"raid_team"      json:"raidTeam"`
	CreatedAt     time.Time `db:"created_at"     json:"createdAt"`
}

type Repo struct {
	db *sqlx.DB
}

func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) List(ctx context.Context) ([]Character, error) {
	const query = `
		SELECT id, name, secondary_name, realm, class, spec, role, is_main, raid_team, created_at
		FROM characters
		ORDER BY is_main DESC, name ASC, secondary_name ASC`

	characters := []Character{}
	if err := r.db.SelectContext(ctx, &characters, query); err != nil {
		return nil, fmt.Errorf("select characters: %w", err)
	}
	// pgx hands back timestamptz in the process-local zone; the API always emits UTC.
	for i := range characters {
		characters[i].CreatedAt = characters[i].CreatedAt.UTC()
	}
	return characters, nil
}

const returningColumns = `id, name, secondary_name, realm, class, spec, role, is_main, raid_team, created_at`

const uniqueViolationCode = "23505"

func (r *Repo) Create(ctx context.Context, c Character) (Character, error) {
	const query = `
		INSERT INTO characters (id, name, secondary_name, realm, class, spec, role, is_main, raid_team)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING ` + returningColumns

	var created Character
	if err := r.db.GetContext(ctx, &created, query, c.ID, c.Name, c.SecondaryName, c.Realm, c.Class, c.Spec, c.Role, c.IsMain, c.RaidTeam); err != nil {
		return Character{}, mapWriteError(err)
	}
	created.CreatedAt = created.CreatedAt.UTC()
	return created, nil
}

func (r *Repo) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (Character, error) {
	const query = `
		UPDATE characters SET
			name           = COALESCE($2, name),
			secondary_name = COALESCE($3, secondary_name),
			realm          = COALESCE($4, realm),
			class          = COALESCE($5, class),
			spec           = COALESCE($6, spec),
			role           = COALESCE($7, role),
			is_main        = COALESCE($8, is_main),
			raid_team      = COALESCE($9, raid_team)
		WHERE id = $1
		RETURNING ` + returningColumns

	var updated Character
	if err := r.db.GetContext(ctx, &updated, query, id, req.Name, req.SecondaryName, req.Realm, req.Class, req.Spec, req.Role, req.IsMain, req.RaidTeam); err != nil {
		return Character{}, mapWriteError(err)
	}
	updated.CreatedAt = updated.CreatedAt.UTC()
	return updated, nil
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM characters WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete character: %w", err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count deleted characters: %w", err)
	}
	if deleted == 0 {
		return ErrNotFound
	}
	return nil
}

func mapWriteError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
		return ErrConflict
	}
	return fmt.Errorf("write character: %w", err)
}
