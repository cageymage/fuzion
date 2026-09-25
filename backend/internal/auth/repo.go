package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrUserNotFound    = errors.New("user not found")
)

type User struct {
	ID         uuid.UUID `db:"id"`
	DiscordID  string    `db:"discord_id"`
	Username   string    `db:"username"`
	AvatarURL  *string   `db:"avatar_url"`
	IsAdmin    bool      `db:"is_admin"`
	IsOfficer  bool      `db:"is_officer"`
	CreatedAt  time.Time `db:"created_at"`
	LastSeenAt time.Time `db:"last_seen_at"`
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
		INSERT INTO users (id, discord_id, username, avatar_url, is_admin)
		VALUES ($1, $2, $3, $4, NOT EXISTS (SELECT 1 FROM users))
		ON CONFLICT (discord_id) DO UPDATE
			SET username = EXCLUDED.username,
			    avatar_url = EXCLUDED.avatar_url,
			    last_seen_at = now()
		RETURNING id, discord_id, username, avatar_url, is_admin, is_officer, created_at, last_seen_at,
		          (xmax = 0 AND is_admin) AS granted_first_admin`

	var avatarURL *string
	if identity.AvatarURL != "" {
		avatarURL = &identity.AvatarURL
	}

	var row struct {
		User
		GrantedFirstAdmin bool `db:"granted_first_admin"`
	}
	if err := r.db.GetContext(ctx, &row, query, uuid.New(), identity.ProviderUserID, identity.Username, avatarURL); err != nil {
		return User{}, false, fmt.Errorf("upsert user by discord id: %w", err)
	}
	return row.User.utc(), row.GrantedFirstAdmin, nil
}

func (r *Repo) ListUsers(ctx context.Context) ([]User, error) {
	const query = `
		SELECT id, discord_id, username, avatar_url, is_admin, is_officer, created_at, last_seen_at
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
		RETURNING id, discord_id, username, avatar_url, is_admin, is_officer, created_at, last_seen_at`

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

func (r *Repo) CreateSession(ctx context.Context, token string, userID uuid.UUID, expiresAt time.Time) error {
	const query = `INSERT INTO sessions (token, user_id, expires_at) VALUES ($1, $2, $3)`
	if _, err := r.db.ExecContext(ctx, query, token, userID, expiresAt); err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

func (r *Repo) FindSession(ctx context.Context, token string) (User, error) {
	const query = `
		SELECT u.id, u.discord_id, u.username, u.avatar_url, u.is_admin, u.is_officer, u.created_at, u.last_seen_at
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
