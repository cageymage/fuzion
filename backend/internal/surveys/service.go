package surveys

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/cageymage/fuzion/backend/internal/auth"
	"github.com/cageymage/fuzion/backend/internal/clock"
)

const maxGuildNameLength = 64

var (
	ErrClosed              = errors.New("survey is closed")
	ErrResultsNotAvailable = errors.New("results are not available until the survey closes")
)

type ValidationError struct {
	Field   string
	Problem string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Problem
}

type Summary struct {
	ID       uuid.UUID `json:"id"`
	Title    string    `json:"title"`
	ClosesAt time.Time `json:"closesAt"`
	Status   string    `json:"status"`
}

type Question struct {
	ID       uuid.UUID `json:"id"`
	Prompt   string    `json:"prompt"`
	Type     string    `json:"type"`
	Required bool      `json:"required"`
	Options  []string  `json:"options"`
}

type Answer struct {
	QuestionID uuid.UUID `json:"questionId"`
	Values     []string  `json:"values"`
}

type MyResponse struct {
	GuildName string   `json:"guildName"`
	Answers   []Answer `json:"answers"`
}

type Detail struct {
	Summary
	Description string      `json:"description"`
	Questions   []Question  `json:"questions"`
	MyResponse  *MyResponse `json:"myResponse"`
}

type SubmitRequest struct {
	GuildName string   `json:"guildName"`
	Answers   []Answer `json:"answers"`
}

type OptionCount struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

type QuestionResult struct {
	ID      uuid.UUID     `json:"id"`
	Prompt  string        `json:"prompt"`
	Type    string        `json:"type"`
	Options []OptionCount `json:"options"`
}

type Results struct {
	SurveyID  uuid.UUID        `json:"surveyId"`
	Title     string           `json:"title"`
	Turnout   int              `json:"turnout"`
	Questions []QuestionResult `json:"questions"`
}

type Service struct {
	repo  *Repo
	clock clock.Clock
}

func NewService(repo *Repo, clock clock.Clock) *Service {
	return &Service{repo: repo, clock: clock}
}

func (s *Service) closed(sv survey) bool {
	return !s.clock.Now().Before(sv.ClosesAt)
}

func (s *Service) summary(sv survey) Summary {
	status := "open"
	if s.closed(sv) {
		status = "closed"
	}
	return Summary{ID: sv.ID, Title: sv.Title, ClosesAt: sv.ClosesAt, Status: status}
}

func (s *Service) List(ctx context.Context) ([]Summary, error) {
	surveys, err := s.repo.ListSurveys(ctx)
	if err != nil {
		return nil, fmt.Errorf("list surveys: %w", err)
	}
	summaries := make([]Summary, len(surveys))
	for i, sv := range surveys {
		summaries[i] = s.summary(sv)
	}
	return summaries, nil
}

func (s *Service) load(ctx context.Context, rawID string) (survey, error) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return survey{}, ErrNotFound
	}
	return s.repo.GetSurvey(ctx, id)
}

func (s *Service) Get(ctx context.Context, rawID string, viewer *auth.User) (Detail, error) {
	sv, err := s.load(ctx, rawID)
	if err != nil {
		return Detail{}, fmt.Errorf("get survey: %w", err)
	}
	questions, err := s.repo.Questions(ctx, sv.ID)
	if err != nil {
		return Detail{}, fmt.Errorf("get survey: %w", err)
	}

	detail := Detail{Summary: s.summary(sv), Description: sv.Description, Questions: make([]Question, len(questions))}
	for i, q := range questions {
		detail.Questions[i] = Question{ID: q.ID, Prompt: q.Prompt, Type: q.Type, Required: q.Required, Options: q.Options}
	}
	if viewer != nil {
		stored, err := s.repo.ResponseOf(ctx, sv.ID, viewer.ID)
		if err != nil {
			return Detail{}, fmt.Errorf("get survey: %w", err)
		}
		detail.MyResponse = toMyResponse(stored)
	}
	return detail, nil
}

func toMyResponse(stored *storedResponse) *MyResponse {
	if stored == nil {
		return nil
	}
	response := &MyResponse{GuildName: stored.GuildName, Answers: make([]Answer, len(stored.Answers))}
	for i, a := range stored.Answers {
		response.Answers[i] = Answer{QuestionID: a.QuestionID, Values: a.Values}
	}
	return response
}

