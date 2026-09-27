package professions

import (
	"context"
	"fmt"

	"github.com/google/uuid"
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

type Repo struct {
	db *sqlx.DB
}

func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) List(ctx context.Context, profession string) ([]Profession, error) {
	const query = `
		SELECT p.id, p.profession, p.skill_level,
		       c.id AS character_id, c.name AS character_name,
		       c.secondary_name AS character_secondary_name, c.class AS character_class
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
		professions[i] = Profession{
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
	return professions, nil
}
