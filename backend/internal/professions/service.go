package professions

import (
	"context"
	"fmt"
	"strings"
)

type Service struct {
	repo *Repo
}

func NewService(repo *Repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, profession string) ([]Profession, error) {
	professions, err := s.repo.List(ctx, strings.TrimSpace(profession))
	if err != nil {
		return nil, fmt.Errorf("professions: %w", err)
	}
	return professions, nil
}
