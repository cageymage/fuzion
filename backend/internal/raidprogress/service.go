package raidprogress

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/cageymage/fuzion/backend/internal/clock"
)

type ValidationError struct {
	Field   string
	Problem string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Problem
}

type CreateTierRequest struct {
	Name      string   `json:"name"`
	SortOrder int      `json:"sortOrder"`
	Bosses    []string `json:"bosses"`
}

type SetCurrentRequest struct {
	IsCurrent bool `json:"isCurrent"`
}

type SetKilledRequest struct {
	Killed bool `json:"killed"`
}

type Service struct {
	repo  *Repo
	clock clock.Clock
}

func NewService(repo *Repo, clock clock.Clock) *Service {
	return &Service{repo: repo, clock: clock}
}

func (s *Service) CurrentProgress(ctx context.Context) ([]Progress, error) {
	progress, err := s.repo.CurrentProgress(ctx)
	if err != nil {
		return nil, fmt.Errorf("current raid progress: %w", err)
	}
	return progress, nil
}

func (s *Service) ListTiers(ctx context.Context) ([]Tier, error) {
	tiers, err := s.repo.ListTiers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list raid tiers: %w", err)
	}
	return tiers, nil
}

func (s *Service) CreateTier(ctx context.Context, req CreateTierRequest) (Tier, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return Tier{}, &ValidationError{Field: "name", Problem: "must not be empty"}
	}

	bossNames := make([]string, len(req.Bosses))
	for i, boss := range req.Bosses {
		trimmed := strings.TrimSpace(boss)
		if trimmed == "" {
			return Tier{}, &ValidationError{Field: "bosses", Problem: "must not contain an empty name"}
		}
		bossNames[i] = trimmed
	}

	created, err := s.repo.CreateTier(ctx, name, req.SortOrder, bossNames)
	if err != nil {
		return Tier{}, fmt.Errorf("create raid tier: %w", err)
	}
	return created, nil
}

func (s *Service) SetTierCurrent(ctx context.Context, id uuid.UUID, req SetCurrentRequest) (Tier, error) {
	updated, err := s.repo.SetTierCurrent(ctx, id, req.IsCurrent)
	if err != nil {
		return Tier{}, fmt.Errorf("set raid tier %s current: %w", id, err)
	}
	return updated, nil
}

func (s *Service) SetBossKilled(ctx context.Context, id uuid.UUID, req SetKilledRequest) (Boss, error) {
	var killedAt *time.Time
	if req.Killed {
		now := s.clock.Now()
		killedAt = &now
	}

	boss, err := s.repo.SetBossKilled(ctx, id, killedAt)
	if err != nil {
		return Boss{}, fmt.Errorf("set raid boss %s killed: %w", id, err)
	}
	return boss, nil
}

func (s *Service) DeleteTier(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.DeleteTier(ctx, id); err != nil {
		return fmt.Errorf("delete raid tier %s: %w", id, err)
	}
	return nil
}

func (s *Service) DeleteBoss(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.DeleteBoss(ctx, id); err != nil {
		return fmt.Errorf("delete raid boss %s: %w", id, err)
	}
	return nil
}
