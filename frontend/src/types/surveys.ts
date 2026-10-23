export type SurveyStatus = 'open' | 'closed'

export type SurveyQuestionType = 'single' | 'multi'

export interface SurveySummary {
  id: string
  title: string
  closesAt: string
  status: SurveyStatus
}

export interface SurveyQuestion {
  id: string
  prompt: string
  type: SurveyQuestionType
  required: boolean
  options: string[]
}

export interface SurveyAnswer {
  questionId: string
  values: string[]
}

export interface SurveyResponse {
  guildName: string
  answers: SurveyAnswer[]
}

export interface Survey extends SurveySummary {
  description: string
  questions: SurveyQuestion[]
  myResponse: SurveyResponse | null
}

export type SubmitSurveyRequest = SurveyResponse

export interface OptionResult {
  label: string
  count: number
}

export interface QuestionResult {
  id: string
  prompt: string
  type: SurveyQuestionType
  options: OptionResult[]
}

export interface SurveyResults {
  surveyId: string
  title: string
  turnout: number
  questions: QuestionResult[]
}
