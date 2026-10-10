package roster

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/cageymage/fuzion/backend/internal/blizzard"
	"github.com/cageymage/fuzion/backend/internal/synclog"
)

const blizzardSyncSource = "blizzard-roster"

// Blizzard's slug for a realm: lowercase, spaces to hyphens, apostrophes dropped.
const realmSlugSQL = `lower(replace(replace(realm, ' ', '-'), '''', ''))`

// Blizzard knows only one name per character, but the table requires a secondary name
// and a role, so new characters get placeholders for an officer to correct; the sync
// never writes those columns again.
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
// matched on name and realm, ignoring case. For existing characters only class, level,
// race, guild rank and left_guild_at are written; role, spec, main/alt, raid team and owner
// belong to officers and members. A character that was seen in an earlier sync and is
// now absent is marked as having left; one the sync has never seen is left alone.
func (r *Repo) UpsertFromBlizzard(ctx context.Context, members []blizzard.RosterMember) (UpsertResult, error) {
	const update = `
		UPDATE characters SET
			class         = $3,
			level         = COALESCE($4, level),
			race          = COALESCE($6, race),
			guild_rank    = $5,
			left_guild_at = NULL
		WHERE lower(name) = lower($1) AND ` + realmSlugSQL + ` = $2`

	const insert = `
		INSERT INTO characters (id, name, secondary_name, realm, class, role, is_main, level, guild_rank, race)
		VALUES ($1, $2, $3, $4, $5, $6, false, $7, $8, $9)
		ON CONFLICT (name, secondary_name) DO NOTHING`

	const markLeft = `
		UPDATE characters SET left_guild_at = now()
		WHERE guild_rank IS NOT NULL
		  AND left_guild_at IS NULL
		  AND (lower(name), ` + realmSlugSQL + `) NOT IN (SELECT * FROM unnest($1::text[], $2::text[]))`

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return UpsertResult{}, fmt.Errorf("begin blizzard upsert: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op once committed

	result := UpsertResult{UnsupportedClasses: map[int]int{}, UnmappedRaces: map[int]int{}}
	seenNames := make([]string, 0, len(members))
	seenRealms := make([]string, 0, len(members))
	for _, m := range members {
		slug := strings.ToLower(m.RealmSlug)
		seenNames = append(seenNames, strings.ToLower(m.Name))
		seenRealms = append(seenRealms, slug)

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
		updated, err := tx.ExecContext(ctx, update, m.Name, slug, m.Class, level, m.Rank, race)
		if err != nil {
			return UpsertResult{}, fmt.Errorf("update character %s: %w", m.Name, err)
		}
		if n, err := updated.RowsAffected(); err != nil {
			return UpsertResult{}, fmt.Errorf("count updated characters: %w", err)
		} else if n > 0 {
			result.Updated++
			continue
		}

		inserted, err := tx.ExecContext(ctx, insert, uuid.New(), m.Name, placeholderSecondaryName, realmDisplayName(slug), m.Class, placeholderRole, level, m.Rank, race)
		if err != nil {
			return UpsertResult{}, fmt.Errorf("insert character %s: %w", m.Name, err)
		}
		if n, err := inserted.RowsAffected(); err != nil {
			return UpsertResult{}, fmt.Errorf("count inserted characters: %w", err)
		} else if n > 0 {
			result.Inserted++
		} else {
			result.Skipped++
		}
	}

	left, err := tx.ExecContext(ctx, markLeft, seenNames, seenRealms)
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

// A level outside what the table accepts is treated as unknown rather than failing the
// whole run, since retail levels go past the cap Forever has.
func validLevel(level int) *int {
	if level < 1 || level > maxLevel {
		return nil
	}
	return &level
}

// "area-52" becomes "Area 52". The slug has lost any apostrophe, so a realm such as
// Mal'Ganis comes out as "Malganis"; officers can correct the stored realm.
func realmDisplayName(slug string) string {
	words := strings.Split(slug, "-")
	for i, word := range words {
		first, size := utf8.DecodeRuneInString(word)
		words[i] = string(unicode.ToUpper(first)) + word[size:]
	}
	return strings.Join(words, " ")
}

type BlizzardRoster interface {
	GuildRoster(ctx context.Context, realmSlug, guildSlug string) ([]blizzard.RosterMember, error)
}

type SyncBlizzardRoster struct {
	repo      *Repo
	blizzard  BlizzardRoster
	syncLog   *synclog.Repo
	realmSlug string
	guildSlug string
}

func NewSyncBlizzardRoster(repo *Repo, client BlizzardRoster, syncLog *synclog.Repo, realmSlug, guildSlug string) *SyncBlizzardRoster {
	return &SyncBlizzardRoster{repo: repo, blizzard: client, syncLog: syncLog, realmSlug: realmSlug, guildSlug: guildSlug}
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
	members, err := j.blizzard.GuildRoster(ctx, j.realmSlug, j.guildSlug)
	if err != nil {
		return "", fmt.Errorf("ask blizzard for the guild roster: %w", err)
	}
	// An empty list is far more likely a bad response than a guild with no members,
	// and acting on it would mark every character as having left.
	if len(members) == 0 {
		return "", errors.New("blizzard returned an empty guild roster")
	}

	result, err := j.repo.UpsertFromBlizzard(ctx, members)
	if err != nil {
		return "", fmt.Errorf("store guild roster: %w", err)
	}
	return fmt.Sprintf("%d members: %d added, %d updated, %d left, %d skipped%s%s",
		len(members), result.Inserted, result.Updated, result.Left, result.Skipped,
		describeCounts("class", result.UnsupportedClasses), describeCounts("unmapped race", result.UnmappedRaces)), nil
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
