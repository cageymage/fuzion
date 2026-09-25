package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

const sessionTTL = 30 * 24 * time.Hour

var ErrCannotRemoveOwnAdmin = errors.New("cannot remove your own admin access")

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
	user, grantedFirstAdmin, err := s.repo.UpsertUserByDiscordID(ctx, identity)
	if err != nil {
		return User{}, "", fmt.Errorf("login: %w", err)
	}
	// Should happen exactly once in a database's life. Seeing it on a live site
	// means the users table was empty when it shouldn't have been.
	if grantedFirstAdmin {
		slog.WarnContext(ctx, "granted admin to the first account created on this site",
			"user_id", user.ID, "username", user.Username, "discord_id", user.DiscordID)
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

func (s *Service) ListUsers(ctx context.Context) ([]User, error) {
	users, err := s.repo.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}

// UpdateUserRoles applies an admin's role change to a user. An admin can never
// remove their own is_admin flag, which would otherwise lock everyone out.
func (s *Service) UpdateUserRoles(ctx context.Context, caller User, targetID uuid.UUID, isOfficer, isAdmin bool) (User, error) {
	if targetID == caller.ID && !isAdmin {
		return User{}, ErrCannotRemoveOwnAdmin
	}
	user, err := s.repo.UpdateUserRoles(ctx, targetID, isOfficer, isAdmin)
	if err != nil {
		return User{}, fmt.Errorf("update user roles: %w", err)
	}
	return user, nil
}
