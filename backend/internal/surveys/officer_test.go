package surveys_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jmoiron/sqlx"

	"github.com/cageymage/fuzion/backend/internal/testutil"
)

const (
	officerDiscordID = "1001"
	memberDiscordID  = "2002"
)

func validCreateBody() map[string]any {
	return map[string]any{
		"title":       "Spring roster",
		"description": "Help us plan",
		"closesAt":    "2026-03-16T20:00:00Z",
		"questions": []map[string]any{
			{"prompt": "Class", "type": "single", "required": true, "preset": "wow-classes"},
			{"prompt": "Raid nights", "type": "multi", "required": false, "options": []string{"Tuesday", "Thursday"}},
		},
	}
}

func TestCreateSurvey_ReturnsForbidden_WhenCallerIsOnlyAMember(t *testing.T) {
	// given a logged-in member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, memberDiscordID, "member")

	// when I create a survey
	resp := srv.Post(t, "/api/surveys", validCreateBody())

	// then I expect a 403 and no survey
	resp.RequireStatus(t, 403)
	var surveys int
	if err := db.Get(&surveys, `SELECT count(*) FROM surveys`); err != nil {
		t.Fatalf("count surveys: %v", err)
	}
	if surveys != 0 {
		t.Errorf("expected no surveys, found %d", surveys)
	}
}

func TestCreateSurvey_CreatesSurveyWithQuestionsInOrder_WhenOfficerSendsAValidPayload(t *testing.T) {
	// given a logged-in officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	officer := srv.LoginAs(t, officerDiscordID, "officer1", testutil.AsOfficer())

	// when I create a survey
	resp := srv.Post(t, "/api/surveys", validCreateBody())

	// then I expect a 201 with the survey as respondents will see it, and the officer recorded as its creator
	resp.RequireStatus(t, 201)
	var got map[string]any
	resp.DecodeJSON(t, &got)
	surveyID, _ := got["id"].(string)
	got["id"] = "<survey id>"
	for _, q := range got["questions"].([]any) {
		if id, _ := q.(map[string]any)["id"].(string); id == "" {
			t.Error("expected every question to have an id")
		}
		q.(map[string]any)["id"] = "<question id>"
	}
	want := map[string]any{
		"id":          "<survey id>",
		"title":       "Spring roster",
		"description": "Help us plan",
		"closesAt":    "2026-03-16T20:00:00Z",
		"status":      "open",
		"questions": []any{
			map[string]any{"id": "<question id>", "prompt": "Class", "type": "single", "required": true, "options": []any{"Druid", "Hunter", "Mage", "Paladin", "Priest", "Rogue", "Shaman", "Warlock", "Warrior"}},
			map[string]any{"id": "<question id>", "prompt": "Raid nights", "type": "multi", "required": false, "options": []any{"Tuesday", "Thursday"}},
		},
		"myResponse": nil,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected survey (-want +got):\n%s", diff)
	}

	var createdBy string
	if err := db.Get(&createdBy, `SELECT created_by FROM surveys WHERE id = $1`, surveyID); err != nil {
		t.Fatalf("select created_by: %v", err)
	}
	if createdBy != officer.ID.String() {
		t.Errorf("expected created_by %s, got %s", officer.ID, createdBy)
	}
}

