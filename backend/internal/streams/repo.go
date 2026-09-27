package streams

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/cageymage/fuzion/backend/internal/twitch"
)

type Stream struct {
	ID           uuid.UUID `db:"id"            json:"id"`
	StreamerName string    `db:"streamer_name" json:"streamerName"`
	GameName     string    `db:"game_name"     json:"gameName"`
	Title        string    `db:"stream_title"  json:"title"`
	ViewerCount  int       `db:"viewer_count"  json:"viewerCount"`
	ThumbnailURL *string   `db:"thumbnail_url" json:"thumbnailUrl"`
	AvatarURL    *string   `db:"profile_image_url" json:"avatarUrl"`
	ChannelURL   string    `db:"channel_url"   json:"channelUrl"`
	IsLive       bool      `db:"is_live"       json:"isLive"`
}

type Repo struct {
	db *sqlx.DB
}

func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) ListLive(ctx context.Context) ([]Stream, error) {
	const query = `
		SELECT id, streamer_name, game_name, stream_title, viewer_count, thumbnail_url, profile_image_url, channel_url, is_live
		FROM streams
		WHERE is_live
		ORDER BY viewer_count DESC`

	live := []Stream{}
	if err := r.db.SelectContext(ctx, &live, query); err != nil {
		return nil, fmt.Errorf("select live streams: %w", err)
	}
	return live, nil
}

func (r *Repo) LinkedLogins(ctx context.Context) ([]string, error) {
	logins := []string{}
	if err := r.db.SelectContext(ctx, &logins, `SELECT twitch_login FROM streams WHERE twitch_login IS NOT NULL ORDER BY twitch_login`); err != nil {
		return nil, fmt.Errorf("select twitch logins: %w", err)
	}
	return logins, nil
}

// Channels without a twitch_login are left alone: they are not managed by the sync.
func (r *Repo) UpdateLiveStatus(ctx context.Context, live map[string]twitch.LiveStream) (err error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin live status update: %w", err)
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				err = errors.Join(err, fmt.Errorf("rollback live status update: %w", rollbackErr))
			}
		}
	}()

	liveLogins := make([]string, 0, len(live))
	for login := range live {
		liveLogins = append(liveLogins, login)
	}

	const markOffline = `
		UPDATE streams SET is_live = false, viewer_count = 0, stream_title = ''
		WHERE twitch_login IS NOT NULL AND NOT (twitch_login = ANY($1))`
	if _, err := tx.ExecContext(ctx, markOffline, liveLogins); err != nil {
		return fmt.Errorf("mark unreturned channels offline: %w", err)
	}

	const markLive = `
		UPDATE streams SET is_live = true, viewer_count = $2, game_name = $3, thumbnail_url = $4, stream_title = $5
		WHERE twitch_login = $1`
	for login, stream := range live {
		if _, err := tx.ExecContext(ctx, markLive, login, stream.ViewerCount, stream.GameName, stream.ThumbnailURL, stream.Title); err != nil {
			return fmt.Errorf("mark %s live: %w", login, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit live status update: %w", err)
	}
	return nil
}

func (r *Repo) UpdateProfileImages(ctx context.Context, avatars map[string]string) error {
	logins := make([]string, 0, len(avatars))
	urls := make([]string, 0, len(avatars))
	for login, url := range avatars {
		logins = append(logins, login)
		urls = append(urls, url)
	}

	const query = `
		UPDATE streams SET profile_image_url = v.url
		FROM unnest($1::text[], $2::text[]) AS v(login, url)
		WHERE streams.twitch_login = v.login`
	if _, err := r.db.ExecContext(ctx, query, logins, urls); err != nil {
		return fmt.Errorf("update profile images: %w", err)
	}
	return nil
}

func (r *Repo) ListAll(ctx context.Context) ([]Stream, error) {
	const query = `
		SELECT id, streamer_name, game_name, stream_title, viewer_count, thumbnail_url, profile_image_url, channel_url, is_live
		FROM streams
		ORDER BY is_live DESC, streamer_name ASC`

	all := []Stream{}
	if err := r.db.SelectContext(ctx, &all, query); err != nil {
		return nil, fmt.Errorf("select all streams: %w", err)
	}
	return all, nil
}
