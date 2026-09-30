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

type UpdateTierRequest struct {
	Name      *string `json:"name"`
	IsCurrent *bool   `json:"isCurrent"`
}

type UpdateBossRequest struct {
	Name   *string `json:"name"`
	Killed *bool   `json:"killed"`
}

type AddBossRequest struct {
	Name string `json:"name"`
}

type ReorderRequest struct {
	IDs []uuid.UUID `json:"ids"`
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

func (s *Service) UpdateTier(ctx context.Context, id uuid.UUID, req UpdateTierRequest) (Tier, error) {
	if req.Name == nil && req.IsCurrent == nil {
		return Tier{}, &ValidationError{Field: "body", Problem: "must set name or isCurrent"}
	}
	name, err := optionalName(req.Name)
	if err != nil {
		return Tier{}, err
	}

	updated, err := s.repo.UpdateTier(ctx, id, name, req.IsCurrent)
	if err != nil {
		return Tier{}, fmt.Errorf("update raid tier %s: %w", id, err)
	}
	return updated, nil
}

func (s *Service) UpdateBoss(ctx context.Context, id uuid.UUID, req UpdateBossRequest) (Boss, error) {
	if req.Name == nil && req.Killed == nil {
		return Boss{}, &ValidationError{Field: "body", Problem: "must set name or killed"}
	}
	name, err := optionalName(req.Name)
	if err != nil {
		return Boss{}, err
	}

	var killedAt *time.Time
	if req.Killed != nil && *req.Killed {
		now := s.clock.Now()
		killedAt = &now
	}

	boss, err := s.repo.UpdateBoss(ctx, id, name, req.Killed != nil, killedAt)
	if err != nil {
		return Boss{}, fmt.Errorf("update raid boss %s: %w", id, err)
	}
	return boss, nil
}

func (s *Service) AddBoss(ctx context.Context, tierID uuid.UUID, req AddBossRequest) (Boss, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return Boss{}, &ValidationError{Field: "name", Problem: "must not be empty"}
	}

	boss, err := s.repo.AddBoss(ctx, tierID, name)
	if err != nil {
		return Boss{}, fmt.Errorf("add boss to raid tier %s: %w", tierID, err)
	}
	return boss, nil
}

func (s *Service) ReorderTiers(ctx context.Context, req ReorderRequest) error {
	if err := s.repo.ReorderTiers(ctx, req.IDs); err != nil {
		return fmt.Errorf("reorder raid tiers: %w", err)
	}
	return nil
}

func (s *Service) ReorderBosses(ctx context.Context, tierID uuid.UUID, req ReorderRequest) error {
	if err := s.repo.ReorderBosses(ctx, tierID, req.IDs); err != nil {
		return fmt.Errorf("reorder bosses in raid tier %s: %w", tierID, err)
	}
	return nil
}

func optionalName(name *string) (*string, error) {
	if name == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*name)
	if trimmed == "" {
		return nil, &ValidationError{Field: "name", Problem: "must not be empty"}
	}
	return &trimmed, nil
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
