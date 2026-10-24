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
)

const (
	maxTitleLength       = 120
	maxDescriptionLength = 2000
	maxQuestions         = 30
	maxPromptLength      = 200
	maxOptions           = 50
	maxOptionLength      = 100
	maxVoidReasonLength  = 500
)

var presets = map[string][]string{
	"wow-classes": {"Druid", "Hunter", "Mage", "Paladin", "Priest", "Rogue", "Shaman", "Warlock", "Warrior"},
	"wow-roles":   {"Tank", "Healer", "Melee DPS", "Ranged DPS"},
	"weekdays":    {"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"},
}

type QuestionInput struct {
	Prompt   string   `json:"prompt"`
	Type     string   `json:"type"`
	Required bool     `json:"required"`
	Options  []string `json:"options"`
	Preset   string   `json:"preset"`
}

type CreateRequest struct {
	Title       string          `json:"title"`
	Description string          `json:"description"`
	ClosesAt    time.Time       `json:"closesAt"`
	Questions   []QuestionInput `json:"questions"`
}

type UpdateRequest struct {
	Title       *string          `json:"title"`
	Description *string          `json:"description"`
	ClosesAt    *time.Time       `json:"closesAt"`
	Questions   *[]QuestionInput `json:"questions"`
}

type AddOptionRequest struct {
	Option string `json:"option"`
}

type VoidRequest struct {
	Voided bool   `json:"voided"`
	Reason string `json:"reason"`
}

