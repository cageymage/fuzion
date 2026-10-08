package roster

import (
	"context"
	"errors"
	"fmt"
	"maps"
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

	maxPrimaryProfessions = 2

	maxLevel = 60
)

var (
	ErrNotFound = errors.New("character not found")
	ErrConflict = errors.New("character with this name and secondary name already exists")

	// Mirrors the CHECK constraint on characters.class.
	classes = []string{"Warrior", "Paladin", "Hunter", "Rogue", "Priest", "Shaman", "Mage", "Warlock", "Druid"}
	roles   = []string{"tank", "healer", "dps"}

	// Mirrors the CHECK constraint on characters.race. Skyborne can join either faction, so
	// each faction has its own variant and faction stays derivable from race.
	raceFactions = map[string]string{
		"Human": "Alliance", "Dwarf": "Alliance", "Night Elf": "Alliance", "Gnome": "Alliance", "Skyborne (High Order)": "Alliance",
		"Orc": "Horde", "Undead": "Horde", "Tauren": "Horde", "Troll": "Horde", "Skyborne (Windshaper)": "Horde",
	}

	// Mirrors what Forever has; Jewelcrafting and Inscription are left out until it adds them.
	primaryProfessions   = []string{"Alchemy", "Blacksmithing", "Enchanting", "Engineering", "Herbalism", "Leatherworking", "Mining", "Skinning", "Tailoring"}
	secondaryProfessions = []string{"Cooking", "Fishing", "First Aid"}

	// Standard three-tree names; spec stays free text in the database so rows that
	// predate this list are left alone until an officer edits their spec.
	classSpecs = map[string][]string{
		"Warrior": {"Arms", "Fury", "Protection"},
		"Paladin": {"Holy", "Protection", "Retribution"},
		"Hunter":  {"Beast Mastery", "Marksmanship", "Survival"},
		"Rogue":   {"Assassination", "Combat", "Subtlety"},
		"Priest":  {"Discipline", "Holy", "Shadow"},
		"Shaman":  {"Elemental", "Enhancement", "Restoration"},
		"Mage":    {"Arcane", "Fire", "Frost"},
		"Warlock": {"Affliction", "Demonology", "Destruction"},
		"Druid":   {"Balance", "Feral", "Restoration"},
	}
)

type ValidationError struct {
	Field   string
	Problem string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Problem
}

type CreateRequest struct {
	Name          string   `json:"name"`
	SecondaryName string   `json:"secondaryName"`
	Realm         *string  `json:"realm"`
	Class         string   `json:"class"`
	Spec          *string  `json:"spec"`
	Role          string   `json:"role"`
	Spec2         *string  `json:"spec2"`
	Role2         *string  `json:"role2"`
	IsMain        bool     `json:"isMain"`
	RaidTeam      *string  `json:"raidTeam"`
	Professions   []string `json:"professions"`
	Race          *string  `json:"race"`
	Level         *int     `json:"level"`
}

type UpdateRequest struct {
	Name          *string   `json:"name"`
	SecondaryName *string   `json:"secondaryName"`
	Realm         *string   `json:"realm"`
	Class         *string   `json:"class"`
	Spec          *string   `json:"spec"`
	Role          *string   `json:"role"`
	Spec2         *string   `json:"spec2"`
	Role2         *string   `json:"role2"`
	IsMain        *bool     `json:"isMain"`
	RaidTeam      *string   `json:"raidTeam"`
	Professions   *[]string `json:"professions"`
	Race          *string   `json:"race"`
	Level         *int      `json:"level"`
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
	for i := range characters {
		characters[i].Faction = factionOf(characters[i].Race)
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
		Spec2:         blankToNil(req.Spec2),
		Role2:         blankToNil(req.Role2),
		IsMain:        req.IsMain,
		RaidTeam:      trimOptional(req.RaidTeam),
		Race:          blankToNil(req.Race),
		Level:         req.Level,
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
	if err := validateRole("role", character.Role); err != nil {
		return Character{}, err
	}
	if err := validateSpecs(character.Class, character.Spec, character.Spec2, character.Role2, allSpecChecks); err != nil {
		return Character{}, err
	}

	if err := validateRace(character.Race); err != nil {
		return Character{}, err
	}
	if err := validateLevel(character.Level); err != nil {
		return Character{}, err
	}

	professions, err := validateProfessions(req.Professions)
	if err != nil {
		return Character{}, err
	}

	created, err := s.repo.Create(ctx, character, professions)
	if err != nil {
		return Character{}, fmt.Errorf("create character: %w", err)
	}
	created.Faction = factionOf(created.Race)
	return created, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateRequest) (Character, error) {
	req.Name = trimOptional(req.Name)
	req.SecondaryName = trimOptional(req.SecondaryName)
	req.Realm = trimOptional(req.Realm)
	req.Class = trimOptional(req.Class)
	req.Spec = trimOptional(req.Spec)
	req.Role = trimOptional(req.Role)
	req.Spec2 = trimOptional(req.Spec2)
	req.Role2 = trimOptional(req.Role2)
	req.RaidTeam = trimOptional(req.RaidTeam)
	req.Race = trimOptional(req.Race)

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
		if err := validateRole("role", *req.Role); err != nil {
			return Character{}, err
		}
	}

	if err := validateRace(req.Race); err != nil {
		return Character{}, err
	}
	if err := validateLevel(req.Level); err != nil {
		return Character{}, err
	}

	var professions *[]string
	if req.Professions != nil {
		validated, err := validateProfessions(*req.Professions)
		if err != nil {
			return Character{}, err
		}
		professions = &validated
	}

	second, err := s.resolveSecondSpec(ctx, id, req)
	if err != nil {
		return Character{}, err
	}

	updated, err := s.repo.Update(ctx, id, req, second, professions)
	if err != nil {
		return Character{}, fmt.Errorf("update character %s: %w", id, err)
	}
	updated.Faction = factionOf(updated.Race)
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete character %s: %w", id, err)
	}
	return nil
}

