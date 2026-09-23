package roster

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

func (s *Service) Roster(ctx context.Context) ([]Character, error) {
	characters, err := s.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("roster: %w", err)
	}
	return characters, nil
}