func TestCreateSurvey_ExpandsPresetOptions_WhenQuestionUsesAPreset(t *testing.T) {
	tests := []struct {
		preset      string
		wantOptions []any
	}{
		{"wow-classes", []any{"Druid", "Hunter", "Mage", "Paladin", "Priest", "Rogue", "Shaman", "Warlock", "Warrior"}},
		{"wow-roles", []any{"Tank", "Healer", "Melee DPS", "Ranged DPS"}},
		{"weekdays", []any{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}},
	}
	for _, tt := range tests {
		t.Run(tt.preset, func(t *testing.T) {
			// given a logged-in officer
			db := testutil.DB(t)
			srv := testutil.NewServer(t, db)
			srv.LoginAs(t, officerDiscordID, "officer1", testutil.AsOfficer())

			// when I create a survey whose only question uses the preset
			body := validCreateBody()
			body["questions"] = []map[string]any{{"prompt": "Pick", "type": "multi", "required": true, "preset": tt.preset}}
			resp := srv.Post(t, "/api/surveys", body)

			// then I expect the preset's options on the question
			resp.RequireStatus(t, 201)
			var got struct {
				Questions []struct {
					Options []any `json:"options"`
				} `json:"questions"`
			}
			resp.DecodeJSON(t, &got)
			if diff := cmp.Diff(tt.wantOptions, got.Questions[0].Options); diff != "" {
				t.Errorf("unexpected options (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCreateSurvey_ReturnsBadRequest_WhenPayloadIsRejected(t *testing.T) {
	question := func(overrides map[string]any) []map[string]any {
		q := map[string]any{"prompt": "Class", "type": "single", "required": true, "options": []string{"Mage", "Priest"}}
		for k, v := range overrides {
			if v == nil {
				delete(q, k)
				continue
			}
			q[k] = v
		}
		return []map[string]any{q}
	}
	tests := []struct {
		name      string
		overrides map[string]any
		wantError string
	}{
		{"the title is blank", map[string]any{"title": "  "}, "title: is required"},
		{"the title is longer than 120 characters", map[string]any{"title": strings.Repeat("a", 121)}, "title: must be at most 120 characters"},
		{"the deadline is missing", map[string]any{"closesAt": nil}, "closesAt: is required"},
		{"the deadline is already past", map[string]any{"closesAt": "2026-03-13T20:00:00Z"}, "closesAt: must be in the future"},
		{"there are no questions", map[string]any{"questions": []map[string]any{}}, "questions: at least one question is required"},
		{"a question prompt is blank", map[string]any{"questions": question(map[string]any{"prompt": ""})}, "questions[0].prompt: is required"},
		{"a question type is unknown", map[string]any{"questions": question(map[string]any{"type": "ranked"})}, `questions[0].type: must be "single" or "multi"`},
		{"a question names an unknown preset", map[string]any{"questions": question(map[string]any{"options": nil, "preset": "wow-retail"})}, `questions[0].preset: unknown preset "wow-retail"`},
		{"a question gives both options and a preset", map[string]any{"questions": question(map[string]any{"preset": "wow-classes"})}, "questions[0].options: give options or a preset, not both"},
		{"a question has fewer than two options", map[string]any{"questions": question(map[string]any{"options": []string{"Mage"}})}, "questions[0].options: at least two options are required"},
		{"a question lists an option twice", map[string]any{"questions": question(map[string]any{"options": []string{"Mage", "Mage"}})}, `questions[0].options: "Mage" is listed more than once`},
		{"a question has a blank option", map[string]any{"questions": question(map[string]any{"options": []string{"Mage", " "}})}, "questions[0].options: options must not be blank"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given a logged-in officer
			db := testutil.DB(t)
			srv := testutil.NewServer(t, db)
			srv.LoginAs(t, officerDiscordID, "officer1", testutil.AsOfficer())

			// when I create a survey with that problem
			body := validCreateBody()
			for k, v := range tt.overrides {
				if v == nil {
					delete(body, k)
					continue
				}
				body[k] = v
			}
			resp := srv.Post(t, "/api/surveys", body)

			// then I expect a 400 naming it and no survey stored
			resp.RequireStatus(t, 400)
			var got map[string]string
			resp.DecodeJSON(t, &got)
			if diff := cmp.Diff(map[string]string{"error": tt.wantError}, got); diff != "" {
				t.Errorf("unexpected error body (-want +got):\n%s", diff)
			}
			var surveys int
			if err := db.Get(&surveys, `SELECT count(*) FROM surveys`); err != nil {
				t.Fatalf("count surveys: %v", err)
			}
			if surveys != 0 {
				t.Errorf("expected no surveys, found %d", surveys)
			}
		})
	}
}

func TestUpdateSurvey_ReturnsForbidden_WhenCallerIsOnlyAMember(t *testing.T) {
	// given an open survey and a logged-in member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)
	srv.LoginAs(t, memberDiscordID, "member")

	// when I edit its title
	resp := srv.Patch(t, "/api/surveys/"+openSurveyID, map[string]any{"title": "Hijacked"})

	// then I expect a 403 and the title unchanged
	resp.RequireStatus(t, 403)
	var title string
	if err := db.Get(&title, `SELECT title FROM surveys`); err != nil {
		t.Fatalf("select title: %v", err)
	}
	if title != "Spring roster" {
		t.Errorf("expected the title to stay %q, got %q", "Spring roster", title)
	}
}

func TestUpdateSurvey_ChangesOnlyTheFieldsSent_WhenOfficerSendsATitle(t *testing.T) {
	// given an open survey and a logged-in officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)
	srv.LoginAs(t, officerDiscordID, "officer1", testutil.AsOfficer())

	// when I send only a new title
	resp := srv.Patch(t, "/api/surveys/"+openSurveyID, map[string]any{"title": "Spring roster 2"})

	// then I expect the new title with the description, deadline and questions untouched
	resp.RequireStatus(t, 200)
	var got map[string]any
	resp.DecodeJSON(t, &got)
	want := map[string]any{
		"id":          openSurveyID,
		"title":       "Spring roster 2",
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

func TestUpdateSurvey_ClosesTheSurvey_WhenOfficerMovesTheDeadlineToNow(t *testing.T) {
	// given an open survey and a logged-in officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)
	srv.LoginAs(t, officerDiscordID, "officer1", testutil.AsOfficer())

	// when I set the deadline to the current time
	resp := srv.Patch(t, "/api/surveys/"+openSurveyID, map[string]any{"closesAt": "2026-03-14T20:00:00Z"})

	// then I expect the survey to be closed
	resp.RequireStatus(t, 200)
	var got struct {
		Status   string `json:"status"`
		ClosesAt string `json:"closesAt"`
	}
	resp.DecodeJSON(t, &got)
	if diff := cmp.Diff("closed 2026-03-14T20:00:00Z", got.Status+" "+got.ClosesAt); diff != "" {
		t.Errorf("unexpected status (-want +got):\n%s", diff)
	}
}

func TestUpdateSurvey_ReplacesQuestions_WhenSurveyHasNoResponses(t *testing.T) {
	// given an open survey nobody has answered and a logged-in officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)
	srv.LoginAs(t, officerDiscordID, "officer1", testutil.AsOfficer())

	// when I send a new list of questions
	resp := srv.Patch(t, "/api/surveys/"+openSurveyID, map[string]any{
		"questions": []map[string]any{{"prompt": "Role", "type": "single", "required": true, "preset": "wow-roles"}},
	})

	// then I expect only the new question to remain
	resp.RequireStatus(t, 200)
	var prompts []string
	if err := db.Select(&prompts, `SELECT prompt FROM survey_questions ORDER BY position`); err != nil {
		t.Fatalf("select questions: %v", err)
	}
	if diff := cmp.Diff([]string{"Role"}, prompts); diff != "" {
		t.Errorf("unexpected questions (-want +got):\n%s", diff)
	}
}

func TestUpdateSurvey_ReturnsConflict_WhenQuestionsChangeAfterResponsesExist(t *testing.T) {
	// given an open survey with one response and a logged-in officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)
	insertResponse(t, db, openSurveyID, "ana", "Anaria", true, "2026-03-10T12:00:00Z")
	srv.LoginAs(t, officerDiscordID, "officer1", testutil.AsOfficer())

	// when I try to replace the questions
	resp := srv.Patch(t, "/api/surveys/"+openSurveyID, map[string]any{
		"questions": []map[string]any{{"prompt": "Role", "type": "single", "required": true, "preset": "wow-roles"}},
	})

	// then I expect a 409 and the original questions kept
	resp.RequireStatus(t, 409)
	var got map[string]string
	resp.DecodeJSON(t, &got)
	if diff := cmp.Diff(map[string]string{"error": "questions cannot change once responses exist"}, got); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
	var questions int
	if err := db.Get(&questions, `SELECT count(*) FROM survey_questions`); err != nil {
		t.Fatalf("count questions: %v", err)
	}
	if questions != 3 {
		t.Errorf("expected the 3 original questions, found %d", questions)
	}
}

func TestUpdateSurvey_ReturnsNotFound_WhenSurveyDoesNotExist(t *testing.T) {
	// given a logged-in officer and no surveys
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, officerDiscordID, "officer1", testutil.AsOfficer())

	// when I edit a survey nobody created
	resp := srv.Patch(t, "/api/surveys/"+openSurveyID, map[string]any{"title": "Ghost"})

	// then I expect a 404
	resp.RequireStatus(t, 404)
}

func TestUpdateSurvey_ReturnsBadRequest_WhenTitleIsBlank(t *testing.T) {
	// given an open survey and a logged-in officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)
	srv.LoginAs(t, officerDiscordID, "officer1", testutil.AsOfficer())

	// when I blank the title
	resp := srv.Patch(t, "/api/surveys/"+openSurveyID, map[string]any{"title": " "})

	// then I expect a 400
	resp.RequireStatus(t, 400)
	var got map[string]string
	resp.DecodeJSON(t, &got)
	if diff := cmp.Diff(map[string]string{"error": "title: is required"}, got); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
}

func TestUpdateSurvey_ChangesOnlyTheDescription_WhenOfficerSendsADescription(t *testing.T) {
	// given an open survey and a logged-in officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)
	srv.LoginAs(t, officerDiscordID, "officer1", testutil.AsOfficer())

	// when I send only a new description
	resp := srv.Patch(t, "/api/surveys/"+openSurveyID, map[string]any{"description": "  New plan  "})

	// then I expect the trimmed description with the title and deadline untouched
	resp.RequireStatus(t, 200)
	var got struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		ClosesAt    string `json:"closesAt"`
	}
	resp.DecodeJSON(t, &got)
	want := struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		ClosesAt    string `json:"closesAt"`
	}{Title: "Spring roster", Description: "New plan", ClosesAt: "2026-03-21T20:00:00Z"}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected survey (-want +got):\n%s", diff)
	}
}

func addOptionPath(surveyID, questionID string) string {
	return "/api/surveys/" + surveyID + "/questions/" + questionID + "/options"
}

func TestAddOption_ReturnsForbidden_WhenCallerIsOnlyAMember(t *testing.T) {
	// given an open survey and a logged-in member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)
	srv.LoginAs(t, memberDiscordID, "member")

	// when I add an option to the class question
	resp := srv.Post(t, addOptionPath(openSurveyID, openQuestions.class), map[string]any{"option": "Druid"})

	// then I expect a 403 and the options unchanged
	resp.RequireStatus(t, 403)
	var options string
	if err := db.Get(&options, `SELECT options::text FROM survey_questions WHERE id = $1`, openQuestions.class); err != nil {
		t.Fatalf("select options: %v", err)
	}
	if options != `["Mage", "Priest", "Warrior"]` {
		t.Errorf("expected the options to stay unchanged, got %s", options)
	}
}

func TestAddOption_AppendsOptionToTheQuestion_WhenSurveyAlreadyHasResponses(t *testing.T) {
	// given an open survey with a response and a logged-in officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)
	anaID := insertResponse(t, db, openSurveyID, "ana", "Anaria", true, "2026-03-10T12:00:00Z")
	insertAnswer(t, db, anaID, openQuestions.class, `["Mage"]`)
	srv.LoginAs(t, officerDiscordID, "officer1", testutil.AsOfficer())

	// when I add a missing option to the class question
	resp := srv.Post(t, addOptionPath(openSurveyID, openQuestions.class), map[string]any{"option": "  Druid  "})

	// then I expect the survey back with the trimmed option last, other questions untouched
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
			map[string]any{"id": openQuestions.class, "prompt": "Class", "type": "single", "required": true, "options": []any{"Mage", "Priest", "Warrior", "Druid"}},
			map[string]any{"id": openQuestions.secondary, "prompt": "Secondary class", "type": "single", "required": false, "options": []any{"Mage", "Priest", "Warrior"}},
			map[string]any{"id": openQuestions.nights, "prompt": "Raid nights", "type": "multi", "required": true, "options": []any{"Tuesday", "Wednesday", "Thursday"}},
		},
		"myResponse": nil,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected survey (-want +got):\n%s", diff)
	}
}

