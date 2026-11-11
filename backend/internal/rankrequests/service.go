package rankrequests

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/cageymage/fuzion/backend/internal/clock"
	"github.com/cageymage/fuzion/backend/internal/discord"
)

const maxNoteLength = 500

var (
	ErrNotFound          = errors.New("rank request not found")
	ErrCharacterNotFound = errors.New("character not found")
	ErrCharacterNotOwned = errors.New("character is not yours")
	ErrRequestNotOwned   = errors.New("rank request is not yours")
	ErrOpenRequestExists = errors.New("this character already has an open rank request")
	ErrNotOpen           = errors.New("rank request is no longer open")
)

type ValidationError struct {
	Field   string
	Problem string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Problem
}

type CreateRequest struct {
	Note string `json:"note"`
}

type ResolveRequest struct {
	Status string `json:"status"`
	Note   string `json:"note"`
}

// Notifier is the officers' rank requests channel; a nil Notifier means notifications are disabled.
type Notifier interface {
	Send(ctx context.Context, msg discord.Message) error
}

type Service struct {
	repo     *Repo
	clock    clock.Clock
	notifier Notifier

	siteBaseURL string
}

func NewService(repo *Repo, clock clock.Clock, notifier Notifier, siteBaseURL string) *Service {
	return &Service{repo: repo, clock: clock, notifier: notifier, siteBaseURL: strings.TrimRight(siteBaseURL, "/")}
}

// Create files a request to move the character to the other kind of rank: a main asks
// for an alt rank and an alt for a main one.
func (s *Service) Create(ctx context.Context, userID, characterID uuid.UUID, req CreateRequest) (Request, error) {
	note, err := validateNote("note", req.Note)
	if err != nil {
		return Request{}, err
	}

	owner, err := s.repo.characterOwner(ctx, characterID)
	if err != nil {
		return Request{}, fmt.Errorf("load character %s: %w", characterID, err)
	}
	if owner == nil || *owner != userID {
		return Request{}, ErrCharacterNotOwned
	}

	created, err := s.repo.Create(ctx, uuid.New(), characterID, userID, note, s.clock.Now())
	if err != nil {
		return Request{}, fmt.Errorf("create rank request: %w", err)
	}
	s.notifyOfficers(ctx, created)
	return created, nil
}

func (s *Service) ListMine(ctx context.Context, userID uuid.UUID) ([]Request, error) {
	requests, err := s.repo.ListLatestForOwner(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list my rank requests: %w", err)
	}
	return requests, nil
}

// Cancel lets the character's current owner withdraw an open request. The current
// owner decides, matching ListMine, so a request filed before the character changed
// hands does not leave the new owner stuck with it.
func (s *Service) Cancel(ctx context.Context, userID, id uuid.UUID) error {
	if err := s.repo.Cancel(ctx, id, userID, s.clock.Now()); err != nil {
		return fmt.Errorf("cancel rank request %s: %w", id, err)
	}
	return nil
}

func (s *Service) List(ctx context.Context) ([]Request, error) {
	requests, err := s.repo.ListForOfficers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list rank requests: %w", err)
	}
	return requests, nil
}

func (s *Service) CountOpen(ctx context.Context) (int, error) {
	n, err := s.repo.CountOpen(ctx)
	if err != nil {
		return 0, fmt.Errorf("count open rank requests: %w", err)
	}
	return n, nil
}

// Resolve records an officer's answer. Marking a request done does not touch the
// character: the next roster sync reads the new rank from the game.
func (s *Service) Resolve(ctx context.Context, officerID, id uuid.UUID, req ResolveRequest) (Request, error) {
	switch req.Status {
	case "done", "declined":
	default:
		return Request{}, &ValidationError{Field: "status", Problem: "must be one of done, declined"}
	}
	note, err := validateNote("note", req.Note)
	if err != nil {
		return Request{}, err
	}

	if err := s.repo.Close(ctx, id, req.Status, note, &officerID, s.clock.Now()); err != nil {
		return Request{}, fmt.Errorf("resolve rank request %s: %w", id, err)
	}
	resolved, err := s.repo.Get(ctx, id)
	if err != nil {
		return Request{}, fmt.Errorf("load resolved rank request %s: %w", id, err)
	}
	return resolved, nil
}

func validateNote(field, note string) (string, error) {
	note = strings.TrimSpace(note)
	if utf8.RuneCountInString(note) > maxNoteLength {
		return "", &ValidationError{Field: field, Problem: fmt.Sprintf("must be at most %d characters", maxNoteLength)}
	}
	return note, nil
}

// The request is already saved, so a failed ping must not fail it.
func (s *Service) notifyOfficers(ctx context.Context, req Request) {
	if s.notifier == nil {
		return
	}
	if err := s.notifier.Send(ctx, newRequestMessage(req, s.siteBaseURL)); err != nil {
		slog.ErrorContext(ctx, "notify officers of rank request", "rank_request_id", req.ID, "error", err)
	}
}

// escapeMarkdown keeps a member's note as plain text in the embed, so a note cannot
// show officers a masked link whose label hides where it goes, or restyle the post.
func escapeMarkdown(text string) string {
	var b strings.Builder
	for _, r := range text {
		if strings.ContainsRune(markdownCharacters, r) {
			b.WriteRune('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

const markdownCharacters = "\\*_~`|>[]()#-"

func newRequestMessage(req Request, siteBaseURL string) discord.Message {
	fullName := req.CharacterName + " " + req.CharacterSecondaryName
	change := "Alt to main rank"
	if req.Direction == "to_alt" {
		change = "Main to alt rank"
	}
	fields := []discord.EmbedField{
		{Name: "Character", Value: fullName + " (" + req.CharacterClass + ")"},
		{Name: "Request", Value: change},
		{Name: "Member", Value: req.RequestedBy},
	}
	// Discord rejects empty field values.
	if req.Note != "" {
		fields = append(fields, discord.EmbedField{Name: "Note", Value: escapeMarkdown(req.Note)})
	}
	return discord.Message{Embeds: []discord.Embed{{
		Title:       "Rank request: " + fullName,
		URL:         siteBaseURL + "/officer/rank-requests",
		Description: "An officer needs to change this character's rank in game.",
		Fields:      fields,
	}}}
}
