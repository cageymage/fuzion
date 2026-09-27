package applications

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/cageymage/fuzion/backend/internal/clock"
)

const (
	maxDiscordHandleLength = 64
	maxNotesLength         = 2000
	maxReviewNoteLength    = 2000
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

type ReviewRequest struct {
	Status     string `json:"status"`
	ReviewNote string `json:"reviewNote"`
}

type ValidationError struct {
	Field   string
	Problem string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Problem
}

type Service struct {
	repo  *Repo
	clock clock.Clock
}

func NewService(repo *Repo, clock clock.Clock) *Service {
	return &Service{repo: repo, clock: clock}
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

func (s *Service) List(ctx context.Context, status string) ([]Reviewable, error) {
	switch status {
	case "", "pending", "accepted", "declined":
	default:
		return nil, &ValidationError{Field: "status", Problem: "must be one of pending, accepted, declined"}
	}

	apps, err := s.repo.List(ctx, status)
	if err != nil {
		return nil, fmt.Errorf("list applications: %w", err)
	}
	return apps, nil
}

func (s *Service) Get(ctx context.Context, id string) (Reviewable, error) {
	appID, err := parseID(id)
	if err != nil {
		return Reviewable{}, err
	}

	app, err := s.repo.Get(ctx, appID)
	if err != nil {
		return Reviewable{}, fmt.Errorf("get application: %w", err)
	}
	return app, nil
}

func (s *Service) Review(ctx context.Context, id string, reviewer uuid.UUID, req ReviewRequest) (Reviewable, error) {
	appID, err := parseID(id)
	if err != nil {
		return Reviewable{}, err
	}
	note := strings.TrimSpace(req.ReviewNote)
	switch req.Status {
	case "accepted", "declined":
	default:
		return Reviewable{}, &ValidationError{Field: "status", Problem: "must be one of accepted, declined"}
	}
	if utf8.RuneCountInString(note) > maxReviewNoteLength {
		return Reviewable{}, &ValidationError{Field: "reviewNote", Problem: fmt.Sprintf("must be at most %d characters", maxReviewNoteLength)}
	}

	app, err := s.repo.Review(ctx, appID, req.Status, note, reviewer, s.clock.Now())
	if err != nil {
		return Reviewable{}, fmt.Errorf("review application: %w", err)
	}
	return app, nil
}

func parseID(id string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, &ValidationError{Field: "id", Problem: "must be a valid UUID"}
	}
	return parsed, nil
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
