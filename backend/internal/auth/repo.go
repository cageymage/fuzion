package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

var (
	ErrSessionNotFound        = errors.New("session not found")
	ErrUserNotFound           = errors.New("user not found")
	ErrBattleNetAlreadyLinked = errors.New("battle.net account is already linked to another user")
)

const uniqueViolationCode = "23505"

type User struct {
	ID              uuid.UUID `db:"id"`
	DiscordID       string    `db:"discord_id"`
	Username        string    `db:"username"`
	AvatarURL       *string   `db:"avatar_url"`
	BattleNetID     *string   `db:"battlenet_id"`
	BattleNetTag    *string   `db:"battlenet_tag"`
	IsAdmin         bool      `db:"is_admin"`
	IsOfficer       bool      `db:"is_officer"`
	InDiscordServer bool      `db:"in_discord_server"`
	CreatedAt       time.Time `db:"created_at"`
	LastSeenAt      time.Time `db:"last_seen_at"`

	// IsGuildMember is only resolved for session users; see WithDiscordServerMembership.
	IsGuildMember bool `db:"-"`
}

// HasOfficerAccess is the effective officer permission: admins get
// officer-level access regardless of their own IsOfficer grant.
func (u User) HasOfficerAccess() bool {
	return u.IsAdmin || u.IsOfficer
}

type Repo struct {
	db *sqlx.DB
}

func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

// UpsertUserByDiscordID inserts or refreshes a user from their Discord identity.
// The first account ever created becomes the admin, because nothing else can
// grant the first one. The conflict branch never writes is_admin, so a repeat
// login can neither re-trigger that nor demote an admin granted in the panel.
// The bool reports that this call was what granted that first admin.
func (r *Repo) UpsertUserByDiscordID(ctx context.Context, identity Identity) (User, bool, error) {
	// xmax is zero only on a row this statement inserted, which is how an
	// upsert distinguishes a brand new account from a returning login.
	const query = `
		INSERT INTO users (id, discord_id, username, avatar_url, in_discord_server, is_admin)
		VALUES ($1, $2, $3, $4, $5, NOT EXISTS (SELECT 1 FROM users))
		ON CONFLICT (discord_id) DO UPDATE
			SET username = EXCLUDED.username,
			    avatar_url = EXCLUDED.avatar_url,
			    in_discord_server = EXCLUDED.in_discord_server,
			    last_seen_at = now()
		RETURNING id, discord_id, username, avatar_url, battlenet_id, battlenet_tag, is_admin, is_officer, in_discord_server, created_at, last_seen_at,
		          (xmax = 0 AND is_admin) AS granted_first_admin`

	var avatarURL *string
	if identity.AvatarURL != "" {
		avatarURL = &identity.AvatarURL
	}

	var row struct {
		User
		GrantedFirstAdmin bool `db:"granted_first_admin"`
	}
	if err := r.db.GetContext(ctx, &row, query, uuid.New(), identity.ProviderUserID, identity.Username, avatarURL, identity.InDiscordServer); err != nil {
		return User{}, false, fmt.Errorf("upsert user by discord id: %w", err)
	}
	return row.User.utc(), row.GrantedFirstAdmin, nil
}

func (r *Repo) ListUsers(ctx context.Context) ([]User, error) {
	const query = `
		SELECT id, discord_id, username, avatar_url, battlenet_id, battlenet_tag, is_admin, is_officer, created_at, last_seen_at
		FROM users
		ORDER BY username`

	var users []User
	if err := r.db.SelectContext(ctx, &users, query); err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	for i := range users {
		users[i] = users[i].utc()
	}
	return users, nil
}

func (r *Repo) UpdateUserRoles(ctx context.Context, id uuid.UUID, isOfficer, isAdmin bool) (User, error) {
	const query = `
		UPDATE users
		SET is_officer = $2, is_admin = $3
		WHERE id = $1
		RETURNING id, discord_id, username, avatar_url, battlenet_id, battlenet_tag, is_admin, is_officer, created_at, last_seen_at`

	var user User
	err := r.db.GetContext(ctx, &user, query, id, isOfficer, isAdmin)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("update user roles: %w", err)
	}
	return user.utc(), nil
}

