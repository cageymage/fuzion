package streams

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/cageymage/fuzion/backend/internal/clock"
	"github.com/cageymage/fuzion/backend/internal/youtube"
)

// Playlists change rarely and the Data API is quota-limited, so one fetch serves
// every visitor for this long.
const SuggestedVideoCacheTTL = time.Hour

type PlaylistSource interface {
	PlaylistVideos(ctx context.Context, playlistID string) ([]youtube.Video, error)
}

// SuggestedVideos picks a random video from a curated playlist. With an empty
// playlistID it is disabled and never calls the source.
type SuggestedVideos struct {
	source     PlaylistSource
	playlistID string
	clock      clock.Clock

	mu        sync.Mutex
	videos    []youtube.Video
	fetchedAt time.Time
}

func NewSuggestedVideos(source PlaylistSource, playlistID string, clk clock.Clock) *SuggestedVideos {
	return &SuggestedVideos{source: source, playlistID: playlistID, clock: clk}
}

// Suggest reports false when there is nothing to suggest: no playlist is
// configured or it holds no playable videos.
func (s *SuggestedVideos) Suggest(ctx context.Context) (youtube.Video, bool, error) {
	if s.playlistID == "" {
		return youtube.Video{}, false, nil
	}

	videos, err := s.playlist(ctx)
	if err != nil {
		return youtube.Video{}, false, fmt.Errorf("suggested video: %w", err)
	}
	if len(videos) == 0 {
		return youtube.Video{}, false, nil
	}
	return videos[rand.IntN(len(videos))], true, nil
}

// The lock is held across the fetch so concurrent visitors after an expiry
// trigger one upstream call, not one each.
func (s *SuggestedVideos) playlist(ctx context.Context) ([]youtube.Video, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.clock.Now()
	if !s.fetchedAt.IsZero() && now.Sub(s.fetchedAt) < SuggestedVideoCacheTTL {
		return s.videos, nil
	}

	videos, err := s.source.PlaylistVideos(ctx, s.playlistID)
	if err != nil {
		return nil, err
	}
	s.videos = videos
	s.fetchedAt = now
	return videos, nil
}
