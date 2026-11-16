package professions

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type Recipe struct {
	ID         int         `json:"id"`
	Name       string      `json:"name"`
	Profession string      `json:"profession"`
	Characters []Character `json:"characters"`
}

type recipeRow struct {
	ID                     int       `db:"id"`
	Name                   string    `db:"name"`
	Profession             string    `db:"profession"`
	CharacterID            uuid.UUID `db:"character_id"`
	CharacterName          string    `db:"character_name"`
	CharacterSecondaryName string    `db:"character_secondary_name"`
	CharacterClass         string    `db:"character_class"`
}

// ListRecipes returns recipes whose name contains q (when not empty) and that characterID
// knows (when not nil), each with every character still in the guild who can craft it.
// A recipe nobody in the guild can craft any more is left out.
func (r *Repo) ListRecipes(ctx context.Context, q string, characterID *uuid.UUID) ([]Recipe, error) {
	const query = `
		WITH matched AS (
			SELECT r.id, r.name, r.profession
			FROM recipes r
			WHERE ($1 = '' OR strpos(lower(r.name), lower($1)) > 0)
			  AND ($2::uuid IS NULL OR EXISTS (
			        SELECT 1 FROM character_recipes mine WHERE mine.recipe_id = r.id AND mine.character_id = $2))
			  AND EXISTS (
			        SELECT 1 FROM character_recipes cr JOIN characters c ON c.id = cr.character_id
			        WHERE cr.recipe_id = r.id AND c.left_guild_at IS NULL)
			ORDER BY r.name, r.id
			LIMIT 500
		)
		SELECT m.id, m.name, m.profession,
		       c.id AS character_id, c.name AS character_name,
		       c.secondary_name AS character_secondary_name, c.class AS character_class
		FROM matched m
		JOIN character_recipes cr ON cr.recipe_id = m.id
		JOIN characters c ON c.id = cr.character_id AND c.left_guild_at IS NULL
		ORDER BY m.name, m.id, c.name, c.secondary_name`

	var rows []recipeRow
	if err := r.db.SelectContext(ctx, &rows, query, q, characterID); err != nil {
		return nil, fmt.Errorf("select recipes: %w", err)
	}

	recipes := []Recipe{}
	for _, row := range rows {
		if len(recipes) == 0 || recipes[len(recipes)-1].ID != row.ID {
			recipes = append(recipes, Recipe{ID: row.ID, Name: row.Name, Profession: row.Profession, Characters: []Character{}})
		}
		last := &recipes[len(recipes)-1]
		last.Characters = append(last.Characters, Character{
			ID: row.CharacterID, Name: row.CharacterName, SecondaryName: row.CharacterSecondaryName, Class: row.CharacterClass,
		})
	}
	return recipes, nil
}

func (s *Service) ListRecipes(ctx context.Context, q, characterID string) ([]Recipe, error) {
	var filter *uuid.UUID
	if characterID = strings.TrimSpace(characterID); characterID != "" {
		id, err := uuid.Parse(characterID)
		if err != nil {
			return nil, errUnknownCharacter()
		}
		filter = &id
	}

	recipes, err := s.repo.ListRecipes(ctx, strings.TrimSpace(q), filter)
	if err != nil {
		return nil, fmt.Errorf("recipes: %w", err)
	}
	return recipes, nil
}

func (h *Handler) listRecipes(w http.ResponseWriter, r *http.Request) {
	recipes, err := h.service.ListRecipes(r.Context(), r.URL.Query().Get("q"), r.URL.Query().Get("characterId"))
	if err != nil {
		writeError(w, r, "list recipes", err)
		return
	}
	writeJSON(w, http.StatusOK, recipes)
}
