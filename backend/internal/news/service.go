package news

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

func (s *Service) LatestPosts(ctx context.Context) ([]Post, error) {
	posts, err := s.repo.ListPosts(ctx)
	if err != nil {
		return nil, fmt.Errorf("latest posts: %w", err)
	}
	return posts, nil
}
