package surveys

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var ErrNotFound = errors.New("survey not found")

type stringList []string

func (l *stringList) Scan(src any) error {
	var raw []byte
	switch v := src.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("scan string list from %T", src)
	}
	return json.Unmarshal(raw, l)
}

func (l stringList) Value() (driver.Value, error) {
	if l == nil {
		l = stringList{}
	}
	encoded, err := json.Marshal([]string(l))
	if err != nil {
		return nil, err
	}
	return string(encoded), nil
}

type survey struct {
	ID          uuid.UUID `db:"id"`
	Title       string    `db:"title"`
	Description string    `db:"description"`
	ClosesAt    time.Time `db:"closes_at"`
}

type question struct {
	ID       uuid.UUID  `db:"id"`
	Prompt   string     `db:"prompt"`
	Type     string     `db:"type"`
	Required bool       `db:"required"`
	Options  stringList `db:"options"`
}

type answer struct {
	QuestionID uuid.UUID
	Values     []string
}

type storedResponse struct {
	GuildName string
	Answers   []answer
}

type responseInput struct {
	SurveyID        uuid.UUID
	UserID          uuid.UUID
	GuildName       string
	DiscordUsername string
	InDiscordServer bool
	Answers         []answer
}

type optionCount struct {
	QuestionID uuid.UUID
	Label      string
	Count      int
}

type Repo struct {
	db *sqlx.DB
}

func NewRepo(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) ListSurveys(ctx context.Context) ([]survey, error) {
	const query = `SELECT id, title, description, closes_at FROM surveys ORDER BY closes_at DESC, id`

	var surveys []survey
	if err := r.db.SelectContext(ctx, &surveys, query); err != nil {
		return nil, fmt.Errorf("list surveys: %w", err)
	}
	for i := range surveys {
		surveys[i].ClosesAt = surveys[i].ClosesAt.UTC()
	}
	return surveys, nil
}

func (r *Repo) GetSurvey(ctx context.Context, id uuid.UUID) (survey, error) {
	const query = `SELECT id, title, description, closes_at FROM surveys WHERE id = $1`

	var s survey
	err := r.db.GetContext(ctx, &s, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return survey{}, ErrNotFound
	}
	if err != nil {
		return survey{}, fmt.Errorf("get survey: %w", err)
	}
	s.ClosesAt = s.ClosesAt.UTC()
	return s, nil
}

func (r *Repo) Questions(ctx context.Context, surveyID uuid.UUID) ([]question, error) {
	const query = `
		SELECT id, prompt, type, required, options
		FROM survey_questions
		WHERE survey_id = $1
		ORDER BY position`

	var questions []question
	if err := r.db.SelectContext(ctx, &questions, query, surveyID); err != nil {
		return nil, fmt.Errorf("list survey questions: %w", err)
	}
	return questions, nil
}

// ResponseOf returns nil when the user has not responded to the survey.
func (r *Repo) ResponseOf(ctx context.Context, surveyID, userID uuid.UUID) (*storedResponse, error) {
	var guildName string
	err := r.db.GetContext(ctx, &guildName, `SELECT guild_name FROM survey_responses WHERE survey_id = $1 AND user_id = $2`, surveyID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get survey response: %w", err)
	}

	const query = `
		SELECT a.question_id, a.chosen
		FROM survey_answers a
		JOIN survey_responses r ON r.id = a.response_id
		JOIN survey_questions q ON q.id = a.question_id
		WHERE r.survey_id = $1 AND r.user_id = $2
		ORDER BY q.position`
	var rows []struct {
		QuestionID uuid.UUID  `db:"question_id"`
		Chosen     stringList `db:"chosen"`
	}
	if err := r.db.SelectContext(ctx, &rows, query, surveyID, userID); err != nil {
		return nil, fmt.Errorf("get survey answers: %w", err)
	}

	response := &storedResponse{GuildName: guildName, Answers: make([]answer, len(rows))}
	for i, row := range rows {
		response.Answers[i] = answer{QuestionID: row.QuestionID, Values: row.Chosen}
	}
	return response, nil
}

// SaveResponse replaces the user's response to a survey. Saving over a voided
// response clears the void: a resubmission is a new submission for officers to judge.
func (r *Repo) SaveResponse(ctx context.Context, in responseInput) (err error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin save survey response: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	const upsert = `
		INSERT INTO survey_responses (survey_id, user_id, guild_name, discord_username, in_discord_server)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (survey_id, user_id) DO UPDATE
			SET guild_name = EXCLUDED.guild_name,
			    discord_username = EXCLUDED.discord_username,
			    in_discord_server = EXCLUDED.in_discord_server,
			    voided_at = NULL,
			    voided_by = NULL,
			    void_reason = '',
			    updated_at = now()
		RETURNING id`
	var responseID uuid.UUID
	if err := tx.GetContext(ctx, &responseID, upsert, in.SurveyID, in.UserID, in.GuildName, in.DiscordUsername, in.InDiscordServer); err != nil {
		return fmt.Errorf("upsert survey response: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM survey_answers WHERE response_id = $1`, responseID); err != nil {
		return fmt.Errorf("clear survey answers: %w", err)
	}
	for _, a := range in.Answers {
		if _, err := tx.ExecContext(ctx, `INSERT INTO survey_answers (response_id, question_id, chosen) VALUES ($1, $2, $3)`,
			responseID, a.QuestionID, stringList(a.Values)); err != nil {
			return fmt.Errorf("insert survey answer: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit survey response: %w", err)
	}
	return nil
}

func (r *Repo) Tally(ctx context.Context, surveyID uuid.UUID) (turnout int, counts []optionCount, err error) {
	const turnoutQuery = `SELECT count(*) FROM survey_responses WHERE survey_id = $1 AND voided_at IS NULL`
	if err := r.db.GetContext(ctx, &turnout, turnoutQuery, surveyID); err != nil {
		return 0, nil, fmt.Errorf("count survey responses: %w", err)
	}

	const countsQuery = `
		SELECT a.question_id, v.label, count(*) AS count
		FROM survey_answers a
		JOIN survey_responses r ON r.id = a.response_id
		CROSS JOIN LATERAL jsonb_array_elements_text(a.chosen) AS v(label)
		WHERE r.survey_id = $1 AND r.voided_at IS NULL
		GROUP BY a.question_id, v.label`
	var rows []struct {
		QuestionID uuid.UUID `db:"question_id"`
		Label      string    `db:"label"`
		Count      int       `db:"count"`
	}
	if err := r.db.SelectContext(ctx, &rows, countsQuery, surveyID); err != nil {
		return 0, nil, fmt.Errorf("count survey answers: %w", err)
	}
	counts = make([]optionCount, len(rows))
	for i, row := range rows {
		counts[i] = optionCount{QuestionID: row.QuestionID, Label: row.Label, Count: row.Count}
	}
	return turnout, counts, nil
}
