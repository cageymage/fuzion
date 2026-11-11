package professions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

type Character struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	SecondaryName string    `json:"secondaryName"`
	Class         string    `json:"class"`
}

type Profession struct {
	ID         uuid.UUID `json:"id"`
	Profession string    `json:"profession"`
	SkillLevel int       `json:"skillLevel"`
	Character  Character `json:"character"`
}

type row struct {
	ID                     uuid.UUID `db:"id"`
	Profession             string    `db:"profession"`
	SkillLevel             int       `db:"skill_level"`
	CharacterID            uuid.UUID `db:"character_id"`
	CharacterName          string    `db:"character_name"`
	CharacterSecondaryName string    `db:"character_secondary_name"`
	CharacterClass         string    `db:"character_class"`
}

func (p row) toProfession() Profession {
	return Profession{
		ID:         p.ID,
		Profession: p.Profession,
		SkillLevel: p.SkillLevel,
		Character: Character{
			ID:            p.CharacterID,
			Name:          p.CharacterName,
			SecondaryName: p.CharacterSecondaryName,
			Class:         p.CharacterClass,
		},
	}
}

const (
	uniqueViolationCode     = "23505"
	foreignKeyViolationCode = "23503"
)

const selectColumns = `
	p.id, p.profession, p.skill_level,
	c.id AS character_id, c.name AS character_name,
	c.secondary_name AS character_secondary_name, c.class AS character_class`

type Repo struct {
	db *sqlx.DB
}

func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) List(ctx context.Context, profession string) ([]Profession, error) {
	const query = `
		SELECT ` + selectColumns + `
		FROM professions p
		JOIN characters c ON c.id = p.character_id
		WHERE $1 = '' OR lower(p.profession) = lower($1)
		ORDER BY p.profession ASC, p.skill_level DESC, c.name ASC, c.secondary_name ASC, p.id ASC`

	var rows []row
	if err := r.db.SelectContext(ctx, &rows, query, profession); err != nil {
		return nil, fmt.Errorf("select professions: %w", err)
	}

	professions := make([]Profession, len(rows))
	for i, p := range rows {
		professions[i] = p.toProfession()
	}
	return professions, nil
}

func (r *Repo) Create(ctx context.Context, id, characterID uuid.UUID, profession string, skillLevel int) (Profession, error) {
	const query = `
		WITH p AS (
			INSERT INTO professions (id, character_id, profession, skill_level)
			VALUES ($1, $2, $3, $4)
			RETURNING id, character_id, profession, skill_level
		)
		SELECT ` + selectColumns + `
		FROM p
		JOIN characters c ON c.id = p.character_id`

	var created row
	if err := r.db.GetContext(ctx, &created, query, id, characterID, profession, skillLevel); err != nil {
		return Profession{}, mapWriteError(err)
	}
	return created.toProfession(), nil
}

// Update applies req with COALESCE semantics, so nil fields keep their stored value.
func (r *Repo) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (Profession, error) {
	const query = `
		WITH p AS (
			UPDATE professions SET
				profession  = COALESCE($2, profession),
				skill_level = COALESCE($3, skill_level)
			WHERE id = $1
			RETURNING id, character_id, profession, skill_level
		)
		SELECT ` + selectColumns + `
		FROM p
		JOIN characters c ON c.id = p.character_id`

	var updated row
	if err := r.db.GetContext(ctx, &updated, query, id, req.Profession, req.SkillLevel); err != nil {
		return Profession{}, mapWriteError(err)
	}
	return updated.toProfession(), nil
}

func (r *Repo) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM professions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete profession: %w", err)
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count deleted professions: %w", err)
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
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case uniqueViolationCode:
			return ErrConflict
		case foreignKeyViolationCode:
			return ErrUnknownCharacter
		}
	}
	return fmt.Errorf("write profession: %w", err)
}
