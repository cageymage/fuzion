package roster

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/cageymage/fuzion/backend/internal/blizzard"
	"github.com/cageymage/fuzion/backend/internal/synclog"
)

const blizzardSyncSource = "blizzard-roster"

// Blizzard supplies a surname but not a role, and the table requires one, so new
// characters get a placeholder role for an officer to correct; the sync never writes
// it again. The secondary name placeholder marks characters imported before the sync
// read surnames; the sync fills it in on first sight.
const (
	placeholderSecondaryName = "Unset"
	placeholderRole          = "dps"
)

type UpsertResult struct {
	Inserted int
	Updated  int
	Left     int
	// Skipped counts members the table cannot hold: an unsupported class, or a
	// name that clashes with an existing character.
	Skipped int
	// UnsupportedClasses counts skipped members by Blizzard class id, so a run shows
	// which classes the site is missing.
	UnsupportedClasses map[int]int
	// UnmappedRaces counts imported members whose race id the sync cannot name, so their
	// race is left as it was.
	UnmappedRaces map[int]int
}

// UpsertFromBlizzard makes the guild's members match Blizzard's list. Characters are
// matched on name and surname, ignoring case. For existing characters only class, level,
// race, guild rank and left_guild_at are written; role, spec, raid team and owner
// belong to officers and members. With alt ranks configured, main/alt follows the
// guild rank until an officer sets it by hand; without them every new character is a
// main and main/alt is never written. A character that was seen in an earlier sync and
// is now absent is marked as having left; one the sync has never seen is left alone.
func (r *Repo) UpsertFromBlizzard(ctx context.Context, members []blizzard.RosterMember, altRanks []int) (UpsertResult, error) {
	const setFromBlizzard = `
			class         = $3,
			level         = COALESCE($4, level),
			race          = COALESCE($6, race),
			guild_rank    = $5,
			is_main       = CASE WHEN is_main_set_by_hand THEN is_main ELSE COALESCE($7, is_main) END,
			left_guild_at = NULL`

	const update = `UPDATE characters SET ` + setFromBlizzard + `
		WHERE lower(name) = lower($1) AND lower(secondary_name) = lower($2)`

	const fillInSurname = `UPDATE characters SET secondary_name = $2, ` + setFromBlizzard + `
		WHERE id = (
			SELECT id FROM characters
			WHERE lower(name) = lower($1) AND secondary_name = '` + placeholderSecondaryName + `'
			LIMIT 1
		)`

	const insert = `
		INSERT INTO characters (id, name, secondary_name, class, role, is_main, level, guild_rank, race)
		VALUES ($1, $2, $3, $4, $5, COALESCE($9, true), $6, $7, $8)
		ON CONFLICT (name, secondary_name) DO NOTHING`

	const markLeft = `
		UPDATE characters SET left_guild_at = now()
		WHERE guild_rank IS NOT NULL
		  AND left_guild_at IS NULL
		  AND (lower(name), lower(secondary_name)) NOT IN (SELECT * FROM unnest($1::text[], $2::text[]))`

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return UpsertResult{}, fmt.Errorf("begin blizzard upsert: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op once committed

	result := UpsertResult{UnsupportedClasses: map[int]int{}, UnmappedRaces: map[int]int{}}
	seenNames := make([]string, 0, len(members))
	seenSurnames := make([]string, 0, len(members))
	for _, m := range members {
		seenNames = append(seenNames, strings.ToLower(m.Name))
		seenSurnames = append(seenSurnames, strings.ToLower(m.Surname))

		if m.Class == "" {
			result.Skipped++
			result.UnsupportedClasses[m.ClassID]++
			continue
		}

		level := validLevel(m.Level)
		race := raceOrNil(m)
		if race == nil {
			result.UnmappedRaces[m.RaceID]++
		}
		isMain := mainByRank(m.Rank, altRanks)

		found := false
		for _, query := range []string{update, fillInSurname} {
			updated, err := tx.ExecContext(ctx, query, m.Name, m.Surname, m.Class, level, m.Rank, race, isMain)
			if err != nil {
				return UpsertResult{}, fmt.Errorf("update character %s %s: %w", m.Name, m.Surname, err)
			}
			n, err := updated.RowsAffected()
			if err != nil {
				return UpsertResult{}, fmt.Errorf("count updated characters: %w", err)
			}
			if n > 0 {
				found = true
				break
			}
		}
		if found {
			result.Updated++
			continue
		}

		inserted, err := tx.ExecContext(ctx, insert, uuid.New(), m.Name, m.Surname, m.Class, placeholderRole, level, m.Rank, race, isMain)
		if err != nil {
			return UpsertResult{}, fmt.Errorf("insert character %s %s: %w", m.Name, m.Surname, err)
		}
		if n, err := inserted.RowsAffected(); err != nil {
			return UpsertResult{}, fmt.Errorf("count inserted characters: %w", err)
		} else if n > 0 {
			result.Inserted++
		} else {
			result.Skipped++
		}
	}

	left, err := tx.ExecContext(ctx, markLeft, seenNames, seenSurnames)
	if err != nil {
		return UpsertResult{}, fmt.Errorf("mark characters as left: %w", err)
	}
	n, err := left.RowsAffected()
	if err != nil {
		return UpsertResult{}, fmt.Errorf("count characters marked as left: %w", err)
	}
	result.Left = int(n)

	if err := tx.Commit(); err != nil {
		return UpsertResult{}, fmt.Errorf("commit blizzard upsert: %w", err)
	}
	return result, nil
}

// nil when no alt ranks are configured, so the sync leaves main/alt alone.
func mainByRank(rank int, altRanks []int) *bool {
	if len(altRanks) == 0 {
		return nil
	}
	isMain := !slices.Contains(altRanks, rank)
	return &isMain
}

// A level outside what the table accepts is treated as unknown rather than failing the
// whole run, since retail levels go past the cap Forever has.
func validLevel(level int) *int {
	if level < 1 || level > maxLevel {
		return nil
	}
	return &level
}

type BlizzardRoster interface {
	GuildRoster(ctx context.Context, guildSlug string) ([]blizzard.RosterMember, error)
	CharacterGender(ctx context.Context, characterSlug string) (string, error)
}

// FillMissingGenders asks Blizzard for the gender of every guild character that has none
// and stores it. It returns how many it stored and how many Blizzard has no profile for.
func (r *Repo) FillMissingGenders(ctx context.Context, blizzardAPI BlizzardRoster) (stored, notFound int, err error) {
	var missing []struct {
		Name          string `db:"name"`
		SecondaryName string `db:"secondary_name"`
	}
	if err := r.db.SelectContext(ctx, &missing, `
		SELECT name, secondary_name FROM characters
		WHERE gender IS NULL AND guild_rank IS NOT NULL AND left_guild_at IS NULL AND secondary_name <> $1
		ORDER BY name, secondary_name`, placeholderSecondaryName); err != nil {
		return 0, 0, fmt.Errorf("select characters without a gender: %w", err)
	}

	for _, c := range missing {
		slug := strings.ToLower(c.Name) + "-" + strings.ToLower(c.SecondaryName)
		gender, err := blizzardAPI.CharacterGender(ctx, slug)
		if errors.Is(err, blizzard.ErrNotFound) {
			notFound++
			continue
		}
		if err != nil {
			return stored, notFound, fmt.Errorf("ask blizzard for the gender of %s %s: %w", c.Name, c.SecondaryName, err)
		}
		if _, err := r.db.ExecContext(ctx,
			`UPDATE characters SET gender = $3 WHERE name = $1 AND secondary_name = $2`, c.Name, c.SecondaryName, gender); err != nil {
			return stored, notFound, fmt.Errorf("store gender of %s %s: %w", c.Name, c.SecondaryName, err)
		}
		stored++
	}
	return stored, notFound, nil
}

type SyncBlizzardRoster struct {
	repo      *Repo
	blizzard  BlizzardRoster
	syncLog   *synclog.Repo
	guildSlug string
	altRanks  []int
}

func NewSyncBlizzardRoster(repo *Repo, client BlizzardRoster, syncLog *synclog.Repo, guildSlug string, altRanks []int) *SyncBlizzardRoster {
	return &SyncBlizzardRoster{repo: repo, blizzard: client, syncLog: syncLog, guildSlug: guildSlug, altRanks: altRanks}
}

func (j *SyncBlizzardRoster) Run(ctx context.Context) error {
	message, err := j.sync(ctx)
	if err != nil {
		if logErr := j.syncLog.Record(ctx, blizzardSyncSource, synclog.StatusError, err.Error()); logErr != nil {
			return errors.Join(err, logErr)
		}
		return err
	}
	return j.syncLog.Record(ctx, blizzardSyncSource, synclog.StatusOK, message)
}

func (j *SyncBlizzardRoster) sync(ctx context.Context) (string, error) {
	members, err := j.blizzard.GuildRoster(ctx, j.guildSlug)
	if err != nil {
		return "", fmt.Errorf("ask blizzard for the guild roster: %w", err)
	}
	// An empty list is far more likely a bad response than a guild with no members,
	// and acting on it would mark every character as having left.
	if len(members) == 0 {
		return "", errors.New("blizzard returned an empty guild roster")
	}

	result, err := j.repo.UpsertFromBlizzard(ctx, members, j.altRanks)
	if err != nil {
		return "", fmt.Errorf("store guild roster: %w", err)
	}

	_, noProfile, err := j.repo.FillMissingGenders(ctx, j.blizzard)
	if err != nil {
		return "", fmt.Errorf("store character genders: %w", err)
	}
	genderNote := ""
	if noProfile > 0 {
		genderNote = fmt.Sprintf(", %d without a Blizzard profile", noProfile)
	}
	return fmt.Sprintf("%d members: %d added, %d updated, %d left, %d skipped%s%s%s",
		len(members), result.Inserted, result.Updated, result.Left, result.Skipped,
		describeCounts("class", result.UnsupportedClasses), describeCounts("unmapped race", result.UnmappedRaces), genderNote), nil
}

func raceOrNil(m blizzard.RosterMember) *string {
	if m.Race == "" {
		return nil
	}
	return &m.Race
}

// " (class 6 x20, 10 x12)", most common id first, or "" when there is nothing to count.
func describeCounts(label string, counts map[int]int) string {
	if len(counts) == 0 {
		return ""
	}
	ids := slices.Collect(maps.Keys(counts))
	slices.SortFunc(ids, func(a, b int) int {
		if counts[a] != counts[b] {
			return counts[b] - counts[a]
		}
		return a - b
	})
	parts := make([]string, 0, len(ids))
	for i, id := range ids {
		if i == 0 {
			parts = append(parts, fmt.Sprintf("%s %d x%d", label, id, counts[id]))
		} else {
			parts = append(parts, fmt.Sprintf("%d x%d", id, counts[id]))
		}
	}
	return " (" + strings.Join(parts, ", ") + ")"
}
