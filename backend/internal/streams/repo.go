package streams

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Stream struct {
	ID           uuid.UUID `db:"id"            json:"id"`
	StreamerName string    `db:"streamer_name" json:"streamerName"`
	GameName     string    `db:"game_name"     json:"gameName"`
	ViewerCount  int       `db:"viewer_count"  json:"viewerCount"`
	ThumbnailURL *string   `db:"thumbnail_url" json:"thumbnailUrl"`
	ChannelURL   string    `db:"channel_url"   json:"channelUrl"`
}

type Repo struct {
	db *sqlx.DB
}

func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) ListLive(ctx context.Context) ([]Stream, error) {
	const query = `
		SELECT id, streamer_name, game_name, viewer_count, thumbnail_url, channel_url
		FROM streams
		WHERE is_live
		ORDER BY viewer_count DESC`

	live := []Stream{}
	if err := r.db.SelectContext(ctx, &live, query); err != nil {
		return nil, fmt.Errorf("select live streams: %w", err)
	}
	return live, nil
}
