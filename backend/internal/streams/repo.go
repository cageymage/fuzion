package streams

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/cageymage/fuzion/backend/internal/twitch"
	"github.com/cageymage/fuzion/backend/internal/youtube"
)

type Stream struct {
	ID              uuid.UUID `db:"id"            json:"id"`
	StreamerName    string    `db:"streamer_name" json:"streamerName"`
	GameName        string    `db:"game_name"     json:"gameName"`
	Title           string    `db:"stream_title"  json:"title"`
	ViewerCount     int       `db:"viewer_count"  json:"viewerCount"`
	ThumbnailURL    *string   `db:"thumbnail_url" json:"thumbnailUrl"`
	AvatarURL       *string   `db:"profile_image_url" json:"avatarUrl"`
	ChannelURL      string    `db:"channel_url"   json:"channelUrl"`
	IsLive          bool      `db:"is_live"       json:"isLive"`
	IsLiveOtherGame bool      `db:"is_live_other_game" json:"isLiveOtherGame"`
	Platform        string    `db:"platform"      json:"platform"`
}

// Rows without a youtube_channel_id are Twitch channels, including hand-entered
// ones that predate the YouTube sync.
const selectStream = `
	SELECT id, streamer_name, game_name, stream_title, viewer_count, thumbnail_url, profile_image_url, channel_url, is_live, is_live_other_game,
		CASE WHEN youtube_channel_id IS NOT NULL THEN 'youtube' ELSE 'twitch' END AS platform
	FROM streams`

// The column each platform's sync matches its channel IDs against. Only these
// constants are ever interpolated into SQL.
const (
	twitchColumn  = "twitch_login"
	youTubeColumn = "youtube_channel_id"
)

type liveDetails struct {
	ViewerCount  int
	GameName     string
	ThumbnailURL string
	Title        string
	InGuildGame  bool
}

type Repo struct {
	db *sqlx.DB
}

func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) ListLive(ctx context.Context) ([]Stream, error) {
	const query = selectStream + `
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

func (r *Repo) YouTubeChannelIDs(ctx context.Context) ([]string, error) {
	ids := []string{}
	if err := r.db.SelectContext(ctx, &ids, `SELECT youtube_channel_id FROM streams WHERE youtube_channel_id IS NOT NULL ORDER BY youtube_channel_id`); err != nil {
		return nil, fmt.Errorf("select youtube channel ids: %w", err)
	}
	return ids, nil
}

// Only rows with a twitch_login are touched: YouTube and hand-entered channels are not
// managed by the Twitch sync.
func (r *Repo) UpdateLiveStatus(ctx context.Context, live map[string]twitch.LiveStream) error {
	details := make(map[string]liveDetails, len(live))
	for login, s := range live {
		details[login] = liveDetails{ViewerCount: s.ViewerCount, GameName: s.GameName, ThumbnailURL: s.ThumbnailURL, Title: s.Title, InGuildGame: s.GameID == twitch.WorldOfWarcraftGameID}
	}
	return r.updateLiveStatus(ctx, twitchColumn, details)
}

// Only rows with a youtube_channel_id are touched. YouTube has no game category, so
// every live stream counts as in the guild's game and the game name is left blank.
func (r *Repo) UpdateYouTubeLiveStatus(ctx context.Context, live map[string]youtube.LiveStream) error {
	details := make(map[string]liveDetails, len(live))
	for channelID, s := range live {
		details[channelID] = liveDetails{ViewerCount: s.ViewerCount, ThumbnailURL: s.ThumbnailURL, Title: s.Title, InGuildGame: true}
	}
	return r.updateLiveStatus(ctx, youTubeColumn, details)
}

func (r *Repo) updateLiveStatus(ctx context.Context, column string, live map[string]liveDetails) (err error) {
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

	liveIDs := make([]string, 0, len(live))
	for id := range live {
		liveIDs = append(liveIDs, id)
	}

	markOffline := fmt.Sprintf(`
		UPDATE streams SET is_live = false, is_live_other_game = false, viewer_count = 0, stream_title = ''
		WHERE %[1]s IS NOT NULL AND NOT (%[1]s = ANY($1))`, column)
	if _, err := tx.ExecContext(ctx, markOffline, liveIDs); err != nil {
		return fmt.Errorf("mark unreturned channels offline: %w", err)
	}

	markLive := fmt.Sprintf(`
		UPDATE streams SET is_live = $6, is_live_other_game = NOT $6, viewer_count = $2, game_name = $3, thumbnail_url = $4, stream_title = $5
		WHERE %s = $1`, column)
	for id, d := range live {
		if _, err := tx.ExecContext(ctx, markLive, id, d.ViewerCount, d.GameName, d.ThumbnailURL, d.Title, d.InGuildGame); err != nil {
			return fmt.Errorf("mark %s live: %w", id, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit live status update: %w", err)
	}
	return nil
}

func (r *Repo) UpdateProfileImages(ctx context.Context, avatars map[string]string) error {
	return r.updateAvatars(ctx, twitchColumn, avatars)
}

func (r *Repo) UpdateYouTubeAvatars(ctx context.Context, avatars map[string]string) error {
	return r.updateAvatars(ctx, youTubeColumn, avatars)
}

func (r *Repo) updateAvatars(ctx context.Context, column string, avatars map[string]string) error {
	ids := make([]string, 0, len(avatars))
	urls := make([]string, 0, len(avatars))
	for id, url := range avatars {
		ids = append(ids, id)
		urls = append(urls, url)
	}

	query := fmt.Sprintf(`
		UPDATE streams SET profile_image_url = v.url
		FROM unnest($1::text[], $2::text[]) AS v(id, url)
		WHERE streams.%s = v.id`, column)
	if _, err := r.db.ExecContext(ctx, query, ids, urls); err != nil {
		return fmt.Errorf("update profile images: %w", err)
	}
	return nil
}

func (r *Repo) ListAll(ctx context.Context) ([]Stream, error) {
	const query = selectStream + `
		ORDER BY is_live DESC, streamer_name ASC`

	all := []Stream{}
	if err := r.db.SelectContext(ctx, &all, query); err != nil {
		return nil, fmt.Errorf("select all streams: %w", err)
	}
	return all, nil
}
