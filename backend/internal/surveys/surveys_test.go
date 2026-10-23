package surveys_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jmoiron/sqlx"

	"github.com/cageymage/fuzion/backend/internal/testutil"
)

func TestMain(m *testing.M) {
	os.Exit(testutil.Run(m))
}

const (
	openSurveyID   = "aaaaaaaa-0000-0000-0000-000000000001"
	closedSurveyID = "aaaaaaaa-0000-0000-0000-000000000002"

	foreignQuestionID = "bbbbbbbb-0000-0000-0000-000000000009"
)

var (
	closesInTheFuture = testutil.FixedNow.Add(7 * 24 * time.Hour)
	closedInThePast   = testutil.FixedNow.Add(-24 * time.Hour)
)

type questionIDs struct{ class, secondary, nights string }

var (
	openQuestions   = questionIDs{"bbbbbbbb-0000-0000-0000-000000000001", "bbbbbbbb-0000-0000-0000-000000000002", "bbbbbbbb-0000-0000-0000-000000000003"}
	closedQuestions = questionIDs{"cccccccc-0000-0000-0000-000000000001", "cccccccc-0000-0000-0000-000000000002", "cccccccc-0000-0000-0000-000000000003"}
)

// insertSurvey creates a survey with a required single-choice class question, an
// optional single-choice secondary class question and a required multi-choice
// raid nights question.
func insertSurvey(t *testing.T, db *sqlx.DB, id, title string, closesAt time.Time) {
	t.Helper()
	questions := openQuestions
	if id == closedSurveyID {
		questions = closedQuestions
	}
	db.MustExec(`INSERT INTO surveys (id, title, description, closes_at) VALUES ($1, $2, 'Help us plan', $3)`, id, title, closesAt)
	db.MustExec(`
		INSERT INTO survey_questions (id, survey_id, position, prompt, type, required, options) VALUES
		($1, $4, 1, 'Class', 'single', true, '["Mage","Priest","Warrior"]'),
		($2, $4, 2, 'Secondary class', 'single', false, '["Mage","Priest","Warrior"]'),
		($3, $4, 3, 'Raid nights', 'multi', true, '["Tuesday","Wednesday","Thursday"]')`,
		questions.class, questions.secondary, questions.nights, id)
}

func TestListSurveys_ReturnsSurveysLatestDeadlineFirst_WithOpenOrClosedStatus(t *testing.T) {
	// given one survey that is still open and one whose deadline has passed
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, closedSurveyID, "Winter plans", closedInThePast)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)

	// when I list the surveys without logging in
	resp := srv.Get(t, "/api/surveys")

	// then I expect both, the open one first
	resp.RequireStatus(t, 200)
	var got []map[string]any
	resp.DecodeJSON(t, &got)
	want := []map[string]any{
		{"id": openSurveyID, "title": "Spring roster", "closesAt": "2026-03-21T20:00:00Z", "status": "open"},
		{"id": closedSurveyID, "title": "Winter plans", "closesAt": "2026-03-13T20:00:00Z", "status": "closed"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected surveys (-want +got):\n%s", diff)
	}
}

func TestGetSurvey_ReturnsQuestionsInOrderWithoutMyResponse_WhenCallerIsAnonymous(t *testing.T) {
	// given an open survey
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)

	// when I fetch it without logging in
	resp := srv.Get(t, "/api/surveys/"+openSurveyID)

	// then I expect its questions in position order and no response of mine
	resp.RequireStatus(t, 200)
	var got map[string]any
	resp.DecodeJSON(t, &got)
	want := map[string]any{
		"id":          openSurveyID,
		"title":       "Spring roster",
		"description": "Help us plan",
		"closesAt":    "2026-03-21T20:00:00Z",
		"status":      "open",
		"questions": []any{
			map[string]any{"id": openQuestions.class, "prompt": "Class", "type": "single", "required": true, "options": []any{"Mage", "Priest", "Warrior"}},
			map[string]any{"id": openQuestions.secondary, "prompt": "Secondary class", "type": "single", "required": false, "options": []any{"Mage", "Priest", "Warrior"}},
			map[string]any{"id": openQuestions.nights, "prompt": "Raid nights", "type": "multi", "required": true, "options": []any{"Tuesday", "Wednesday", "Thursday"}},
		},
		"myResponse": nil,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected survey (-want +got):\n%s", diff)
	}
}

