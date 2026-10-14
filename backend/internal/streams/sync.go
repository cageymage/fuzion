package streams

import (
	"context"
	"errors"
	"fmt"

	"github.com/cageymage/fuzion/backend/internal/synclog"
	"github.com/cageymage/fuzion/backend/internal/twitch"
	"github.com/cageymage/fuzion/backend/internal/youtube"
)

const (
	twitchSyncSource  = "twitch"
	youTubeSyncSource = "youtube"
)

type SyncTwitch struct {
	repo    *Repo
	twitch  *twitch.Client
	syncLog *synclog.Repo
}

func NewSyncTwitch(repo *Repo, client *twitch.Client, syncLog *synclog.Repo) *SyncTwitch {
	return &SyncTwitch{repo: repo, twitch: client, syncLog: syncLog}
}

func (j *SyncTwitch) Run(ctx context.Context) error {
	message, err := j.sync(ctx)
	if err != nil {
		if logErr := j.syncLog.Record(ctx, twitchSyncSource, synclog.StatusError, err.Error()); logErr != nil {
			return errors.Join(err, logErr)
		}
		return err
	}
	return j.syncLog.Record(ctx, twitchSyncSource, synclog.StatusOK, message)
}

func (j *SyncTwitch) sync(ctx context.Context) (string, error) {
	logins, err := j.repo.LinkedLogins(ctx)
	if err != nil {
		return "", fmt.Errorf("list channels to check: %w", err)
	}

	liveStreams, err := j.twitch.LiveStreams(ctx, logins)
	if err != nil {
		return "", fmt.Errorf("ask twitch who is live: %w", err)
	}

	avatars, err := j.twitch.Avatars(ctx, logins)
	if err != nil {
		return "", fmt.Errorf("ask twitch for avatars: %w", err)
	}

	live := make(map[string]twitch.LiveStream, len(liveStreams))
	for _, s := range liveStreams {
		live[s.Login] = s
	}
	if err := j.repo.UpdateLiveStatus(ctx, live); err != nil {
		return "", fmt.Errorf("store live status: %w", err)
	}
	if err := j.repo.UpdateProfileImages(ctx, avatars); err != nil {
		return "", fmt.Errorf("store avatars: %w", err)
	}
	return fmt.Sprintf("%d of %d channels live", len(live), len(logins)), nil
}

type SyncYouTube struct {
	repo    *Repo
	youtube *youtube.Client
	syncLog *synclog.Repo
}

func NewSyncYouTube(repo *Repo, client *youtube.Client, syncLog *synclog.Repo) *SyncYouTube {
	return &SyncYouTube{repo: repo, youtube: client, syncLog: syncLog}
}

func (j *SyncYouTube) Run(ctx context.Context) error {
	message, err := j.sync(ctx)
	if err != nil {
		if logErr := j.syncLog.Record(ctx, youTubeSyncSource, synclog.StatusError, err.Error()); logErr != nil {
			return errors.Join(err, logErr)
		}
		return err
	}
	return j.syncLog.Record(ctx, youTubeSyncSource, synclog.StatusOK, message)
}

func (j *SyncYouTube) sync(ctx context.Context) (string, error) {
	channelIDs, err := j.repo.YouTubeChannelIDs(ctx)
	if err != nil {
		return "", fmt.Errorf("list channels to check: %w", err)
	}

	liveStreams, err := j.youtube.LiveStreams(ctx, channelIDs)
	if err != nil {
		return "", fmt.Errorf("ask youtube who is live: %w", err)
	}

	avatars, err := j.youtube.Avatars(ctx, channelIDs)
	if err != nil {
		return "", fmt.Errorf("ask youtube for avatars: %w", err)
	}

	live := make(map[string]youtube.LiveStream, len(liveStreams))
	for _, s := range liveStreams {
		live[s.ChannelID] = s
	}
	if err := j.repo.UpdateYouTubeLiveStatus(ctx, live); err != nil {
		return "", fmt.Errorf("store live status: %w", err)
	}
	if err := j.repo.UpdateYouTubeAvatars(ctx, avatars); err != nil {
		return "", fmt.Errorf("store avatars: %w", err)
	}
	return fmt.Sprintf("%d of %d channels live", len(live), len(channelIDs)), nil
}
