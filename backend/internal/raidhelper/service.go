package raidhelper

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
)

var ErrInvalidEvent = errors.New("raid-helper event has no id or start time")

// Event holds the webhook body fields the Calendar mirror uses. The body is the full
// Raid-Helper event object; see plans/integrations/raid-helper.md.
type Event struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	StartTime   int64  `json:"startTime"`
	LastUpdated int64  `json:"lastUpdated"`
	ChannelID   string `json:"channelId"`
	ChannelName string `json:"channelName"`
}

type Service struct {
	repo *Repo
}

func NewService(repo *Repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) Save(ctx context.Context, event Event) error {
	raid, err := toRaid(event)
	if err != nil {
		return err
	}
	return s.repo.Upsert(ctx, raid)
}

func (s *Service) Delete(ctx context.Context, event Event) error {
	raid, err := toRaid(event)
	if err != nil {
		return err
	}
	return s.repo.MarkDeleted(ctx, raid)
}

func toRaid(event Event) (mirroredRaid, error) {
	if event.ID == "" || event.StartTime <= 0 {
		return mirroredRaid{}, ErrInvalidEvent
	}
	return mirroredRaid{
		EventID:     event.ID,
		Difficulty:  difficultyFromTitle(event.Title),
		Title:       event.Title,
		StartsAt:    time.Unix(event.StartTime, 0).UTC(),
		LastUpdated: event.LastUpdated,
	}, nil
}

var difficultyWord = regexp.MustCompile(`(?i)\b(normal|heroic|mythic)\b`)

// Raid-Helper events have no difficulty field, so it can only come from the title.
func difficultyFromTitle(title string) string {
	word := difficultyWord.FindString(title)
	if word == "" {
		return "Unknown"
	}
	return strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
}