func TestGetSurvey_IncludesMyResponse_WhenCallerHasAlreadyResponded(t *testing.T) {
	// given an open survey I have already answered
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)
	srv.LoginAs(t, "80351110224678912", "thundermane")
	srv.Put(t, "/api/surveys/"+openSurveyID+"/response", map[string]any{
		"guildName": "Thunder",
		"answers": []map[string]any{
			{"questionId": openQuestions.class, "values": []string{"Mage"}},
			{"questionId": openQuestions.nights, "values": []string{"Tuesday", "Thursday"}},
		},
	}).RequireStatus(t, 200)

	// when I fetch the survey
	resp := srv.Get(t, "/api/surveys/"+openSurveyID)

	// then I expect my earlier answers back
	resp.RequireStatus(t, 200)
	var got struct {
		MyResponse map[string]any `json:"myResponse"`
	}
	resp.DecodeJSON(t, &got)
	want := map[string]any{
		"guildName": "Thunder",
		"answers": []any{
			map[string]any{"questionId": openQuestions.class, "values": []any{"Mage"}},
			map[string]any{"questionId": openQuestions.nights, "values": []any{"Tuesday", "Thursday"}},
		},
	}
	if diff := cmp.Diff(want, got.MyResponse); diff != "" {
		t.Errorf("unexpected response (-want +got):\n%s", diff)
	}
}

func TestGetSurvey_ReturnsNotFound_WhenSurveyDoesNotExist(t *testing.T) {
	// given no surveys
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I fetch a survey id nobody created
	resp := srv.Get(t, "/api/surveys/"+openSurveyID)

	// then I expect a 404
	resp.RequireStatus(t, 404)
	var got map[string]string
	resp.DecodeJSON(t, &got)
	if diff := cmp.Diff(map[string]string{"error": "survey not found"}, got); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
}

func TestGetSurvey_ReturnsNotFound_WhenIdIsNotAUuid(t *testing.T) {
	// given no surveys
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I fetch a survey with an id that is not a uuid
	resp := srv.Get(t, "/api/surveys/not-a-uuid")

	// then I expect a 404
	resp.RequireStatus(t, 404)
}

func TestSubmitResponse_ReturnsUnauthorized_WhenCallerIsNotLoggedIn(t *testing.T) {
	// given an open survey and no session
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)

	// when I submit a response
	resp := srv.Put(t, "/api/surveys/"+openSurveyID+"/response", map[string]any{"answers": []any{}})

	// then I expect a 401
	resp.RequireStatus(t, 401)
}

