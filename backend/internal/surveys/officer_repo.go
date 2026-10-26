package surveys

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var (
	ErrQuestionsLocked  = errors.New("questions cannot change once responses exist")
	ErrResponseNotFound = errors.New("response not found")
	ErrQuestionNotFound = errors.New("question not found")
	ErrDuplicateOption  = errors.New("option already exists")
	ErrTooManyOptions   = errors.New("question has the maximum number of options")
)

type surveyUpdate struct {
	Title       *string
	Description *string
	ClosesAt    *time.Time
	Questions   *[]question
}

type officerRow struct {
	ID              uuid.UUID  `db:"id"`
	DiscordUsername string     `db:"discord_username"`
	GuildName       string     `db:"guild_name"`
	InDiscordServer bool       `db:"in_discord_server"`
	SubmittedAt     time.Time  `db:"submitted_at"`
	UpdatedAt       time.Time  `db:"updated_at"`
	VoidedAt        *time.Time `db:"voided_at"`
	VoidedBy        *string    `db:"voided_by"`
	VoidReason      string     `db:"void_reason"`
	Answers         []answer
}

func (r *Repo) CreateSurvey(ctx context.Context, s survey, createdBy uuid.UUID, questions []question) (err error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create survey: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	const insert = `INSERT INTO surveys (id, title, description, closes_at, created_by) VALUES ($1, $2, $3, $4, $5)`
	if _, err := tx.ExecContext(ctx, insert, s.ID, s.Title, s.Description, s.ClosesAt, createdBy); err != nil {
		return fmt.Errorf("insert survey: %w", err)
	}
	if err := insertQuestions(ctx, tx, s.ID, questions); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create survey: %w", err)
	}
	return nil
}

func insertQuestions(ctx context.Context, tx *sqlx.Tx, surveyID uuid.UUID, questions []question) error {
	const insert = `
		INSERT INTO survey_questions (id, survey_id, position, prompt, type, required, options)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	for i, q := range questions {
		if _, err := tx.ExecContext(ctx, insert, q.ID, surveyID, i+1, q.Prompt, q.Type, q.Required, q.Options); err != nil {
			return fmt.Errorf("insert survey question: %w", err)
		}
	}
	return nil
}

// UpdateSurvey applies the non-nil fields. Replacing questions is refused once
// any response exists, because the stored answers refer to the old ones.
func (r *Repo) UpdateSurvey(ctx context.Context, id uuid.UUID, update surveyUpdate) (err error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin update survey: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var locked uuid.UUID
	err = tx.GetContext(ctx, &locked, `SELECT id FROM surveys WHERE id = $1 FOR UPDATE`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("lock survey: %w", err)
	}

	if update.Questions != nil {
		var responses int
		if err := tx.GetContext(ctx, &responses, `SELECT count(*) FROM survey_responses WHERE survey_id = $1`, id); err != nil {
			return fmt.Errorf("count survey responses: %w", err)
		}
		if responses > 0 {
			return ErrQuestionsLocked
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM survey_questions WHERE survey_id = $1`, id); err != nil {
			return fmt.Errorf("clear survey questions: %w", err)
		}
		if err := insertQuestions(ctx, tx, id, *update.Questions); err != nil {
			return err
		}
	}

	const query = `
		UPDATE surveys
		SET title = COALESCE($2, title),
		    description = COALESCE($3, description),
		    closes_at = COALESCE($4, closes_at)
		WHERE id = $1`
	if _, err := tx.ExecContext(ctx, query, id, update.Title, update.Description, update.ClosesAt); err != nil {
		return fmt.Errorf("update survey: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit update survey: %w", err)
	}
	return nil
}

func (r *Repo) AddOption(ctx context.Context, surveyID, questionID uuid.UUID, option string) (err error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin add survey option: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var options stringList
	err = tx.GetContext(ctx, &options, `SELECT options FROM survey_questions WHERE id = $1 AND survey_id = $2 FOR UPDATE`, questionID, surveyID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrQuestionNotFound
	}
	if err != nil {
		return fmt.Errorf("lock survey question: %w", err)
	}
	if slices.Contains(options, option) {
		return ErrDuplicateOption
	}
	if len(options) >= maxOptions {
		return ErrTooManyOptions
	}

	if _, err := tx.ExecContext(ctx, `UPDATE survey_questions SET options = $2 WHERE id = $1`, questionID, append(options, option)); err != nil {
		return fmt.Errorf("append survey option: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit add survey option: %w", err)
	}
	return nil
}

// Responses returns the survey's responses, voided ones included, oldest first.
// A non-nil responseID narrows it to that one response.
func (r *Repo) Responses(ctx context.Context, surveyID uuid.UUID, responseID *uuid.UUID) ([]officerRow, error) {
	const query = `
		SELECT r.id, r.discord_username, r.guild_name, u.in_discord_server, r.submitted_at, r.updated_at,
		       r.voided_at, v.username AS voided_by, r.void_reason
		FROM survey_responses r
		JOIN users u ON u.id = r.user_id
		LEFT JOIN users v ON v.id = r.voided_by
		WHERE r.survey_id = $1 AND ($2::uuid IS NULL OR r.id = $2)
		ORDER BY r.submitted_at, r.id`
	var rows []officerRow
	if err := r.db.SelectContext(ctx, &rows, query, surveyID, responseID); err != nil {
		return nil, fmt.Errorf("list survey responses: %w", err)
	}

	const answersQuery = `
		SELECT a.response_id, a.question_id, a.chosen
		FROM survey_answers a
		JOIN survey_responses r ON r.id = a.response_id
		JOIN survey_questions q ON q.id = a.question_id
		WHERE r.survey_id = $1 AND ($2::uuid IS NULL OR r.id = $2)
		ORDER BY q.position`
	var answerRows []struct {
		ResponseID uuid.UUID  `db:"response_id"`
		QuestionID uuid.UUID  `db:"question_id"`
		Chosen     stringList `db:"chosen"`
	}
	if err := r.db.SelectContext(ctx, &answerRows, answersQuery, surveyID, responseID); err != nil {
		return nil, fmt.Errorf("list survey response answers: %w", err)
	}

	byResponse := make(map[uuid.UUID][]answer, len(rows))
	for _, a := range answerRows {
		byResponse[a.ResponseID] = append(byResponse[a.ResponseID], answer{QuestionID: a.QuestionID, Values: a.Chosen})
	}
	for i := range rows {
		rows[i].SubmittedAt = rows[i].SubmittedAt.UTC()
		rows[i].UpdatedAt = rows[i].UpdatedAt.UTC()
		if rows[i].VoidedAt != nil {
			voidedAt := rows[i].VoidedAt.UTC()
			rows[i].VoidedAt = &voidedAt
		}
		rows[i].Answers = byResponse[rows[i].ID]
	}
	return rows, nil
}

func (r *Repo) SetVoid(ctx context.Context, surveyID, responseID uuid.UUID, voided bool, by uuid.UUID, reason string, at time.Time) error {
	const query = `
		UPDATE survey_responses
		SET voided_at = CASE WHEN $3::boolean THEN $4::timestamptz END,
		    voided_by = CASE WHEN $3::boolean THEN $5::uuid END,
		    void_reason = CASE WHEN $3::boolean THEN $6::text ELSE '' END
		WHERE id = $1 AND survey_id = $2`
	result, err := r.db.ExecContext(ctx, query, responseID, surveyID, voided, at, by, reason)
	if err != nil {
		return fmt.Errorf("set survey response void: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count voided survey responses: %w", err)
	}
	if affected == 0 {
		return ErrResponseNotFound
	}
	return nil
}
