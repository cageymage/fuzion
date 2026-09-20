package auth

import (
	"context"
	"fmt"
	"time"
)

const sessionTTL = 30 * 24 * time.Hour

type Service struct {
	provider Provider
	repo     *Repo
}

func NewService(provider Provider, repo *Repo) *Service {
	return &Service{provider: provider, repo: repo}
}

func (s *Service) AuthURL(state string) string {
	return s.provider.AuthURL(state)
}

// Login turns a provider callback code into a persisted user and a fresh session token.
func (s *Service) Login(ctx context.Context, code string) (User, string, error) {
	identity, err := s.provider.Exchange(ctx, code)
	if err != nil {
		return User{}, "", fmt.Errorf("login: %w", err)
	}
	user, err := s.repo.UpsertUserByDiscordID(ctx, identity)
	if err != nil {
		return User{}, "", fmt.Errorf("login: %w", err)
	}
	token, err := newSessionToken()
	if err != nil {
		return User{}, "", fmt.Errorf("login: %w", err)
	}
	if err := s.repo.CreateSession(ctx, token, user.ID, time.Now().Add(sessionTTL)); err != nil {
		return User{}, "", fmt.Errorf("login: %w", err)
	}
	return user, token, nil
}

func (s *Service) UserBySession(ctx context.Context, token string) (User, error) {
	user, err := s.repo.FindSession(ctx, token)
	if err != nil {
		return User{}, fmt.Errorf("user by session: %w", err)
	}
	return user, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	if err := s.repo.DeleteSession(ctx, token); err != nil {
		return fmt.Errorf("logout: %w", err)
	}
	return nil
}