func TestSubmitResponse_SavesResponseWithDiscordSnapshot_WhenAnswersAreValid(t *testing.T) {
	// given a logged-in member who is in the Discord server and an open survey
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)
	user := srv.LoginAs(t, "80351110224678912", "thundermane")
	db.MustExec(`UPDATE users SET in_discord_server = true WHERE id = $1`, user.ID)

	// when I submit answers and a guild name
	resp := srv.Put(t, "/api/surveys/"+openSurveyID+"/response", map[string]any{
		"guildName": "  Thunder  ",
		"answers": []map[string]any{
			{"questionId": openQuestions.class, "values": []string{"Mage"}},
			{"questionId": openQuestions.secondary, "values": []string{"Priest"}},
			{"questionId": openQuestions.nights, "values": []string{"Tuesday", "Thursday"}},
		},
	})

	// then I expect the response saved with my Discord name and server flag at submit time
	resp.RequireStatus(t, 200)
	type responseRow struct {
		GuildName       string `db:"guild_name"`
		DiscordUsername string `db:"discord_username"`
		InDiscordServer bool   `db:"in_discord_server"`
		Voided          bool   `db:"voided"`
	}
	var rows []responseRow
	if err := db.Select(&rows, `SELECT guild_name, discord_username, in_discord_server, voided_at IS NOT NULL AS voided FROM survey_responses`); err != nil {
		t.Fatalf("select responses: %v", err)
	}
	if diff := cmp.Diff([]responseRow{{GuildName: "Thunder", DiscordUsername: "thundermane", InDiscordServer: true}}, rows); diff != "" {
		t.Errorf("unexpected responses (-want +got):\n%s", diff)
	}

	var answers []struct {
		Prompt string `db:"prompt"`
		Chosen string `db:"chosen"`
	}
	if err := db.Select(&answers, `
		SELECT q.prompt, a.chosen::text AS chosen
		FROM survey_answers a JOIN survey_questions q ON q.id = a.question_id
		ORDER BY q.position`); err != nil {
		t.Fatalf("select answers: %v", err)
	}
	wantAnswers := []struct {
		Prompt string `db:"prompt"`
		Chosen string `db:"chosen"`
	}{
		{Prompt: "Class", Chosen: `["Mage"]`},
		{Prompt: "Secondary class", Chosen: `["Priest"]`},
		{Prompt: "Raid nights", Chosen: `["Tuesday", "Thursday"]`},
	}
	if diff := cmp.Diff(wantAnswers, answers); diff != "" {
		t.Errorf("unexpected answers (-want +got):\n%s", diff)
	}
}

func TestSubmitResponse_StoresNoAnswer_WhenOptionalQuestionIsLeftOut(t *testing.T) {
	// given an open survey
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)
	srv.LoginAs(t, "80351110224678912", "thundermane")

	// when I answer only the required questions
	resp := srv.Put(t, "/api/surveys/"+openSurveyID+"/response", map[string]any{
		"answers": []map[string]any{
			{"questionId": openQuestions.class, "values": []string{"Mage"}},
			{"questionId": openQuestions.nights, "values": []string{"Tuesday"}},
		},
	})

	// then I expect two stored answers and none for the optional question
	resp.RequireStatus(t, 200)
	var prompts []string
	if err := db.Select(&prompts, `
		SELECT q.prompt FROM survey_answers a JOIN survey_questions q ON q.id = a.question_id ORDER BY q.position`); err != nil {
		t.Fatalf("select answers: %v", err)
	}
	if diff := cmp.Diff([]string{"Class", "Raid nights"}, prompts); diff != "" {
		t.Errorf("unexpected answered questions (-want +got):\n%s", diff)
	}
}

func TestSubmitResponse_ReplacesEarlierAnswers_WhenCallerSubmitsAgainBeforeTheDeadline(t *testing.T) {
	// given a member who already answered an open survey
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)
	srv.LoginAs(t, "80351110224678912", "thundermane")
	srv.Put(t, "/api/surveys/"+openSurveyID+"/response", map[string]any{
		"guildName": "Thunder",
		"answers": []map[string]any{
			{"questionId": openQuestions.class, "values": []string{"Mage"}},
			{"questionId": openQuestions.secondary, "values": []string{"Priest"}},
			{"questionId": openQuestions.nights, "values": []string{"Tuesday"}},
		},
	}).RequireStatus(t, 200)

	// when I submit a different set of answers
	resp := srv.Put(t, "/api/surveys/"+openSurveyID+"/response", map[string]any{
		"guildName": "Thundermane",
		"answers": []map[string]any{
			{"questionId": openQuestions.class, "values": []string{"Warrior"}},
			{"questionId": openQuestions.nights, "values": []string{"Wednesday"}},
		},
	})

	// then I expect one response holding only the new answers
	resp.RequireStatus(t, 200)
	var guildNames []string
	if err := db.Select(&guildNames, `SELECT guild_name FROM survey_responses`); err != nil {
		t.Fatalf("select responses: %v", err)
	}
	if diff := cmp.Diff([]string{"Thundermane"}, guildNames); diff != "" {
		t.Errorf("unexpected responses (-want +got):\n%s", diff)
	}
	var answers []string
	if err := db.Select(&answers, `
		SELECT q.prompt || '=' || a.chosen::text FROM survey_answers a JOIN survey_questions q ON q.id = a.question_id ORDER BY q.position`); err != nil {
		t.Fatalf("select answers: %v", err)
	}
	if diff := cmp.Diff([]string{`Class=["Warrior"]`, `Raid nights=["Wednesday"]`}, answers); diff != "" {
		t.Errorf("unexpected answers (-want +got):\n%s", diff)
	}
}

