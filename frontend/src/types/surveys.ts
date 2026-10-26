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

export type SurveyOptionPreset = 'wow-classes' | 'wow-roles' | 'weekdays'

export interface SurveyQuestionInput {
  prompt: string
  type: SurveyQuestionType
  required: boolean
  options?: string[]
  preset?: SurveyOptionPreset
}

export interface CreateSurveyRequest {
  title: string
  description: string
  closesAt: string
  questions: SurveyQuestionInput[]
}

export interface UpdateSurveyRequest {
  title?: string
  description?: string
  closesAt?: string
  questions?: SurveyQuestionInput[]
}

export interface OfficerSurveyResponse {
  id: string
  discordUsername: string
  guildName: string
  inDiscordServer: boolean
  submittedAt: string
  updatedAt: string
  voidedAt: string | null
  voidedBy: string | null
  voidReason: string
  answers: SurveyAnswer[]
}

export interface VoidSurveyResponseRequest {
  voided: boolean
  reason?: string
}
