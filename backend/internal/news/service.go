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

type Page struct {
	Posts []Post
	Total int
}

func (s *Service) ListPosts(ctx context.Context, params ListParams) (Page, error) {
	posts, err := s.repo.ListPosts(ctx, params)
	if err != nil {
		return Page{}, fmt.Errorf("list posts: %w", err)
	}
	total, err := s.repo.CountPosts(ctx, params.Category)
	if err != nil {
		return Page{}, fmt.Errorf("list posts: %w", err)
	}
	return Page{Posts: posts, Total: total}, nil
}