func TestSubmitResponse_ClearsVoid_WhenVoidedMemberSubmitsAgain(t *testing.T) {
	// given a member whose response an officer voided
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)
	user := srv.LoginAs(t, "80351110224678912", "thundermane")
	srv.Put(t, "/api/surveys/"+openSurveyID+"/response", map[string]any{
		"answers": []map[string]any{
			{"questionId": openQuestions.class, "values": []string{"Mage"}},
			{"questionId": openQuestions.nights, "values": []string{"Tuesday"}},
		},
	}).RequireStatus(t, 200)
	db.MustExec(`UPDATE survey_responses SET voided_at = now(), voided_by = $1, void_reason = 'joke answers'`, user.ID)

	// when I submit again
	resp := srv.Put(t, "/api/surveys/"+openSurveyID+"/response", map[string]any{
		"answers": []map[string]any{
			{"questionId": openQuestions.class, "values": []string{"Priest"}},
			{"questionId": openQuestions.nights, "values": []string{"Thursday"}},
		},
	})

	// then I expect the void to be cleared
	resp.RequireStatus(t, 200)
	var voided bool
	if err := db.Get(&voided, `SELECT voided_at IS NOT NULL FROM survey_responses`); err != nil {
		t.Fatalf("select response: %v", err)
	}
	if voided {
		t.Error("expected the resubmitted response to no longer be voided")
	}
}

func TestSubmitResponse_ReturnsConflict_WhenSurveyHasClosed(t *testing.T) {
	// given a survey whose deadline has passed
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, closedSurveyID, "Winter plans", closedInThePast)
	srv.LoginAs(t, "80351110224678912", "thundermane")

	// when I submit a response
	resp := srv.Put(t, "/api/surveys/"+closedSurveyID+"/response", map[string]any{
		"answers": []map[string]any{
			{"questionId": closedQuestions.class, "values": []string{"Mage"}},
			{"questionId": closedQuestions.nights, "values": []string{"Tuesday"}},
		},
	})

	// then I expect a 409 and nothing stored
	resp.RequireStatus(t, 409)
	var got map[string]string
	resp.DecodeJSON(t, &got)
	if diff := cmp.Diff(map[string]string{"error": "survey is closed"}, got); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
	var responses int
	if err := db.Get(&responses, `SELECT count(*) FROM survey_responses`); err != nil {
		t.Fatalf("count responses: %v", err)
	}
	if responses != 0 {
		t.Errorf("expected no responses, found %d", responses)
	}
}

func TestSubmitResponse_ReturnsNotFound_WhenSurveyDoesNotExist(t *testing.T) {
	// given a logged-in member and no surveys
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, "80351110224678912", "thundermane")

	// when I submit a response to a survey nobody created
	resp := srv.Put(t, "/api/surveys/"+openSurveyID+"/response", map[string]any{"answers": []any{}})

	// then I expect a 404
	resp.RequireStatus(t, 404)
}