func TestAddOption_KeepsExistingAnswersAndAcceptsTheNewOption_WhenMemberRespondsAfterwards(t *testing.T) {
	// given an open survey where ana answered Mage and an officer has since added Druid
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)
	anaID := insertResponse(t, db, openSurveyID, "ana", "Anaria", true, "2026-03-10T12:00:00Z")
	insertAnswer(t, db, anaID, openQuestions.class, `["Mage"]`)
	srv.LoginAs(t, officerDiscordID, "officer1", testutil.AsOfficer())
	srv.Post(t, addOptionPath(openSurveyID, openQuestions.class), map[string]any{"option": "Druid"}).RequireStatus(t, 200)
	srv.LoginAs(t, memberDiscordID, "member", testutil.InDiscordServer())

	// when I submit a response choosing the new option
	resp := srv.Put(t, "/api/surveys/"+openSurveyID+"/response", map[string]any{
		"answers": []map[string]any{
			{"questionId": openQuestions.class, "values": []string{"Druid"}},
			{"questionId": openQuestions.nights, "values": []string{"Tuesday"}},
		},
	})

	// then I expect it saved next to ana's untouched answer
	resp.RequireStatus(t, 200)
	var chosen []string
	if err := db.Select(&chosen, `SELECT chosen::text FROM survey_answers WHERE question_id = $1 ORDER BY chosen::text`, openQuestions.class); err != nil {
		t.Fatalf("select answers: %v", err)
	}
	if diff := cmp.Diff([]string{`["Druid"]`, `["Mage"]`}, chosen); diff != "" {
		t.Errorf("unexpected class answers (-want +got):\n%s", diff)
	}
}

