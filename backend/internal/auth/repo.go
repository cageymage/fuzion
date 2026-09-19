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

var ErrSessionNotFound = errors.New("session not found")

type User struct {
	ID         uuid.UUID `db:"id"`
	DiscordID  string    `db:"discord_id"`
	Username   string    `db:"username"`
	AvatarURL  *string   `db:"avatar_url"`
	CreatedAt  time.Time `db:"created_at"`
	LastSeenAt time.Time `db:"last_seen_at"`
}

type Repo struct {
	db *sqlx.DB
}

func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) UpsertUserByDiscordID(ctx context.Context, identity Identity) (User, error) {
	const query = `
		INSERT INTO users (id, discord_id, username, avatar_url)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (discord_id) DO UPDATE
			SET username = EXCLUDED.username,
			    avatar_url = EXCLUDED.avatar_url,
			    last_seen_at = now()
		RETURNING id, discord_id, username, avatar_url, created_at, last_seen_at`

	var avatarURL *string
	if identity.AvatarURL != "" {
		avatarURL = &identity.AvatarURL
	}

	var user User
	if err := r.db.GetContext(ctx, &user, query, uuid.New(), identity.ProviderUserID, identity.Username, avatarURL); err != nil {
		return User{}, fmt.Errorf("upsert user by discord id: %w", err)
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
		SELECT u.id, u.discord_id, u.username, u.avatar_url, u.created_at, u.last_seen_at
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
