package raids

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

func (s *Service) NextRaid(ctx context.Context) (*Raid, error) {
	raid, err := s.repo.NextRaid(ctx)
	if err != nil {
		return nil, fmt.Errorf("next raid: %w", err)
	}
	return raid, nil
}
