package professions

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/cageymage/fuzion/backend/internal/auth"
)

var (
	ErrNotOwner     = errors.New("character is not yours")
	ErrFromBlizzard = errors.New("specialization comes from Battle.net")
)

const (
	sourceBlizzard = "blizzard"
	sourceManual   = "manual"
)

var specializations = map[string][]string{
	"Blacksmithing":  {"Armorsmith", "Weaponsmith"},
	"Engineering":    {"Gnomish", "Goblin"},
	"Leatherworking": {"Dragonscale", "Elemental", "Tribal"},
}

// Blizzard names the engineering specializations after the crafter. "Gnomish Engineer" is
// the only one seen in a real Forever response; "Goblin Engineer" follows its pattern.
var blizzardSpecializations = map[string]string{
	"Gnomish Engineer": "Gnomish",
	"Goblin Engineer":  "Goblin",
}

// specializationFromBlizzard returns false when the site does not know the name.
func specializationFromBlizzard(profession, reported string) (string, bool) {
	name := reported
	if mapped, ok := blizzardSpecializations[reported]; ok {
		name = mapped
	}
	return name, slices.Contains(specializations[profession], name)
}

// SetSpecializationRequest keeps the value raw so that a missing field can be told from
// null, which clears the specialization.
type SetSpecializationRequest struct {
	Specialization json.RawMessage `json:"specialization"`
}

func (s *Service) SetSpecialization(ctx context.Context, user auth.User, id uuid.UUID, req SetSpecializationRequest) (Profession, error) {
	target, err := s.repo.specializationTarget(ctx, id)
	if err != nil {
		return Profession{}, fmt.Errorf("set specialization of %s: %w", id, err)
	}
	owned := target.OwnerID != nil && *target.OwnerID == user.ID
	if !owned && !user.HasOfficerAccess() {
		return Profession{}, ErrNotOwner
	}

	specialization, err := parseSpecialization(target.Profession, req.Specialization)
	if err != nil {
		return Profession{}, err
	}
	updated, err := s.repo.setManualSpecialization(ctx, id, target.Profession, specialization)
	if err != nil {
		return Profession{}, fmt.Errorf("set specialization of %s: %w", id, err)
	}
	return updated, nil
}

func parseSpecialization(profession string, raw json.RawMessage) (*string, error) {
	var specialization *string
	if len(raw) == 0 || json.Unmarshal(raw, &specialization) != nil {
		return nil, &ValidationError{Field: "specialization", Problem: "must be a string or null"}
	}
	if specialization == nil {
		return nil, nil
	}
	valid := specializations[profession]
	if len(valid) == 0 {
		return nil, &ValidationError{Field: "specialization", Problem: profession + " has none to choose from"}
	}
	if !slices.Contains(valid, *specialization) {
		return nil, &ValidationError{Field: "specialization", Problem: "must be one of " + strings.Join(valid, ", ")}
	}
	return specialization, nil
}

type specializationTarget struct {
	Profession string     `db:"profession"`
	OwnerID    *uuid.UUID `db:"user_id"`
}

func (r *Repo) specializationTarget(ctx context.Context, id uuid.UUID) (specializationTarget, error) {
	const query = `
		SELECT p.profession, c.user_id
		FROM professions p
		JOIN characters c ON c.id = p.character_id
		WHERE p.id = $1`

	var target specializationTarget
	if err := r.db.GetContext(ctx, &target, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return specializationTarget{}, ErrNotFound
		}
		return specializationTarget{}, fmt.Errorf("select profession: %w", err)
	}
	return target, nil
}

// setManualSpecialization writes only while the row is still the profession the value was
// checked against and no sync has stored a Blizzard value.
func (r *Repo) setManualSpecialization(ctx context.Context, id uuid.UUID, profession string, specialization *string) (Profession, error) {
	const query = `
		WITH p AS (
			UPDATE professions SET
				specialization        = $3::text,
				specialization_source = CASE WHEN $3::text IS NULL THEN NULL ELSE '` + sourceManual + `' END
			WHERE id = $1 AND profession = $2
			  AND specialization_source IS DISTINCT FROM '` + sourceBlizzard + `'
			RETURNING id, character_id, profession, skill_level, specialization, specialization_source
		)
		SELECT ` + selectColumns + `
		FROM p
		JOIN characters c ON c.id = p.character_id`

	var updated row
	if err := r.db.GetContext(ctx, &updated, query, id, profession, specialization); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Profession{}, ErrFromBlizzard
		}
		return Profession{}, fmt.Errorf("write specialization: %w", err)
	}
	return updated.toProfession(), nil
}

func (h *Handler) setSpecialization(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var req SetSpecializationRequest
	if !decodeBody(w, r, &req) {
		return
	}

	user, _ := auth.UserFrom(r.Context())
	updated, err := h.service.SetSpecialization(r.Context(), user, id, req)
	if err != nil {
		writeError(w, r, "set specialization", err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}
