package professions

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrNotFound         = errors.New("profession not found")
	ErrConflict         = errors.New("character already has this profession")
	ErrUnknownCharacter = errors.New("character not found")
)

type ValidationError struct {
	Field   string
	Problem string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Problem
}

type CreateRequest struct {
	CharacterID string `json:"characterId"`
	Profession  string `json:"profession"`
	SkillLevel  int    `json:"skillLevel"`
}

type UpdateRequest struct {
	Profession *string `json:"profession"`
	SkillLevel *int    `json:"skillLevel"`
}

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

func (s *Service) Create(ctx context.Context, req CreateRequest) (Profession, error) {
	characterID, err := uuid.Parse(strings.TrimSpace(req.CharacterID))
	if err != nil {
		return Profession{}, errUnknownCharacter()
	}
	profession := strings.TrimSpace(req.Profession)
	if err := validateProfession(profession); err != nil {
		return Profession{}, err
	}
	if err := validateSkillLevel(req.SkillLevel); err != nil {
		return Profession{}, err
	}

	created, err := s.repo.Create(ctx, uuid.New(), characterID, profession, req.SkillLevel)
	if errors.Is(err, ErrUnknownCharacter) {
		return Profession{}, errUnknownCharacter()
	}
	if err != nil {
		return Profession{}, fmt.Errorf("create profession: %w", err)
	}
	return created, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (Profession, error) {
	if req.Profession != nil {
		trimmed := strings.TrimSpace(*req.Profession)
		if err := validateProfession(trimmed); err != nil {
			return Profession{}, err
		}
		req.Profession = &trimmed
	}
	if req.SkillLevel != nil {
		if err := validateSkillLevel(*req.SkillLevel); err != nil {
			return Profession{}, err
		}
	}

	updated, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return Profession{}, fmt.Errorf("update profession %s: %w", id, err)
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete profession %s: %w", id, err)
	}
	return nil
}

func errUnknownCharacter() error {
	return &ValidationError{Field: "characterId", Problem: "must be an existing character"}
}

func validateProfession(profession string) error {
	if profession == "" {
		return &ValidationError{Field: "profession", Problem: "must not be empty"}
	}
	return nil
}

func validateSkillLevel(skillLevel int) error {
	if skillLevel < 0 {
		return &ValidationError{Field: "skillLevel", Problem: "must be 0 or more"}
	}
	return nil
}
