package roster

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	minNameLength = 2
	maxNameLength = 12
	// Mirrors the default of the characters.realm column.
	defaultRealm = "Emberreach"
)

var (
	ErrNotFound = errors.New("character not found")
	ErrConflict = errors.New("character with this name and secondary name already exists")

	// Mirrors the CHECK constraint on characters.class.
	classes = []string{"Warrior", "Paladin", "Hunter", "Rogue", "Priest", "Shaman", "Mage", "Warlock", "Druid"}
	roles   = []string{"tank", "healer", "dps"}
)

type ValidationError struct {
	Field   string
	Problem string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Problem
}

type CreateRequest struct {
	Name          string  `json:"name"`
	SecondaryName string  `json:"secondaryName"`
	Realm         *string `json:"realm"`
	Class         string  `json:"class"`
	Spec          *string `json:"spec"`
	Role          string  `json:"role"`
	IsMain        bool    `json:"isMain"`
	RaidTeam      *string `json:"raidTeam"`
}

type UpdateRequest struct {
	Name          *string `json:"name"`
	SecondaryName *string `json:"secondaryName"`
	Realm         *string `json:"realm"`
	Class         *string `json:"class"`
	Spec          *string `json:"spec"`
	Role          *string `json:"role"`
	IsMain        *bool   `json:"isMain"`
	RaidTeam      *string `json:"raidTeam"`
}

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

func (s *Service) Create(ctx context.Context, req CreateRequest) (Character, error) {
	character := Character{
		ID:            uuid.New(),
		Name:          strings.TrimSpace(req.Name),
		SecondaryName: strings.TrimSpace(req.SecondaryName),
		Realm:         defaultRealm,
		Class:         strings.TrimSpace(req.Class),
		Spec:          trimOptional(req.Spec),
		Role:          strings.TrimSpace(req.Role),
		IsMain:        req.IsMain,
		RaidTeam:      trimOptional(req.RaidTeam),
	}
	if req.Realm != nil {
		character.Realm = strings.TrimSpace(*req.Realm)
	}
	if err := validateName("name", character.Name); err != nil {
		return Character{}, err
	}
	if err := validateName("secondaryName", character.SecondaryName); err != nil {
		return Character{}, err
	}
	if err := validateRealm(character.Realm); err != nil {
		return Character{}, err
	}
	if err := validateClass(character.Class); err != nil {
		return Character{}, err
	}
	if err := validateRole(character.Role); err != nil {
		return Character{}, err
	}

	created, err := s.repo.Create(ctx, character)
	if err != nil {
		return Character{}, fmt.Errorf("create character: %w", err)
	}
	return created, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (Character, error) {
	req.Name = trimOptional(req.Name)
	req.SecondaryName = trimOptional(req.SecondaryName)
	req.Realm = trimOptional(req.Realm)
	req.Class = trimOptional(req.Class)
	req.Spec = trimOptional(req.Spec)
	req.Role = trimOptional(req.Role)
	req.RaidTeam = trimOptional(req.RaidTeam)

	if req.Name != nil {
		if err := validateName("name", *req.Name); err != nil {
			return Character{}, err
		}
	}
	if req.SecondaryName != nil {
		if err := validateName("secondaryName", *req.SecondaryName); err != nil {
			return Character{}, err
		}
	}
	if req.Realm != nil {
		if err := validateRealm(*req.Realm); err != nil {
			return Character{}, err
		}
	}
	if req.Class != nil {
		if err := validateClass(*req.Class); err != nil {
			return Character{}, err
		}
	}
	if req.Role != nil {
		if err := validateRole(*req.Role); err != nil {
			return Character{}, err
		}
	}

	updated, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return Character{}, fmt.Errorf("update character %s: %w", id, err)
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete character %s: %w", id, err)
	}
	return nil
}

func trimOptional(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}

func validateName(field, name string) error {
	if length := utf8.RuneCountInString(name); length < minNameLength || length > maxNameLength {
		return &ValidationError{Field: field, Problem: fmt.Sprintf("must be between %d and %d characters", minNameLength, maxNameLength)}
	}
	return nil
}

func validateRealm(realm string) error {
	if realm == "" {
		return &ValidationError{Field: "realm", Problem: "must not be empty"}
	}
	return nil
}

func validateClass(class string) error {
	if !slices.Contains(classes, class) {
		return &ValidationError{Field: "class", Problem: "must be one of " + strings.Join(classes, ", ")}
	}
	return nil
}

func validateRole(role string) error {
	if !slices.Contains(roles, role) {
		return &ValidationError{Field: "role", Problem: "must be one of " + strings.Join(roles, ", ")}
	}
	return nil
}
