import type {
  Application,
  ApplicationStatusFilter,
  ReviewApplicationRequest,
  SubmitApplicationRequest,
} from '../types/applications'
import { apiGet, apiSend } from './client'

export function submitApplication(request: SubmitApplicationRequest): Promise<unknown> {
  return apiSend<unknown>('POST', '/applications', request)
}

export function fetchApplications(status: ApplicationStatusFilter): Promise<Application[]> {
  const query = status === 'all' ? '' : `?status=${status}`
  return apiGet<Application[]>(`/applications${query}`)
}

export function reviewApplication(id: string, request: ReviewApplicationRequest): Promise<Application> {
  return apiSend<Application>('PATCH', `/applications/${id}`, request)
}