func TestSubmitResponse_ReturnsBadRequest_WhenRequestIsRejected(t *testing.T) {
	tests := []struct {
		name      string
		guildName string
		answers   []map[string]any
		wantError string
	}{
		{
			name:      "a required question is unanswered",
			answers:   []map[string]any{{"questionId": openQuestions.class, "values": []string{"Mage"}}},
			wantError: "answers: Raid nights is required",
		},
		{
			name: "an answer is not one of the question's options",
			answers: []map[string]any{
				{"questionId": openQuestions.class, "values": []string{"Shaman"}},
				{"questionId": openQuestions.nights, "values": []string{"Tuesday"}},
			},
			wantError: `answers: "Shaman" is not an option for Class`,
		},
		{
			name: "a single choice question has two values",
			answers: []map[string]any{
				{"questionId": openQuestions.class, "values": []string{"Mage", "Priest"}},
				{"questionId": openQuestions.nights, "values": []string{"Tuesday"}},
			},
			wantError: "answers: Class takes exactly one value",
		},
		{
			name: "a question belongs to a different survey",
			answers: []map[string]any{
				{"questionId": openQuestions.class, "values": []string{"Mage"}},
				{"questionId": openQuestions.nights, "values": []string{"Tuesday"}},
				{"questionId": foreignQuestionID, "values": []string{"Mage"}},
			},
			wantError: "answers: " + foreignQuestionID + " is not a question of this survey",
		},
		{
			name: "the same question is answered twice",
			answers: []map[string]any{
				{"questionId": openQuestions.class, "values": []string{"Mage"}},
				{"questionId": openQuestions.class, "values": []string{"Priest"}},
				{"questionId": openQuestions.nights, "values": []string{"Tuesday"}},
			},
			wantError: "answers: Class is answered more than once",
		},
		{
			name: "a multi choice answer repeats a value",
			answers: []map[string]any{
				{"questionId": openQuestions.class, "values": []string{"Mage"}},
				{"questionId": openQuestions.nights, "values": []string{"Tuesday", "Tuesday"}},
			},
			wantError: `answers: "Tuesday" is chosen more than once for Raid nights`,
		},
		{
			name:      "the guild name is longer than 64 characters",
			guildName: strings.Repeat("a", 65),
			answers: []map[string]any{
				{"questionId": openQuestions.class, "values": []string{"Mage"}},
				{"questionId": openQuestions.nights, "values": []string{"Tuesday"}},
			},
			wantError: "guildName: must be at most 64 characters",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given an open survey and a logged-in member
			db := testutil.DB(t)
			srv := testutil.NewServer(t, db)
			insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)
			srv.LoginAs(t, "80351110224678912", "thundermane")

			// when I submit the response
			resp := srv.Put(t, "/api/surveys/"+openSurveyID+"/response", map[string]any{"guildName": tt.guildName, "answers": tt.answers})

			// then I expect a 400 naming the problem and nothing stored
			resp.RequireStatus(t, 400)
			var got map[string]string
			resp.DecodeJSON(t, &got)
			if diff := cmp.Diff(map[string]string{"error": tt.wantError}, got); diff != "" {
				t.Errorf("unexpected error body (-want +got):\n%s", diff)
			}
			var responses int
			if err := db.Get(&responses, `SELECT count(*) FROM survey_responses`); err != nil {
				t.Fatalf("count responses: %v", err)
			}
			if responses != 0 {
				t.Errorf("expected no responses, found %d", responses)
			}
		})
	}
}

func TestGetResults_ReturnsForbidden_WhenSurveyIsStillOpen(t *testing.T) {
	// given an open survey
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)

	// when I ask for the results
	resp := srv.Get(t, "/api/surveys/"+openSurveyID+"/results")

	// then I expect a 403 saying they are not available yet
	resp.RequireStatus(t, 403)
	var got map[string]string
	resp.DecodeJSON(t, &got)
	if diff := cmp.Diff(map[string]string{"error": "results are not available until the survey closes"}, got); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
}

func TestGetResults_ReturnsNotFound_WhenSurveyDoesNotExist(t *testing.T) {
	// given no surveys
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)

	// when I ask for the results of a survey nobody created
	resp := srv.Get(t, "/api/surveys/"+closedSurveyID+"/results")

	// then I expect a 404
	resp.RequireStatus(t, 404)
}