// The roster sync's realm slug for a stored realm: lowercase, spaces to hyphens,
// apostrophes dropped, so a link matches characters the same way the sync does.
const characterRealmSlugSQL = `lower(replace(replace(realm, ' ', '-'), '''', ''))`

// LinkBattleNet stores the Battle.net identity and makes the user the owner of the
// roster characters on that account, matched on name and realm, ignoring case.
// Characters another member already owns stay theirs. Characters this user owned
// that are no longer on the account are released, so linking again refreshes the set.
func (r *Repo) LinkBattleNet(ctx context.Context, userID uuid.UUID, identity Identity) error {
	const link = `
		UPDATE users
		SET battlenet_id = $2, battlenet_tag = $3
		WHERE id = $1`

	const release = `
		UPDATE characters SET user_id = NULL
		WHERE user_id = $1
		  AND (lower(name), ` + characterRealmSlugSQL + `) NOT IN (SELECT * FROM unnest($2::text[], $3::text[]))`

	const claim = `
		UPDATE characters SET user_id = $1
		WHERE user_id IS NULL
		  AND (lower(name), ` + characterRealmSlugSQL + `) IN (SELECT * FROM unnest($2::text[], $3::text[]))`

	names := make([]string, 0, len(identity.Characters))
	realms := make([]string, 0, len(identity.Characters))
	for _, c := range identity.Characters {
		names = append(names, strings.ToLower(c.Name))
		realms = append(realms, strings.ToLower(c.RealmSlug))
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin link battle.net: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op once committed

	_, err = tx.ExecContext(ctx, link, userID, identity.ProviderUserID, identity.Username)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
		return ErrBattleNetAlreadyLinked
	}
	if err != nil {
		return fmt.Errorf("link battle.net: %w", err)
	}
	if _, err := tx.ExecContext(ctx, release, userID, names, realms); err != nil {
		return fmt.Errorf("release characters no longer on the account: %w", err)
	}
	if _, err := tx.ExecContext(ctx, claim, userID, names, realms); err != nil {
		return fmt.Errorf("claim characters: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit link battle.net: %w", err)
	}
	return nil
}

// UnlinkBattleNet clears the identity and releases every character the user owns,
// since ownership only ever came from the link.
func (r *Repo) UnlinkBattleNet(ctx context.Context, userID uuid.UUID) error {
	const unlink = `
		UPDATE users
		SET battlenet_id = NULL, battlenet_tag = NULL
		WHERE id = $1`

	const release = `UPDATE characters SET user_id = NULL WHERE user_id = $1`

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin unlink battle.net: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op once committed

	if _, err := tx.ExecContext(ctx, unlink, userID); err != nil {
		return fmt.Errorf("unlink battle.net: %w", err)
	}
	if _, err := tx.ExecContext(ctx, release, userID); err != nil {
		return fmt.Errorf("release characters: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit unlink battle.net: %w", err)
	}
	return nil
}

func (r *Repo) CreateSession(ctx context.Context, token string, userID uuid.UUID, expiresAt time.Time) error {
	const query = `INSERT INTO sessions (token, user_id, expires_at) VALUES ($1, $2, $3)`
	if _, err := r.db.ExecContext(ctx, query, token, userID, expiresAt); err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

func (r *Repo) FindSession(ctx context.Context, token string) (User, error) {
	const query = `
		SELECT u.id, u.discord_id, u.username, u.avatar_url, u.battlenet_id, u.battlenet_tag, u.is_admin, u.is_officer, u.in_discord_server, u.created_at, u.last_seen_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token = $1 AND s.expires_at > now()`

	var user User
	err := r.db.GetContext(ctx, &user, query, token)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrSessionNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("select session: %w", err)
	}
	return user.utc(), nil
}

func (r *Repo) DeleteSession(ctx context.Context, token string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE token = $1`, token); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// pgx hands back timestamptz in the process-local zone; the API always emits UTC.
func (u User) utc() User {
	u.CreatedAt = u.CreatedAt.UTC()
	u.LastSeenAt = u.LastSeenAt.UTC()
	return u
}