func (s *Service) Submit(ctx context.Context, rawID string, user auth.User, req SubmitRequest) (MyResponse, error) {
	sv, err := s.load(ctx, rawID)
	if err != nil {
		return MyResponse{}, fmt.Errorf("submit survey response: %w", err)
	}
	if s.closed(sv) {
		return MyResponse{}, ErrClosed
	}
	questions, err := s.repo.Questions(ctx, sv.ID)
	if err != nil {
		return MyResponse{}, fmt.Errorf("submit survey response: %w", err)
	}

	guildName := strings.TrimSpace(req.GuildName)
	if utf8.RuneCountInString(guildName) > maxGuildNameLength {
		return MyResponse{}, &ValidationError{Field: "guildName", Problem: fmt.Sprintf("must be at most %d characters", maxGuildNameLength)}
	}
	answers, err := validateAnswers(questions, req.Answers)
	if err != nil {
		return MyResponse{}, err
	}

	err = s.repo.SaveResponse(ctx, responseInput{
		SurveyID:        sv.ID,
		UserID:          user.ID,
		GuildName:       guildName,
		DiscordUsername: user.Username,
		InDiscordServer: user.InDiscordServer,
		Answers:         answers,
	})
	if err != nil {
		return MyResponse{}, fmt.Errorf("submit survey response: %w", err)
	}
	return *toMyResponse(&storedResponse{GuildName: guildName, Answers: answers}), nil
}

// validateAnswers returns the answers to store, in question order. An optional
// question with no values counts as unanswered and stores nothing.
func validateAnswers(questions []question, submitted []Answer) ([]answer, error) {
	byID := make(map[uuid.UUID]question, len(questions))
	for _, q := range questions {
		byID[q.ID] = q
	}

	given := make(map[uuid.UUID][]string, len(submitted))
	for _, a := range submitted {
		q, ok := byID[a.QuestionID]
		if !ok {
			return nil, &ValidationError{Field: "answers", Problem: a.QuestionID.String() + " is not a question of this survey"}
		}
		if _, dup := given[q.ID]; dup {
			return nil, &ValidationError{Field: "answers", Problem: q.Prompt + " is answered more than once"}
		}
		if len(a.Values) == 0 {
			continue
		}
		if q.Type == "single" && len(a.Values) != 1 {
			return nil, &ValidationError{Field: "answers", Problem: q.Prompt + " takes exactly one value"}
		}
		for i, value := range a.Values {
			if !slices.Contains(q.Options, value) {
				return nil, &ValidationError{Field: "answers", Problem: fmt.Sprintf("%q is not an option for %s", value, q.Prompt)}
			}
			if slices.Contains(a.Values[:i], value) {
				return nil, &ValidationError{Field: "answers", Problem: fmt.Sprintf("%q is chosen more than once for %s", value, q.Prompt)}
			}
		}
		given[q.ID] = a.Values
	}

	var answers []answer
	for _, q := range questions {
		values, answered := given[q.ID]
		if !answered {
			if q.Required {
				return nil, &ValidationError{Field: "answers", Problem: q.Prompt + " is required"}
			}
			continue
		}
		answers = append(answers, answer{QuestionID: q.ID, Values: values})
	}
	return answers, nil
}

// Results is public, but only once the deadline has passed.
func (s *Service) Results(ctx context.Context, rawID string) (Results, error) {
	sv, err := s.load(ctx, rawID)
	if err != nil {
		return Results{}, fmt.Errorf("get survey results: %w", err)
	}
	if !s.closed(sv) {
		return Results{}, ErrResultsNotAvailable
	}
	questions, err := s.repo.Questions(ctx, sv.ID)
	if err != nil {
		return Results{}, fmt.Errorf("get survey results: %w", err)
	}
	turnout, counts, err := s.repo.Tally(ctx, sv.ID)
	if err != nil {
		return Results{}, fmt.Errorf("get survey results: %w", err)
	}

	type key struct {
		questionID uuid.UUID
		label      string
	}
	tallied := make(map[key]int, len(counts))
	for _, c := range counts {
		tallied[key{c.QuestionID, c.Label}] = c.Count
	}

	results := Results{SurveyID: sv.ID, Title: sv.Title, Turnout: turnout, Questions: make([]QuestionResult, len(questions))}
	for i, q := range questions {
		options := make([]OptionCount, len(q.Options))
		for j, label := range q.Options {
			options[j] = OptionCount{Label: label, Count: tallied[key{q.ID, label}]}
		}
		results.Questions[i] = QuestionResult{ID: q.ID, Prompt: q.Prompt, Type: q.Type, Options: options}
	}
	return results, nil
}
