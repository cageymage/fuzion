package professions

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/cageymage/fuzion/backend/internal/blizzard"
	"github.com/cageymage/fuzion/backend/internal/synclog"
)

const blizzardSyncSource = "blizzard-professions"

// Characters imported before the roster sync read surnames still carry this secondary
// name, which gives no Blizzard slug to ask about.
const placeholderSecondaryName = "Unset"

type BlizzardProfessions interface {
	CharacterProfessions(ctx context.Context, characterSlug string) ([]blizzard.CharacterProfession, error)
}

type SyncBlizzardProfessions struct {
	repo     *Repo
	blizzard BlizzardProfessions
	syncLog  *synclog.Repo
}

func NewSyncBlizzardProfessions(repo *Repo, client BlizzardProfessions, syncLog *synclog.Repo) *SyncBlizzardProfessions {
	return &SyncBlizzardProfessions{repo: repo, blizzard: client, syncLog: syncLog}
}

func (j *SyncBlizzardProfessions) Run(ctx context.Context) error {
	message, err := j.sync(ctx)
	if err != nil {
		if logErr := j.syncLog.Record(ctx, blizzardSyncSource, synclog.StatusError, err.Error()); logErr != nil {
			return errors.Join(err, logErr)
		}
		return err
	}
	return j.syncLog.Record(ctx, blizzardSyncSource, synclog.StatusOK, message)
}

func (j *SyncBlizzardProfessions) sync(ctx context.Context) (string, error) {
	characters, err := j.repo.syncableCharacters(ctx)
	if err != nil {
		return "", err
	}

	stored, notFound := 0, 0
	for _, c := range characters {
		slug := strings.ToLower(c.Name) + "-" + strings.ToLower(c.SecondaryName)
		known, err := j.blizzard.CharacterProfessions(ctx, slug)
		if errors.Is(err, blizzard.ErrNotFound) {
			slog.Warn("blizzard has no professions for character", "character", slug)
			notFound++
			continue
		}
		if err != nil {
			return "", fmt.Errorf("ask blizzard for the professions of %s: %w", slug, err)
		}
		if err := j.repo.upsertFromBlizzard(ctx, c.ID, known); err != nil {
			return "", fmt.Errorf("store professions of %s: %w", slug, err)
		}
		stored += len(known)
	}
	return fmt.Sprintf("%d characters: %d professions stored, %d not found", len(characters), stored, notFound), nil
}

type syncableCharacter struct {
	ID            uuid.UUID `db:"id"`
	Name          string    `db:"name"`
	SecondaryName string    `db:"secondary_name"`
}

func (r *Repo) syncableCharacters(ctx context.Context) ([]syncableCharacter, error) {
	const query = `
		SELECT id, name, secondary_name FROM characters
		WHERE left_guild_at IS NULL AND secondary_name <> '` + placeholderSecondaryName + `'
		ORDER BY name, secondary_name`

	var characters []syncableCharacter
	if err := r.db.SelectContext(ctx, &characters, query); err != nil {
		return nil, fmt.Errorf("select characters to sync professions for: %w", err)
	}
	return characters, nil
}

// upsertFromBlizzard lets Blizzard win for the professions it reports; professions it
// does not report are left as an officer set them.
func (r *Repo) upsertFromBlizzard(ctx context.Context, characterID uuid.UUID, known []blizzard.CharacterProfession) error {
	const upsert = `
		INSERT INTO professions (id, character_id, profession, skill_level)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (character_id, profession) DO UPDATE SET skill_level = EXCLUDED.skill_level`

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin professions upsert: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op once committed

	for _, p := range known {
		if _, err := tx.ExecContext(ctx, upsert, uuid.New(), characterID, p.Name, p.SkillPoints); err != nil {
			return fmt.Errorf("upsert profession %q: %w", p.Name, err)
		}
		if err := replaceRecipes(ctx, tx, characterID, p); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit professions upsert: %w", err)
	}
	return nil
}

// replaceRecipes makes the character's recipes for one profession match what Blizzard
// lists. Forever lists the profession itself ("Skinning", "Fishing") among its recipes;
// it is not something to craft, so it is left out.
func replaceRecipes(ctx context.Context, tx *sqlx.Tx, characterID uuid.UUID, p blizzard.CharacterProfession) error {
	const upsertRecipe = `
		INSERT INTO recipes (id, profession, name) VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE SET profession = EXCLUDED.profession, name = EXCLUDED.name`
	const link = `
		INSERT INTO character_recipes (character_id, recipe_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING`
	const forget = `
		DELETE FROM character_recipes
		WHERE character_id = $1
		  AND recipe_id IN (SELECT id FROM recipes WHERE profession = $2)
		  AND NOT (recipe_id = ANY($3::int[]))`

	ids := []int32{}
	for _, recipe := range p.Recipes {
		if recipe.Name == p.Name {
			continue
		}
		if _, err := tx.ExecContext(ctx, upsertRecipe, recipe.ID, p.Name, recipe.Name); err != nil {
			return fmt.Errorf("upsert recipe %q: %w", recipe.Name, err)
		}
		if _, err := tx.ExecContext(ctx, link, characterID, recipe.ID); err != nil {
			return fmt.Errorf("link recipe %q: %w", recipe.Name, err)
		}
		ids = append(ids, int32(recipe.ID))
	}
	if _, err := tx.ExecContext(ctx, forget, characterID, p.Name, ids); err != nil {
		return fmt.Errorf("forget dropped %s recipes: %w", p.Name, err)
	}
	return nil
}