// insertClosedResponse stores a response to the closed survey for a fresh user.
func insertClosedResponse(t *testing.T, db *sqlx.DB, discordID string, voided bool, class, nights string) {
	t.Helper()
	var userID string
	if err := db.Get(&userID, `INSERT INTO users (id, discord_id, username) VALUES (gen_random_uuid(), $1, $1) RETURNING id`, discordID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	var responseID string
	if err := db.Get(&responseID, `
		INSERT INTO survey_responses (survey_id, user_id, discord_username, in_discord_server, voided_at)
		VALUES ($1, $2, $3, true, CASE WHEN $4 THEN now() END) RETURNING id`, closedSurveyID, userID, discordID, voided); err != nil {
		t.Fatalf("insert response: %v", err)
	}
	db.MustExec(`INSERT INTO survey_answers (response_id, question_id, chosen) VALUES ($1, $2, $3), ($1, $4, $5)`,
		responseID, closedQuestions.class, `["`+class+`"]`, closedQuestions.nights, nights)
}

func TestGetResults_ReturnsCountsPerOptionIncludingZeroes_WhenSurveyHasClosed(t *testing.T) {
	// given a closed survey with three responses
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, closedSurveyID, "Winter plans", closedInThePast)
	insertClosedResponse(t, db, "ana", false, "Mage", `["Tuesday","Wednesday"]`)
	insertClosedResponse(t, db, "ben", false, "Mage", `["Tuesday"]`)
	insertClosedResponse(t, db, "cy", false, "Warrior", `["Thursday","Tuesday"]`)

	// when I ask for the results without logging in
	resp := srv.Get(t, "/api/surveys/"+closedSurveyID+"/results")

	// then I expect turnout and per-option counts, with unchosen options at zero
	resp.RequireStatus(t, 200)
	var got map[string]any
	resp.DecodeJSON(t, &got)
	want := map[string]any{
		"surveyId": closedSurveyID,
		"title":    "Winter plans",
		"turnout":  float64(3),
		"questions": []any{
			map[string]any{"id": closedQuestions.class, "prompt": "Class", "type": "single", "options": []any{
				map[string]any{"label": "Mage", "count": float64(2)},
				map[string]any{"label": "Priest", "count": float64(0)},
				map[string]any{"label": "Warrior", "count": float64(1)},
			}},
			map[string]any{"id": closedQuestions.secondary, "prompt": "Secondary class", "type": "single", "options": []any{
				map[string]any{"label": "Mage", "count": float64(0)},
				map[string]any{"label": "Priest", "count": float64(0)},
				map[string]any{"label": "Warrior", "count": float64(0)},
			}},
			map[string]any{"id": closedQuestions.nights, "prompt": "Raid nights", "type": "multi", "options": []any{
				map[string]any{"label": "Tuesday", "count": float64(3)},
				map[string]any{"label": "Wednesday", "count": float64(1)},
				map[string]any{"label": "Thursday", "count": float64(1)},
			}},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected results (-want +got):\n%s", diff)
	}
}

func TestGetResults_LeavesVoidedResponsesOut_WhenOneResponseWasVoided(t *testing.T) {
	// given a closed survey where one of two responses was voided
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, closedSurveyID, "Winter plans", closedInThePast)
	insertClosedResponse(t, db, "ana", false, "Mage", `["Tuesday"]`)
	insertClosedResponse(t, db, "troll", true, "Priest", `["Wednesday"]`)

	// when I ask for the results
	resp := srv.Get(t, "/api/surveys/"+closedSurveyID+"/results")

	// then I expect turnout of one and the voided choices uncounted
	resp.RequireStatus(t, 200)
	var got struct {
		Turnout   int `json:"turnout"`
		Questions []struct {
			Prompt  string `json:"prompt"`
			Options []struct {
				Label string `json:"label"`
				Count int    `json:"count"`
			} `json:"options"`
		} `json:"questions"`
	}
	resp.DecodeJSON(t, &got)
	if got.Turnout != 1 {
		t.Errorf("expected turnout 1, got %d", got.Turnout)
	}
	counts := map[string]int{}
	for _, option := range got.Questions[0].Options {
		counts[option.Label] = option.Count
	}
	if diff := cmp.Diff(map[string]int{"Mage": 1, "Priest": 0, "Warrior": 0}, counts); diff != "" {
		t.Errorf("unexpected class counts (-want +got):\n%s", diff)
	}
}