func TestAddOption_ReturnsBadRequest_WhenOptionIsRejected(t *testing.T) {
	tests := []struct {
		name      string
		option    string
		wantError string
	}{
		{"the option is blank", "  ", "option: options must not be blank"},
		{"the option is longer than 100 characters", strings.Repeat("a", 101), "option: options must be at most 100 characters each"},
		{"the option is already listed", "Mage", `option: "Mage" is already an option`},
		{"the option only differs by surrounding spaces from a listed one", " Mage ", `option: "Mage" is already an option`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given an open survey and a logged-in officer
			db := testutil.DB(t)
			srv := testutil.NewServer(t, db)
			insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)
			srv.LoginAs(t, officerDiscordID, "officer1", testutil.AsOfficer())

			// when I add that option to the class question
			resp := srv.Post(t, addOptionPath(openSurveyID, openQuestions.class), map[string]any{"option": tt.option})

			// then I expect a 400 naming the problem and the options unchanged
			resp.RequireStatus(t, 400)
			var got map[string]string
			resp.DecodeJSON(t, &got)
			if diff := cmp.Diff(map[string]string{"error": tt.wantError}, got); diff != "" {
				t.Errorf("unexpected error body (-want +got):\n%s", diff)
			}
			var options string
			if err := db.Get(&options, `SELECT options::text FROM survey_questions WHERE id = $1`, openQuestions.class); err != nil {
				t.Fatalf("select options: %v", err)
			}
			if options != `["Mage", "Priest", "Warrior"]` {
				t.Errorf("expected the options to stay unchanged, got %s", options)
			}
		})
	}
}

func TestAddOption_ReturnsBadRequest_WhenQuestionAlreadyHasFiftyOptions(t *testing.T) {
	// given an open survey whose class question is full and a logged-in officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)
	full := make([]string, 50)
	for i := range full {
		full[i] = fmt.Sprintf("Option %d", i+1)
	}
	encoded, err := json.Marshal(full)
	if err != nil {
		t.Fatalf("encode options: %v", err)
	}
	db.MustExec(`UPDATE survey_questions SET options = $2 WHERE id = $1`, openQuestions.class, string(encoded))
	srv.LoginAs(t, officerDiscordID, "officer1", testutil.AsOfficer())

	// when I add one more option
	resp := srv.Post(t, addOptionPath(openSurveyID, openQuestions.class), map[string]any{"option": "One too many"})

	// then I expect a 400 naming the limit
	resp.RequireStatus(t, 400)
	var got map[string]string
	resp.DecodeJSON(t, &got)
	if diff := cmp.Diff(map[string]string{"error": "option: a question can have at most 50 options"}, got); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
}

