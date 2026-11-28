package roster

import (
	"context"
	"database/sql"
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

// The secondary name placeholder marks characters imported before the sync read
// surnames; the sync fills it in on first sight.
const placeholderSecondaryName = "Unset"

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
	// RankRequestsClosed counts open rank change requests the run found already carried
	// out in game.
	RankRequestsClosed int
	// Renamed counts characters Blizzard renamed, matched by their Blizzard ID.
	Renamed int
	// RenamesSkipped names renames that were not applied because another character already
	// has the new name, as "Old Name to New Name", for an officer to sort out.
	RenamesSkipped []string
}

// UpsertFromBlizzard makes the guild's members match Blizzard's list. Every member is
// first matched on Blizzard's character ID, which is expected to survive a rename, so a
// renamed character keeps its row and takes the new name. Members whose ID matches no row
// are then matched on name and surname, ignoring case, which stores or replaces the ID.
// For existing characters only class, level, race, guild rank and left_guild_at are written; role, spec, raid team and owner
// belong to officers and members, and new characters start with no role or spec for
// their owner to choose. Main/alt always follows the guild rank, so a rank change in
// game moves the character and closes an open rank change request it carries out. A
// character that was seen in an earlier sync and is now absent is marked as having
// left; one the sync has never seen is left alone.
func (r *Repo) UpsertFromBlizzard(ctx context.Context, members []blizzard.RosterMember) (UpsertResult, error) {
	const setFromBlizzard = `
			class         = $3,
			level         = COALESCE($4, level),
			race          = COALESCE($6, race),
			guild_rank    = $5,
			is_main       = $7,
			left_guild_at = NULL`

	const findByBlizzardID = `SELECT id, name, secondary_name FROM characters WHERE blizzard_id = $1`

	const nameTakenByAnother = `
		SELECT EXISTS (
			SELECT 1 FROM characters
			WHERE id <> $1 AND lower(name) = lower($2) AND lower(secondary_name) = lower($3)
		)`

	const updateByID = `UPDATE characters SET name = $1, secondary_name = $2, ` + setFromBlizzard + `
		WHERE id = $8`

	// A row whose stored ID belongs to someone in this roster is that member's character,
	// so a name match must not take it over. Updating one row, preferring the exact
	// casing, keeps two rows that differ only in case from both claiming the same ID.
	const notClaimedByRoster = `(blizzard_id IS NULL OR NOT (blizzard_id = ANY($9::bigint[])))`

	const update = `UPDATE characters SET blizzard_id = COALESCE($8, blizzard_id), ` + setFromBlizzard + `
		WHERE id = (
			SELECT id FROM characters
			WHERE lower(name) = lower($1) AND lower(secondary_name) = lower($2) AND ` + notClaimedByRoster + `
			ORDER BY (name = $1 AND secondary_name = $2) DESC
			LIMIT 1
		)`

	const fillInSurname = `UPDATE characters SET secondary_name = $2, blizzard_id = COALESCE($8, blizzard_id), ` + setFromBlizzard + `
		WHERE id = (
			SELECT id FROM characters
			WHERE lower(name) = lower($1) AND secondary_name = '` + placeholderSecondaryName + `' AND ` + notClaimedByRoster + `
			LIMIT 1
		)`

	const insert = `
		INSERT INTO characters (id, name, secondary_name, class, is_main, level, guild_rank, race, blizzard_id)
		VALUES ($1, $2, $3, $4, $8, $5, $6, $7, $9)
		ON CONFLICT (name, secondary_name) DO NOTHING`

	const closeCarriedOutRequests = `
		UPDATE rank_change_requests r SET status = 'done', resolved_at = now()
		FROM characters c
		WHERE r.character_id = c.id AND r.status = 'open'
		  AND c.guild_rank IS NOT NULL
		  AND c.is_main = (r.direction = 'to_main')`

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
	rosterIDs := make([]int64, 0, len(members))
	for _, m := range members {
		if m.ID != 0 {
			rosterIDs = append(rosterIDs, m.ID)
		}
	}

	type unmatchedMember struct {
		blizzard.RosterMember
		level      *int
		race       *string
		isMain     bool
		blizzardID *int64
	}
	var byName []unmatchedMember

	// IDs are matched for the whole list before any name, so a renamed character has
	// already moved off its old name when a new member with that name is looked up.
	for _, m := range members {
		seenNames = append(seenNames, strings.ToLower(m.Name))
		seenSurnames = append(seenSurnames, strings.ToLower(m.Surname))

		if m.Class == "" {
			result.Skipped++
			result.UnsupportedClasses[m.ClassID]++
			continue
		}

		member := unmatchedMember{RosterMember: m, level: validLevel(m.Level), race: raceOrNil(m), isMain: !isAltRank(m.Rank), blizzardID: blizzardIDOrNil(m.ID)}
		if member.race == nil {
			result.UnmappedRaces[m.RaceID]++
		}
		if member.blizzardID == nil {
			byName = append(byName, member)
			continue
		}

		var stored struct {
			ID            uuid.UUID `db:"id"`
			Name          string    `db:"name"`
			SecondaryName string    `db:"secondary_name"`
		}
		err := tx.GetContext(ctx, &stored, findByBlizzardID, *member.blizzardID)
		if errors.Is(err, sql.ErrNoRows) {
			byName = append(byName, member)
			continue
		}
		if err != nil {
			return UpsertResult{}, fmt.Errorf("find character with blizzard id %d: %w", m.ID, err)
		}

		name, surname := stored.Name, stored.SecondaryName
		if !strings.EqualFold(name, m.Name) || !strings.EqualFold(surname, m.Surname) {
			var taken bool
			if err := tx.GetContext(ctx, &taken, nameTakenByAnother, stored.ID, m.Name, m.Surname); err != nil {
				return UpsertResult{}, fmt.Errorf("check new name %s %s: %w", m.Name, m.Surname, err)
			}
			if taken {
				result.RenamesSkipped = append(result.RenamesSkipped, fmt.Sprintf("%s %s to %s %s", name, surname, m.Name, m.Surname))
				seenNames = append(seenNames, strings.ToLower(name))
				seenSurnames = append(seenSurnames, strings.ToLower(surname))
			} else {
				name, surname = m.Name, m.Surname
				result.Renamed++
			}
		}
		if _, err := tx.ExecContext(ctx, updateByID, name, surname, m.Class, member.level, m.Rank, member.race, member.isMain, stored.ID); err != nil {
			return UpsertResult{}, fmt.Errorf("update character with blizzard id %d: %w", m.ID, err)
		}
		result.Updated++
	}

	for _, m := range byName {
		found := false
		for _, query := range []string{update, fillInSurname} {
			updated, err := tx.ExecContext(ctx, query, m.Name, m.Surname, m.Class, m.level, m.Rank, m.race, m.isMain, m.blizzardID, rosterIDs)
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

		inserted, err := tx.ExecContext(ctx, insert, uuid.New(), m.Name, m.Surname, m.Class, m.level, m.Rank, m.race, m.isMain, m.blizzardID)
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

	closed, err := tx.ExecContext(ctx, closeCarriedOutRequests)
	if err != nil {
		return UpsertResult{}, fmt.Errorf("close carried out rank requests: %w", err)
	}
	n, err = closed.RowsAffected()
	if err != nil {
		return UpsertResult{}, fmt.Errorf("count closed rank requests: %w", err)
	}
	result.RankRequestsClosed = int(n)

	if err := tx.Commit(); err != nil {
		return UpsertResult{}, fmt.Errorf("commit blizzard upsert: %w", err)
	}
	return result, nil
}

// A zero ID means Blizzard sent none, which must not be stored: the column is unique.
func blizzardIDOrNil(id int64) *int64 {
	if id == 0 {
		return nil
	}
	return &id
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
}

func NewSyncBlizzardRoster(repo *Repo, client BlizzardRoster, syncLog *synclog.Repo, guildSlug string) *SyncBlizzardRoster {
	return &SyncBlizzardRoster{repo: repo, blizzard: client, syncLog: syncLog, guildSlug: guildSlug}
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

	result, err := j.repo.UpsertFromBlizzard(ctx, members)
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
	requestsNote := ""
	if result.RankRequestsClosed > 0 {
		requestsNote = fmt.Sprintf(", %d rank requests closed", result.RankRequestsClosed)
	}
	renamesNote := ""
	if result.Renamed > 0 {
		renamesNote = fmt.Sprintf(", %d renamed", result.Renamed)
	}
	if skipped := len(result.RenamesSkipped); skipped > 0 {
		noun := "renames"
		if skipped == 1 {
			noun = "rename"
		}
		renamesNote += fmt.Sprintf(", %d %s skipped (%s)", skipped, noun, strings.Join(result.RenamesSkipped, ", "))
	}
	return fmt.Sprintf("%d members: %d added, %d updated, %d left, %d skipped%s%s%s%s%s",
		len(members), result.Inserted, result.Updated, result.Left, result.Skipped,
		describeCounts("class", result.UnsupportedClasses), describeCounts("unmapped race", result.UnmappedRaces), genderNote, requestsNote, renamesNote), nil
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