type OfficerResponse struct {
	ID              uuid.UUID  `json:"id"`
	DiscordUsername string     `json:"discordUsername"`
	GuildName       string     `json:"guildName"`
	InDiscordServer bool       `json:"inDiscordServer"`
	SubmittedAt     time.Time  `json:"submittedAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	VoidedAt        *time.Time `json:"voidedAt"`
	VoidedBy        *string    `json:"voidedBy"`
	VoidReason      string     `json:"voidReason"`
	Answers         []Answer   `json:"answers"`
}

func (s *Service) Create(ctx context.Context, officer auth.User, req CreateRequest) (Detail, error) {
	title, err := validateTitle(req.Title)
	if err != nil {
		return Detail{}, err
	}
	description, err := validateDescription(req.Description)
	if err != nil {
		return Detail{}, err
	}
	if req.ClosesAt.IsZero() {
		return Detail{}, &ValidationError{Field: "closesAt", Problem: "is required"}
	}
	if !req.ClosesAt.After(s.clock.Now()) {
		return Detail{}, &ValidationError{Field: "closesAt", Problem: "must be in the future"}
	}
	questions, err := buildQuestions(req.Questions)
	if err != nil {
		return Detail{}, err
	}

	sv := survey{ID: uuid.New(), Title: title, Description: description, ClosesAt: req.ClosesAt.UTC()}
	if err := s.repo.CreateSurvey(ctx, sv, officer.ID, questions); err != nil {
		return Detail{}, fmt.Errorf("create survey: %w", err)
	}
	return s.Get(ctx, sv.ID.String(), nil)
}

func (s *Service) Update(ctx context.Context, rawID string, req UpdateRequest) (Detail, error) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return Detail{}, ErrNotFound
	}

	var update surveyUpdate
	if req.Title != nil {
		title, err := validateTitle(*req.Title)
		if err != nil {
			return Detail{}, err
		}
		update.Title = &title
	}
	if req.Description != nil {
		description, err := validateDescription(*req.Description)
		if err != nil {
			return Detail{}, err
		}
		update.Description = &description
	}
	if req.ClosesAt != nil {
		closesAt := req.ClosesAt.UTC()
		update.ClosesAt = &closesAt
	}
	if req.Questions != nil {
		questions, err := buildQuestions(*req.Questions)
		if err != nil {
			return Detail{}, err
		}
		update.Questions = &questions
	}

	if err := s.repo.UpdateSurvey(ctx, id, update); err != nil {
		return Detail{}, fmt.Errorf("update survey: %w", err)
	}
	return s.Get(ctx, rawID, nil)
}

// AddOption appends to a question's options. Unlike replacing questions it is
// allowed once responses exist, because every stored answer still names an option that exists.
func (s *Service) AddOption(ctx context.Context, rawSurveyID, rawQuestionID string, req AddOptionRequest) (Detail, error) {
	sv, err := s.load(ctx, rawSurveyID)
	if err != nil {
		return Detail{}, fmt.Errorf("add survey option: %w", err)
	}
	questionID, err := uuid.Parse(rawQuestionID)
	if err != nil {
		return Detail{}, ErrQuestionNotFound
	}
	if s.closed(sv) {
		return Detail{}, ErrClosed
	}
	option, err := validateOption(req.Option, "option")
	if err != nil {
		return Detail{}, err
	}

	err = s.repo.AddOption(ctx, sv.ID, questionID, option)
	switch {
	case errors.Is(err, ErrDuplicateOption):
		return Detail{}, &ValidationError{Field: "option", Problem: fmt.Sprintf("%q is already an option", option)}
	case errors.Is(err, ErrTooManyOptions):
		return Detail{}, &ValidationError{Field: "option", Problem: fmt.Sprintf("a question can have at most %d options", maxOptions)}
	case err != nil:
		return Detail{}, fmt.Errorf("add survey option: %w", err)
	}
	return s.Get(ctx, rawSurveyID, nil)
}

func (s *Service) Responses(ctx context.Context, rawID string) ([]OfficerResponse, error) {
	sv, err := s.load(ctx, rawID)
	if err != nil {
		return nil, fmt.Errorf("list survey responses: %w", err)
	}
	rows, err := s.repo.Responses(ctx, sv.ID, nil)
	if err != nil {
		return nil, fmt.Errorf("list survey responses: %w", err)
	}
	return toOfficerResponses(rows), nil
}

func (s *Service) Void(ctx context.Context, rawSurveyID, rawResponseID string, officer auth.User, req VoidRequest) (OfficerResponse, error) {
	sv, err := s.load(ctx, rawSurveyID)
	if err != nil {
		return OfficerResponse{}, fmt.Errorf("void survey response: %w", err)
	}
	responseID, err := uuid.Parse(rawResponseID)
	if err != nil {
		return OfficerResponse{}, ErrResponseNotFound
	}
	reason := strings.TrimSpace(req.Reason)
	if utf8.RuneCountInString(reason) > maxVoidReasonLength {
		return OfficerResponse{}, &ValidationError{Field: "reason", Problem: fmt.Sprintf("must be at most %d characters", maxVoidReasonLength)}
	}

	if err := s.repo.SetVoid(ctx, sv.ID, responseID, req.Voided, officer.ID, reason, s.clock.Now().UTC()); err != nil {
		return OfficerResponse{}, fmt.Errorf("void survey response: %w", err)
	}
	rows, err := s.repo.Responses(ctx, sv.ID, &responseID)
	if err != nil {
		return OfficerResponse{}, fmt.Errorf("void survey response: %w", err)
	}
	if len(rows) == 0 {
		return OfficerResponse{}, ErrResponseNotFound
	}
	return toOfficerResponses(rows)[0], nil
}

type Export struct {
	SurveyID uuid.UUID
	Rows     [][]string
}

func (s *Service) Export(ctx context.Context, rawID string) (Export, error) {
	sv, err := s.load(ctx, rawID)
	if err != nil {
		return Export{}, fmt.Errorf("export survey responses: %w", err)
	}
	questions, err := s.repo.Questions(ctx, sv.ID)
	if err != nil {
		return Export{}, fmt.Errorf("export survey responses: %w", err)
	}
	responses, err := s.repo.Responses(ctx, sv.ID, nil)
	if err != nil {
		return Export{}, fmt.Errorf("export survey responses: %w", err)
	}

	header := []string{"Discord name", "Guild name", "In Discord server", "Submitted", "Voided", "Void reason"}
	for _, q := range questions {
		header = append(header, q.Prompt)
	}
	rows := [][]string{header}
	for _, response := range responses {
		row := []string{
			response.DiscordUsername,
			response.GuildName,
			fmt.Sprint(response.InDiscordServer),
			response.SubmittedAt.Format(time.RFC3339),
			fmt.Sprint(response.VoidedAt != nil),
			response.VoidReason,
		}
		for _, q := range questions {
			var values []string
			for _, a := range response.Answers {
				if a.QuestionID == q.ID {
					values = a.Values
				}
			}
			row = append(row, strings.Join(values, "; "))
		}
		rows = append(rows, row)
	}
	for _, row := range rows {
		for i, cell := range row {
			row[i] = defuseFormula(cell)
		}
	}
	return Export{SurveyID: sv.ID, Rows: rows}, nil
}

// Spreadsheets run cells that start with these characters as formulas, and guild names are free text from the public.
func defuseFormula(cell string) string {
	if cell != "" && strings.ContainsRune("=+-@\t\r", rune(cell[0])) {
		return "'" + cell
	}
	return cell
}

func toOfficerResponses(rows []officerRow) []OfficerResponse {
	responses := make([]OfficerResponse, len(rows))
	for i, row := range rows {
		answers := make([]Answer, len(row.Answers))
		for j, a := range row.Answers {
			answers[j] = Answer{QuestionID: a.QuestionID, Values: a.Values}
		}
		responses[i] = OfficerResponse{
			ID:              row.ID,
			DiscordUsername: row.DiscordUsername,
			GuildName:       row.GuildName,
			InDiscordServer: row.InDiscordServer,
			SubmittedAt:     row.SubmittedAt,
			UpdatedAt:       row.UpdatedAt,
			VoidedAt:        row.VoidedAt,
			VoidedBy:        row.VoidedBy,
			VoidReason:      row.VoidReason,
			Answers:         answers,
		}
	}
	return responses
}

func validateTitle(raw string) (string, error) {
	title := strings.TrimSpace(raw)
	if title == "" {
		return "", &ValidationError{Field: "title", Problem: "is required"}
	}
	if utf8.RuneCountInString(title) > maxTitleLength {
		return "", &ValidationError{Field: "title", Problem: fmt.Sprintf("must be at most %d characters", maxTitleLength)}
	}
	return title, nil
}

func validateDescription(raw string) (string, error) {
	description := strings.TrimSpace(raw)
	if utf8.RuneCountInString(description) > maxDescriptionLength {
		return "", &ValidationError{Field: "description", Problem: fmt.Sprintf("must be at most %d characters", maxDescriptionLength)}
	}
	return description, nil
}

func buildQuestions(inputs []QuestionInput) ([]question, error) {
	if len(inputs) == 0 {
		return nil, &ValidationError{Field: "questions", Problem: "at least one question is required"}
	}
	if len(inputs) > maxQuestions {
		return nil, &ValidationError{Field: "questions", Problem: fmt.Sprintf("at most %d questions are allowed", maxQuestions)}
	}

	questions := make([]question, len(inputs))
	for i, in := range inputs {
		field := func(name string) string { return fmt.Sprintf("questions[%d].%s", i, name) }

		prompt := strings.TrimSpace(in.Prompt)
		if prompt == "" {
			return nil, &ValidationError{Field: field("prompt"), Problem: "is required"}
		}
		if utf8.RuneCountInString(prompt) > maxPromptLength {
			return nil, &ValidationError{Field: field("prompt"), Problem: fmt.Sprintf("must be at most %d characters", maxPromptLength)}
		}
		if in.Type != "single" && in.Type != "multi" {
			return nil, &ValidationError{Field: field("type"), Problem: `must be "single" or "multi"`}
		}

		options := in.Options
		if in.Preset != "" {
			if len(in.Options) > 0 {
				return nil, &ValidationError{Field: field("options"), Problem: "give options or a preset, not both"}
			}
			preset, ok := presets[in.Preset]
			if !ok {
				return nil, &ValidationError{Field: field("preset"), Problem: fmt.Sprintf("unknown preset %q", in.Preset)}
			}
			options = preset
		}
		options, err := validateOptions(options, field("options"))
		if err != nil {
			return nil, err
		}
		questions[i] = question{ID: uuid.New(), Prompt: prompt, Type: in.Type, Required: in.Required, Options: options}
	}
	return questions, nil
}

func validateOption(raw, field string) (string, error) {
	option := strings.TrimSpace(raw)
	if option == "" {
		return "", &ValidationError{Field: field, Problem: "options must not be blank"}
	}
	if utf8.RuneCountInString(option) > maxOptionLength {
		return "", &ValidationError{Field: field, Problem: fmt.Sprintf("options must be at most %d characters each", maxOptionLength)}
	}
	return option, nil
}

func validateOptions(raw []string, field string) ([]string, error) {
	options := make([]string, len(raw))
	for i, rawOption := range raw {
		option, err := validateOption(rawOption, field)
		if err != nil {
			return nil, err
		}
		options[i] = option
	}
	if len(options) < 2 {
		return nil, &ValidationError{Field: field, Problem: "at least two options are required"}
	}
	if len(options) > maxOptions {
		return nil, &ValidationError{Field: field, Problem: fmt.Sprintf("at most %d options are allowed", maxOptions)}
	}
	for i, option := range options {
		if slices.Contains(options[:i], option) {
			return nil, &ValidationError{Field: field, Problem: fmt.Sprintf("%q is listed more than once", option)}
		}
	}
	return options, nil
}