func TestAddOption_ReturnsConflict_WhenSurveyHasClosed(t *testing.T) {
	// given a closed survey and a logged-in officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, closedSurveyID, "Winter plans", closedInThePast)
	srv.LoginAs(t, officerDiscordID, "officer1", testutil.AsOfficer())

	// when I add an option to its class question
	resp := srv.Post(t, addOptionPath(closedSurveyID, closedQuestions.class), map[string]any{"option": "Druid"})

	// then I expect a 409 and the options unchanged
	resp.RequireStatus(t, 409)
	var got map[string]string
	resp.DecodeJSON(t, &got)
	if diff := cmp.Diff(map[string]string{"error": "survey is closed"}, got); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
	var options string
	if err := db.Get(&options, `SELECT options::text FROM survey_questions WHERE id = $1`, closedQuestions.class); err != nil {
		t.Fatalf("select options: %v", err)
	}
	if options != `["Mage", "Priest", "Warrior"]` {
		t.Errorf("expected the options to stay unchanged, got %s", options)
	}
}

func TestAddOption_ReturnsNotFound_WhenQuestionBelongsToAnotherSurvey(t *testing.T) {
	// given two surveys and a logged-in officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)
	insertSurvey(t, db, closedSurveyID, "Winter plans", closedInThePast)
	srv.LoginAs(t, officerDiscordID, "officer1", testutil.AsOfficer())

	// when I add an option through the open survey to a question of the other one
	resp := srv.Post(t, addOptionPath(openSurveyID, closedQuestions.class), map[string]any{"option": "Druid"})

	// then I expect a 404 and the other survey's options unchanged
	resp.RequireStatus(t, 404)
	var got map[string]string
	resp.DecodeJSON(t, &got)
	if diff := cmp.Diff(map[string]string{"error": "question not found"}, got); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
	var options string
	if err := db.Get(&options, `SELECT options::text FROM survey_questions WHERE id = $1`, closedQuestions.class); err != nil {
		t.Fatalf("select options: %v", err)
	}
	if options != `["Mage", "Priest", "Warrior"]` {
		t.Errorf("expected the options to stay unchanged, got %s", options)
	}
}

func TestAddOption_ReturnsNotFound_WhenSurveyDoesNotExist(t *testing.T) {
	// given a logged-in officer and no surveys
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, officerDiscordID, "officer1", testutil.AsOfficer())

	// when I add an option to a survey nobody created
	resp := srv.Post(t, addOptionPath(openSurveyID, openQuestions.class), map[string]any{"option": "Druid"})

	// then I expect a 404
	resp.RequireStatus(t, 404)
}

// insertResponse stores a response to a survey for a fresh user and returns the response id.
func insertResponse(t *testing.T, db *sqlx.DB, surveyID, discordID, guildName string, inServer bool, submittedAt string) string {
	t.Helper()
	var userID string
	if err := db.Get(&userID, `INSERT INTO users (id, discord_id, username, in_discord_server) VALUES (gen_random_uuid(), $1, $1, $2) RETURNING id`, discordID, inServer); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	var responseID string
	if err := db.Get(&responseID, `
		INSERT INTO survey_responses (survey_id, user_id, guild_name, discord_username, in_discord_server, submitted_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $6) RETURNING id`, surveyID, userID, guildName, discordID, inServer, submittedAt); err != nil {
		t.Fatalf("insert response: %v", err)
	}
	return responseID
}

func insertAnswer(t *testing.T, db *sqlx.DB, responseID, questionID, chosenJSON string) {
	t.Helper()
	db.MustExec(`INSERT INTO survey_answers (response_id, question_id, chosen) VALUES ($1, $2, $3)`, responseID, questionID, chosenJSON)
}

// seedTwoResponses stores ana's response and ben's voided one on the open survey, and returns their ids.
func seedTwoResponses(t *testing.T, db *sqlx.DB, voidedBy string) (anaID, benID string) {
	t.Helper()
	anaID = insertResponse(t, db, openSurveyID, "ana", "Anaria", true, "2026-03-10T12:00:00Z")
	insertAnswer(t, db, anaID, openQuestions.class, `["Mage"]`)
	insertAnswer(t, db, anaID, openQuestions.nights, `["Tuesday","Thursday"]`)
	benID = insertResponse(t, db, openSurveyID, "ben", "=SUM(1)", false, "2026-03-10T13:00:00Z")
	insertAnswer(t, db, benID, openQuestions.class, `["Priest"]`)
	insertAnswer(t, db, benID, openQuestions.nights, `["Wednesday"]`)
	db.MustExec(`UPDATE survey_responses SET voided_at = '2026-03-11T09:00:00Z', voided_by = $1, void_reason = 'joke, answers' WHERE id = $2`, voidedBy, benID)
	return anaID, benID
}

