package applications

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	maxDiscordHandleLength = 64
	maxNotesLength         = 2000
)

type SubmitRequest struct {
	ApplicantName string `json:"applicantName"`
	CharacterName string `json:"characterName"`
	Class         string `json:"class"`
	Role          string `json:"role"`
	Availability  string `json:"availability"`
	DiscordHandle string `json:"discordHandle"`
	Notes         string `json:"notes"`
}

type ValidationError struct {
	Field   string
	Problem string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Problem
}

type Service struct {
	repo *Repo
}

func NewService(repo *Repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) Submit(ctx context.Context, req SubmitRequest) (Submitted, error) {
	app := Application{
		ID:            uuid.New(),
		ApplicantName: strings.TrimSpace(req.ApplicantName),
		CharacterName: strings.TrimSpace(req.CharacterName),
		Class:         strings.TrimSpace(req.Class),
		Role:          strings.TrimSpace(req.Role),
		Availability:  strings.TrimSpace(req.Availability),
		DiscordHandle: strings.TrimSpace(req.DiscordHandle),
		Notes:         strings.TrimSpace(req.Notes),
	}
	if err := validate(app); err != nil {
		return Submitted{}, err
	}

	submitted, err := s.repo.Create(ctx, app)
	if err != nil {
		return Submitted{}, fmt.Errorf("submit application: %w", err)
	}
	return submitted, nil
}

func validate(app Application) error {
	required := []struct {
		field string
		value string
	}{
		{"applicantName", app.ApplicantName},
		{"characterName", app.CharacterName},
		{"class", app.Class},
		{"role", app.Role},
		{"availability", app.Availability},
		{"discordHandle", app.DiscordHandle},
	}
	for _, r := range required {
		if r.value == "" {
			return &ValidationError{Field: r.field, Problem: "is required"}
		}
	}

	switch app.Role {
	case "tank", "healer", "dps":
	default:
		return &ValidationError{Field: "role", Problem: "must be one of tank, healer, dps"}
	}

	if utf8.RuneCountInString(app.DiscordHandle) > maxDiscordHandleLength {
		return &ValidationError{Field: "discordHandle", Problem: fmt.Sprintf("must be at most %d characters", maxDiscordHandleLength)}
	}
	if utf8.RuneCountInString(app.Notes) > maxNotesLength {
		return &ValidationError{Field: "notes", Problem: fmt.Sprintf("must be at most %d characters", maxNotesLength)}
	}
	return nil
}
