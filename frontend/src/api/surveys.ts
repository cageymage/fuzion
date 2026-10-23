import type {
  CreateSurveyRequest,
  OfficerSurveyResponse,
  SubmitSurveyRequest,
  Survey,
  SurveyResponse,
  SurveyResults,
  SurveySummary,
  UpdateSurveyRequest,
  VoidSurveyResponseRequest,
} from '../types/surveys'
import { apiGet, apiSend, baseUrl } from './client'

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

export function createSurvey(request: CreateSurveyRequest): Promise<Survey> {
  return apiSend<Survey>('POST', '/surveys', request)
}

export function updateSurvey(id: string, request: UpdateSurveyRequest): Promise<Survey> {
  return apiSend<Survey>('PATCH', `/surveys/${id}`, request)
}

export function fetchSurveyResponses(id: string): Promise<OfficerSurveyResponse[]> {
  return apiGet<OfficerSurveyResponse[]>(`/surveys/${id}/responses`)
}

export function voidSurveyResponse(
  surveyId: string,
  responseId: string,
  request: VoidSurveyResponseRequest,
): Promise<OfficerSurveyResponse> {
  return apiSend<OfficerSurveyResponse>('PATCH', `/surveys/${surveyId}/responses/${responseId}`, request)
}

// A plain anchor target: the browser sends the session cookie and downloads the attachment.
export function surveyExportUrl(id: string): string {
  return `${baseUrl}/surveys/${id}/responses.csv`
}