func TestListResponses_ReturnsForbidden_WhenCallerIsOnlyAMember(t *testing.T) {
	// given an open survey and a logged-in member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)
	srv.LoginAs(t, memberDiscordID, "member")

	// when I list the responses
	resp := srv.Get(t, "/api/surveys/"+openSurveyID+"/responses")

	// then I expect a 403
	resp.RequireStatus(t, 403)
}

func TestListResponses_ReturnsEveryResponseWithDiscordDetailsAndVoids_WhenCallerIsAnOfficer(t *testing.T) {
	// given an open survey with an ordinary response and a voided one
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)
	officer := srv.LoginAs(t, officerDiscordID, "officer1", testutil.AsOfficer())
	anaID, benID := seedTwoResponses(t, db, officer.ID.String())

	// when I list the responses
	resp := srv.Get(t, "/api/surveys/"+openSurveyID+"/responses")

	// then I expect both, oldest first, the voided one marked with who voided it and why
	resp.RequireStatus(t, 200)
	var got []map[string]any
	resp.DecodeJSON(t, &got)
	want := []map[string]any{
		{
			"id": anaID, "discordUsername": "ana", "guildName": "Anaria", "inDiscordServer": true,
			"submittedAt": "2026-03-10T12:00:00Z", "updatedAt": "2026-03-10T12:00:00Z",
			"voidedAt": nil, "voidedBy": nil, "voidReason": "",
			"answers": []any{
				map[string]any{"questionId": openQuestions.class, "values": []any{"Mage"}},
				map[string]any{"questionId": openQuestions.nights, "values": []any{"Tuesday", "Thursday"}},
			},
		},
		{
			"id": benID, "discordUsername": "ben", "guildName": "=SUM(1)", "inDiscordServer": false,
			"submittedAt": "2026-03-10T13:00:00Z", "updatedAt": "2026-03-10T13:00:00Z",
			"voidedAt": "2026-03-11T09:00:00Z", "voidedBy": "officer1", "voidReason": "joke, answers",
			"answers": []any{
				map[string]any{"questionId": openQuestions.class, "values": []any{"Priest"}},
				map[string]any{"questionId": openQuestions.nights, "values": []any{"Wednesday"}},
			},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected responses (-want +got):\n%s", diff)
	}
}

func TestListResponses_ReportsTheAccountsCurrentServerFlag_WhenItChangedAfterTheResponseWasSubmitted(t *testing.T) {
	// given a response saved while its author was not known to be in the Discord server
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)
	srv.LoginAs(t, officerDiscordID, "officer1", testutil.AsOfficer())
	insertResponse(t, db, openSurveyID, "ana", "Anaria", false, "2026-03-10T12:00:00Z")
	// and the author has since logged in again and been found in the server
	db.MustExec(`UPDATE users SET in_discord_server = true WHERE discord_id = 'ana'`)

	// when I list the responses
	resp := srv.Get(t, "/api/surveys/"+openSurveyID+"/responses")

	// then I expect the response to show the account's current flag
	resp.RequireStatus(t, 200)
	var got []struct {
		DiscordUsername string `json:"discordUsername"`
		InDiscordServer bool   `json:"inDiscordServer"`
	}
	resp.DecodeJSON(t, &got)
	type row = struct {
		DiscordUsername string `json:"discordUsername"`
		InDiscordServer bool   `json:"inDiscordServer"`
	}
	if diff := cmp.Diff([]row{{DiscordUsername: "ana", InDiscordServer: true}}, got); diff != "" {
		t.Errorf("unexpected responses (-want +got):\n%s", diff)
	}
}

func TestListResponses_ReturnsNotFound_WhenSurveyDoesNotExist(t *testing.T) {
	// given a logged-in officer and no surveys
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, officerDiscordID, "officer1", testutil.AsOfficer())

	// when I list the responses of a survey nobody created
	resp := srv.Get(t, "/api/surveys/"+openSurveyID+"/responses")

	// then I expect a 404
	resp.RequireStatus(t, 404)
}

func TestVoidResponse_ReturnsForbidden_WhenCallerIsOnlyAMember(t *testing.T) {
	// given a response and a logged-in member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)
	responseID := insertResponse(t, db, openSurveyID, "ana", "Anaria", true, "2026-03-10T12:00:00Z")
	srv.LoginAs(t, memberDiscordID, "member")

	// when I try to void it
	resp := srv.Patch(t, "/api/surveys/"+openSurveyID+"/responses/"+responseID, map[string]any{"voided": true, "reason": "nope"})

	// then I expect a 403 and the response left alone
	resp.RequireStatus(t, 403)
	var voided bool
	if err := db.Get(&voided, `SELECT voided_at IS NOT NULL FROM survey_responses`); err != nil {
		t.Fatalf("select response: %v", err)
	}
	if voided {
		t.Error("expected the response not to be voided")
	}
}