// resolveSecondSpec validates the class, spec and second spec as they will be
// stored after req is applied. It returns the second spec to write when req
// touches it, or nil to leave the stored one unchanged.
func (s *Service) resolveSecondSpec(ctx context.Context, id uuid.UUID, req UpdateRequest) (*secondSpec, error) {
	touchesSecond := req.Spec2 != nil || req.Role2 != nil
	if req.Class == nil && req.Spec == nil && !touchesSecond {
		return nil, nil
	}

	current, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load character %s: %w", id, err)
	}

	class := current.Class
	if req.Class != nil {
		class = *req.Class
	}
	spec := current.Spec
	if req.Spec != nil {
		spec = req.Spec
	}
	spec2, role2 := current.Spec2, current.Role2
	if req.Spec2 != nil {
		spec2 = blankToNil(req.Spec2)
	}
	if req.Role2 != nil {
		role2 = blankToNil(req.Role2)
	}

	checks := specChecks{
		spec:  req.Class != nil || req.Spec != nil,
		spec2: req.Class != nil || req.Spec2 != nil,
		role2: req.Role2 != nil,
	}
	if err := validateSpecs(class, spec, spec2, role2, checks); err != nil {
		return nil, err
	}
	if !touchesSecond {
		return nil, nil
	}
	return &secondSpec{spec: spec2, role: role2}, nil
}

func blankToNil(value *string) *string {
	trimmed := trimOptional(value)
	if trimmed == nil || *trimmed == "" {
		return nil
	}
	return trimmed
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

func validateRole(field, role string) error {
	if !slices.Contains(roles, role) {
		return &ValidationError{Field: field, Problem: "must be one of " + strings.Join(roles, ", ")}
	}
	return nil
}

// specChecks selects which fields to validate against the class's spec list, so
// an update that leaves a legacy free-text spec alone does not trip over it.
type specChecks struct {
	spec, spec2, role2 bool
}

var allSpecChecks = specChecks{spec: true, spec2: true, role2: true}

func validateSpecs(class string, spec, spec2, role2 *string, checks specChecks) error {
	if checks.spec && spec != nil && *spec != "" {
		if err := validateClassSpec("spec", class, *spec); err != nil {
			return err
		}
	}
	if spec2 == nil && role2 != nil {
		return &ValidationError{Field: "spec2", Problem: "must be set together with role2"}
	}
	if spec2 != nil && role2 == nil {
		return &ValidationError{Field: "role2", Problem: "must be set together with spec2"}
	}
	if spec2 == nil {
		return nil
	}
	if checks.role2 {
		if err := validateRole("role2", *role2); err != nil {
			return err
		}
	}
	if checks.spec2 {
		if err := validateClassSpec("spec2", class, *spec2); err != nil {
			return err
		}
	}
	if spec != nil && *spec == *spec2 {
		return &ValidationError{Field: "spec2", Problem: "must differ from spec"}
	}
	return nil
}

func validateClassSpec(field, class, spec string) error {
	specs := classSpecs[class]
	if !slices.Contains(specs, spec) {
		return &ValidationError{Field: field, Problem: "must be one of " + strings.Join(specs, ", ")}
	}
	return nil
}

// validateProfessions trims the names and returns them as a non-nil slice.
func validateProfessions(names []string) ([]string, error) {
	validated := make([]string, 0, len(names))
	primaries := 0
	for _, name := range names {
		name = strings.TrimSpace(name)
		isPrimary := slices.Contains(primaryProfessions, name)
		if !isPrimary && !slices.Contains(secondaryProfessions, name) {
			return nil, &ValidationError{Field: "professions", Problem: fmt.Sprintf("%q is not a known profession", name)}
		}
		if slices.Contains(validated, name) {
			return nil, &ValidationError{Field: "professions", Problem: fmt.Sprintf("%q is listed more than once", name)}
		}
		if isPrimary {
			primaries++
		}
		validated = append(validated, name)
	}
	if primaries > maxPrimaryProfessions {
		return nil, &ValidationError{Field: "professions", Problem: fmt.Sprintf("at most %d primary professions are allowed", maxPrimaryProfessions)}
	}
	return validated, nil
}

func validateRace(race *string) error {
	if race == nil {
		return nil
	}
	if _, ok := raceFactions[*race]; !ok {
		races := slices.Sorted(maps.Keys(raceFactions))
		return &ValidationError{Field: "race", Problem: "must be one of " + strings.Join(races, ", ")}
	}
	return nil
}

func validateLevel(level *int) error {
	if level != nil && (*level < 1 || *level > maxLevel) {
		return &ValidationError{Field: "level", Problem: fmt.Sprintf("must be between 1 and %d", maxLevel)}
	}
	return nil
}

func factionOf(race *string) *string {
	if race == nil {
		return nil
	}
	faction := raceFactions[*race]
	return &faction
}
