import type { SubmitApplicationRequest } from '../types/applications'
import { apiSend } from './client'

export function submitApplication(request: SubmitApplicationRequest): Promise<unknown> {
  return apiSend<unknown>('POST', '/applications', request)
}
