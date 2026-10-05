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
	ID            uuid.UUID             `db:"id"             json:"id"`
	Name          string                `db:"name"           json:"name"`
	SecondaryName string                `db:"secondary_name" json:"secondaryName"`
	Realm         string                `db:"realm"          json:"realm"`
	Class         string                `db:"class"          json:"class"`
	Spec          *string               `db:"spec"           json:"spec"`
	Role          string                `db:"role"           json:"role"`
	Spec2         *string               `db:"spec2"          json:"spec2"`
	Role2         *string               `db:"role2"          json:"role2"`
	IsMain        bool                  `db:"is_main"        json:"isMain"`
	RaidTeam      *string               `db:"raid_team"      json:"raidTeam"`
	CreatedAt     time.Time             `db:"created_at"     json:"createdAt"`
	Professions   []CharacterProfession `db:"-" json:"professions"`
}

type CharacterProfession struct {
	Profession string `json:"profession"`
	SkillLevel int    `json:"skillLevel"`
}

type Repo struct {
	db *sqlx.DB
}

func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) List(ctx context.Context) ([]Character, error) {
	const query = `
		SELECT id, name, secondary_name, realm, class, spec, role, spec2, role2, is_main, raid_team, created_at
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

	professions, err := r.professionsByCharacter(ctx, nil)
	if err != nil {
		return nil, err
	}
	for i := range characters {
		characters[i].Professions = append([]CharacterProfession{}, professions[characters[i].ID]...)
	}
	return characters, nil
}

// professionsByCharacter returns every character's professions, highest skill first.
// Characters without any are absent from the map, so callers get empty slices.
func (r *Repo) professionsByCharacter(ctx context.Context, only *uuid.UUID) (map[uuid.UUID][]CharacterProfession, error) {
	const query = `
		SELECT character_id, profession, skill_level
		FROM professions
		WHERE $1::uuid IS NULL OR character_id = $1
		ORDER BY skill_level DESC, profession ASC`

	var rows []struct {
		CharacterID uuid.UUID `db:"character_id"`
		Profession  string    `db:"profession"`
		SkillLevel  int       `db:"skill_level"`
	}
	if err := r.db.SelectContext(ctx, &rows, query, only); err != nil {
		return nil, fmt.Errorf("select professions: %w", err)
	}

	byCharacter := make(map[uuid.UUID][]CharacterProfession)
	for _, row := range rows {
		byCharacter[row.CharacterID] = append(byCharacter[row.CharacterID], CharacterProfession{Profession: row.Profession, SkillLevel: row.SkillLevel})
	}
	return byCharacter, nil
}

const returningColumns = `id, name, secondary_name, realm, class, spec, role, spec2, role2, is_main, raid_team, created_at`

const uniqueViolationCode = "23505"

func (r *Repo) Create(ctx context.Context, c Character, professions []string) (Character, error) {
	const query = `
		INSERT INTO characters (id, name, secondary_name, realm, class, spec, role, spec2, role2, is_main, raid_team)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING ` + returningColumns

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return Character{}, fmt.Errorf("begin create character: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op once committed

	var created Character
	if err := tx.GetContext(ctx, &created, query, c.ID, c.Name, c.SecondaryName, c.Realm, c.Class, c.Spec, c.Role, c.Spec2, c.Role2, c.IsMain, c.RaidTeam); err != nil {
		return Character{}, mapWriteError(err)
	}
	if err := replaceProfessions(ctx, tx, created.ID, professions); err != nil {
		return Character{}, err
	}
	if err := tx.Commit(); err != nil {
		return Character{}, fmt.Errorf("commit create character: %w", err)
	}

	created.CreatedAt = created.CreatedAt.UTC()
	stored, err := r.professionsByCharacter(ctx, &created.ID)
	if err != nil {
		return Character{}, err
	}
	created.Professions = append([]CharacterProfession{}, stored[created.ID]...)
	return created, nil
}

// secondSpec carries the resolved second spec and role for a PATCH, where nil
// fields mean "clear" rather than "leave unchanged".
type secondSpec struct {
	spec *string
	role *string
}

func (r *Repo) Get(ctx context.Context, id uuid.UUID) (Character, error) {
	const query = `SELECT ` + returningColumns + ` FROM characters WHERE id = $1`

	var character Character
	if err := r.db.GetContext(ctx, &character, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Character{}, ErrNotFound
		}
		return Character{}, fmt.Errorf("select character: %w", err)
	}
	character.CreatedAt = character.CreatedAt.UTC()
	return character, nil
}

// Update applies req with COALESCE semantics; second, when non-nil, overwrites
// spec2 and role2 outright so they can be cleared. professions, when non-nil,
// replaces the character's whole profession set.
func (r *Repo) Update(ctx context.Context, id uuid.UUID, req UpdateRequest, second *secondSpec, professions *[]string) (Character, error) {
	const query = `
		UPDATE characters SET
			name           = COALESCE($2, name),
			secondary_name = COALESCE($3, secondary_name),
			realm          = COALESCE($4, realm),
			class          = COALESCE($5, class),
			spec           = COALESCE($6, spec),
			role           = COALESCE($7, role),
			is_main        = COALESCE($8, is_main),
			raid_team      = COALESCE($9, raid_team),
			spec2          = CASE WHEN $10::boolean THEN $11::text ELSE spec2 END,
			role2          = CASE WHEN $10::boolean THEN $12::text ELSE role2 END
		WHERE id = $1
		RETURNING ` + returningColumns

	var spec2, role2 *string
	if second != nil {
		spec2, role2 = second.spec, second.role
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return Character{}, fmt.Errorf("begin update character: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op once committed

	var updated Character
	if err := tx.GetContext(ctx, &updated, query, id, req.Name, req.SecondaryName, req.Realm, req.Class, req.Spec, req.Role, req.IsMain, req.RaidTeam, second != nil, spec2, role2); err != nil {
		return Character{}, mapWriteError(err)
	}
	if professions != nil {
		if err := replaceProfessions(ctx, tx, id, *professions); err != nil {
			return Character{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return Character{}, fmt.Errorf("commit update character: %w", err)
	}

	updated.CreatedAt = updated.CreatedAt.UTC()
	stored, err := r.professionsByCharacter(ctx, &id)
	if err != nil {
		return Character{}, err
	}
	updated.Professions = append([]CharacterProfession{}, stored[id]...)
	return updated, nil
}

// replaceProfessions makes names the character's whole profession set. A profession
// that stays keeps its stored skill level; a new one starts at 0.
func replaceProfessions(ctx context.Context, tx *sqlx.Tx, characterID uuid.UUID, names []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM professions WHERE character_id = $1 AND NOT (profession = ANY($2::text[]))`, characterID, names); err != nil {
		return fmt.Errorf("delete dropped professions: %w", err)
	}
	for _, name := range names {
		const insert = `
			INSERT INTO professions (id, character_id, profession, skill_level)
			VALUES ($1, $2, $3, 0)
			ON CONFLICT (character_id, profession) DO NOTHING`
		if _, err := tx.ExecContext(ctx, insert, uuid.New(), characterID, name); err != nil {
			return fmt.Errorf("insert profession %q: %w", name, err)
		}
	}
	return nil
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
