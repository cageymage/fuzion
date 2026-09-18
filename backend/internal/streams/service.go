package streams

import (
	"context"
	"fmt"
)

type Service struct {
	repo *Repo
}

func NewService(repo *Repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) LiveStreams(ctx context.Context) ([]Stream, error) {
	live, err := s.repo.ListLive(ctx)
	if err != nil {
		return nil, fmt.Errorf("live streams: %w", err)
	}
	return live, nil
}