func TestVoidResponse_RecordsWhoVoidedItAndWhy_WhenOfficerVoidsAResponse(t *testing.T) {
	// given a response and a logged-in officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)
	responseID := insertResponse(t, db, openSurveyID, "ana", "Anaria", true, "2026-03-10T12:00:00Z")
	officer := srv.LoginAs(t, officerDiscordID, "officer1", testutil.AsOfficer())

	// when I void it with a reason
	resp := srv.Patch(t, "/api/surveys/"+openSurveyID+"/responses/"+responseID, map[string]any{"voided": true, "reason": "  joke answers  "})

	// then I expect the returned response to carry the void, and the row to name me
	resp.RequireStatus(t, 200)
	type voidJSON struct {
		VoidedAt   *string `json:"voidedAt"`
		VoidedBy   *string `json:"voidedBy"`
		VoidReason string  `json:"voidReason"`
	}
	var got voidJSON
	resp.DecodeJSON(t, &got)
	voidedAt, voidedBy := "2026-03-14T20:00:00Z", "officer1"
	if diff := cmp.Diff(voidJSON{VoidedAt: &voidedAt, VoidedBy: &voidedBy, VoidReason: "joke answers"}, got); diff != "" {
		t.Errorf("unexpected void (-want +got):\n%s", diff)
	}
	var storedBy string
	if err := db.Get(&storedBy, `SELECT voided_by FROM survey_responses`); err != nil {
		t.Fatalf("select voided_by: %v", err)
	}
	if storedBy != officer.ID.String() {
		t.Errorf("expected voided_by %s, got %s", officer.ID, storedBy)
	}
}

func TestVoidResponse_ClearsTheVoid_WhenOfficerUnvoidsAResponse(t *testing.T) {
	// given a voided response and a logged-in officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)
	officer := srv.LoginAs(t, officerDiscordID, "officer1", testutil.AsOfficer())
	_, benID := seedTwoResponses(t, db, officer.ID.String())

	// when I unvoid it
	resp := srv.Patch(t, "/api/surveys/"+openSurveyID+"/responses/"+benID, map[string]any{"voided": false})

	// then I expect the void, its author and its reason all cleared
	resp.RequireStatus(t, 200)
	type voidRow struct {
		Voided     bool    `db:"voided"`
		VoidedBy   *string `db:"voided_by"`
		VoidReason string  `db:"void_reason"`
	}
	var row voidRow
	if err := db.Get(&row, `SELECT voided_at IS NOT NULL AS voided, voided_by, void_reason FROM survey_responses WHERE id = $1`, benID); err != nil {
		t.Fatalf("select response: %v", err)
	}
	if diff := cmp.Diff(voidRow{}, row); diff != "" {
		t.Errorf("unexpected response row (-want +got):\n%s", diff)
	}
}

func TestVoidResponse_ReturnsNotFound_WhenResponseBelongsToAnotherSurvey(t *testing.T) {
	// given a response on the open survey and a second, closed survey
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)
	insertSurvey(t, db, closedSurveyID, "Winter plans", closedInThePast)
	responseID := insertResponse(t, db, openSurveyID, "ana", "Anaria", true, "2026-03-10T12:00:00Z")
	srv.LoginAs(t, officerDiscordID, "officer1", testutil.AsOfficer())

	// when I void it through the other survey's URL
	resp := srv.Patch(t, "/api/surveys/"+closedSurveyID+"/responses/"+responseID, map[string]any{"voided": true})

	// then I expect a 404 and the response left alone
	resp.RequireStatus(t, 404)
	var got map[string]string
	resp.DecodeJSON(t, &got)
	if diff := cmp.Diff(map[string]string{"error": "response not found"}, got); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
	var voided bool
	if err := db.Get(&voided, `SELECT voided_at IS NOT NULL FROM survey_responses`); err != nil {
		t.Fatalf("select response: %v", err)
	}
	if voided {
		t.Error("expected the response not to be voided")
	}
}

func TestVoidResponse_ReturnsBadRequest_WhenReasonIsLongerThan500Characters(t *testing.T) {
	// given a response and a logged-in officer
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)
	responseID := insertResponse(t, db, openSurveyID, "ana", "Anaria", true, "2026-03-10T12:00:00Z")
	srv.LoginAs(t, officerDiscordID, "officer1", testutil.AsOfficer())

	// when I void it with a 501 character reason
	resp := srv.Patch(t, "/api/surveys/"+openSurveyID+"/responses/"+responseID, map[string]any{"voided": true, "reason": strings.Repeat("a", 501)})

	// then I expect a 400
	resp.RequireStatus(t, 400)
	var got map[string]string
	resp.DecodeJSON(t, &got)
	if diff := cmp.Diff(map[string]string{"error": "reason: must be at most 500 characters"}, got); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
}

func TestExportResponses_ReturnsForbidden_WhenCallerIsOnlyAMember(t *testing.T) {
	// given an open survey and a logged-in member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)
	srv.LoginAs(t, memberDiscordID, "member")

	// when I download the CSV
	resp := srv.Get(t, "/api/surveys/"+openSurveyID+"/responses.csv")

	// then I expect a 403
	resp.RequireStatus(t, 403)
}

