import type {
  SubmitSurveyRequest,
  Survey,
  SurveyResponse,
  SurveyResults,
  SurveySummary,
} from '../types/surveys'
import { apiGet, apiSend } from './client'

export function fetchSurveys(): Promise<SurveySummary[]> {
  return apiGet<SurveySummary[]>('/surveys')
}

export function fetchSurvey(id: string): Promise<Survey> {
  return apiGet<Survey>(`/surveys/${id}`)
}

export function fetchSurveyResults(id: string): Promise<SurveyResults> {
  return apiGet<SurveyResults>(`/surveys/${id}/results`)
}

export function submitSurveyResponse(id: string, request: SubmitSurveyRequest): Promise<SurveyResponse> {
  return apiSend<SurveyResponse>('PUT', `/surveys/${id}/response`, request)
}