func TestExportResponses_ReturnsOneRowPerResponseWithFormulaCellsDefused_WhenCallerIsAnOfficer(t *testing.T) {
	// given an open survey with an ordinary response and a voided one whose guild name looks like a formula
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)
	officer := srv.LoginAs(t, officerDiscordID, "officer1", testutil.AsOfficer())
	seedTwoResponses(t, db, officer.ID.String())

	// when I download the CSV
	resp := srv.Get(t, "/api/surveys/"+openSurveyID+"/responses.csv")

	// then I expect a CSV attachment with a column per question, and the formula cell prefixed with a quote
	resp.RequireStatus(t, 200)
	if got := resp.Header.Get("Content-Type"); got != "text/csv; charset=utf-8" {
		t.Errorf("unexpected content type %q", got)
	}
	if got := resp.Header.Get("Content-Disposition"); got != `attachment; filename="survey-`+openSurveyID+`-responses.csv"` {
		t.Errorf("unexpected content disposition %q", got)
	}
	want := "Discord name,Guild name,In Discord server,Submitted,Voided,Void reason,Class,Secondary class,Raid nights\n" +
		"ana,Anaria,true,2026-03-10T12:00:00Z,false,,Mage,,Tuesday; Thursday\n" +
		"ben,'=SUM(1),false,2026-03-10T13:00:00Z,true,\"joke, answers\",Priest,,Wednesday\n"
	if diff := cmp.Diff(want, string(resp.Body)); diff != "" {
		t.Errorf("unexpected csv (-want +got):\n%s", diff)
	}
}

func TestExportResponses_ReturnsNotFound_WhenSurveyDoesNotExist(t *testing.T) {
	// given a logged-in officer and no surveys
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, officerDiscordID, "officer1", testutil.AsOfficer())

	// when I download the CSV of a survey nobody created
	resp := srv.Get(t, "/api/surveys/"+openSurveyID+"/responses.csv")

	// then I expect a 404
	resp.RequireStatus(t, 404)
}

func TestGetLiveResults_ReturnsForbidden_WhenCallerIsOnlyAMember(t *testing.T) {
	// given an open survey and a logged-in member
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)
	srv.LoginAs(t, memberDiscordID, "member")

	// when I ask for the live results
	resp := srv.Get(t, "/api/surveys/"+openSurveyID+"/live-results")

	// then I expect a 403
	resp.RequireStatus(t, 403)
}

func TestGetLiveResults_ReturnsNotFound_WhenSurveyDoesNotExist(t *testing.T) {
	// given a logged-in officer and no surveys
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	srv.LoginAs(t, officerDiscordID, "officer1", testutil.AsOfficer())

	// when I ask for the live results of a survey nobody created
	resp := srv.Get(t, "/api/surveys/"+openSurveyID+"/live-results")

	// then I expect a 404 saying the survey was not found
	resp.RequireStatus(t, 404)
	var got map[string]string
	resp.DecodeJSON(t, &got)
	if diff := cmp.Diff(map[string]string{"error": "survey not found"}, got); diff != "" {
		t.Errorf("unexpected error body (-want +got):\n%s", diff)
	}
}

func TestGetLiveResults_ReturnsCountsWithoutVoidedResponses_WhenSurveyIsStillOpen(t *testing.T) {
	// given an open survey with one ordinary response and one voided response
	db := testutil.DB(t)
	srv := testutil.NewServer(t, db)
	insertSurvey(t, db, openSurveyID, "Spring roster", closesInTheFuture)
	officer := srv.LoginAs(t, officerDiscordID, "officer1", testutil.AsOfficer())
	seedTwoResponses(t, db, officer.ID.String())

	// when I ask for the live results
	resp := srv.Get(t, "/api/surveys/"+openSurveyID+"/live-results")

	// then I expect the turnout and counts of the ordinary response only
	resp.RequireStatus(t, 200)
	var got map[string]any
	resp.DecodeJSON(t, &got)
	want := map[string]any{
		"surveyId": openSurveyID,
		"title":    "Spring roster",
		"turnout":  float64(1),
		"questions": []any{
			map[string]any{"id": openQuestions.class, "prompt": "Class", "type": "single", "options": []any{
				map[string]any{"label": "Mage", "count": float64(1)},
				map[string]any{"label": "Priest", "count": float64(0)},
				map[string]any{"label": "Warrior", "count": float64(0)},
			}},
			map[string]any{"id": openQuestions.secondary, "prompt": "Secondary class", "type": "single", "options": []any{
				map[string]any{"label": "Mage", "count": float64(0)},
				map[string]any{"label": "Priest", "count": float64(0)},
				map[string]any{"label": "Warrior", "count": float64(0)},
			}},
			map[string]any{"id": openQuestions.nights, "prompt": "Raid nights", "type": "multi", "options": []any{
				map[string]any{"label": "Tuesday", "count": float64(1)},
				map[string]any{"label": "Wednesday", "count": float64(0)},
				map[string]any{"label": "Thursday", "count": float64(1)},
			}},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected live results (-want +got):\n%s", diff)
	}
}
