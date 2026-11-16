import type { SubmissionFileResult, SubmissionPage, SubmissionStatus } from '../types/submissions'
import { ApiError, apiDelete, apiGet, apiSend, baseUrl } from './client'

// Must match AttestationVersion in internal/submissions; the server refuses any other.
// Change both whenever the license or attestation wording on the form changes.
export const ATTESTATION_VERSION = '2026-10-26'

// Mirrors the server's limits in internal/submissions, so the page can say no before uploading.
export const MAX_FILE_BYTES = 20 * 1024 * 1024
export const MAX_BATCH_FILES = 10
export const MAX_BATCH_BYTES = 50 * 1024 * 1024
export const MAX_DESCRIPTION = 500
export const ALLOWED_TYPES = ['image/png', 'image/jpeg', 'image/webp', 'image/gif']

export function fetchSubmissionStatus(): Promise<SubmissionStatus> {
  return apiGet<SubmissionStatus>('/submissions/status')
}

export interface SubmissionUpload {
  file: File
  description: string
}

// A whole-request failure (quota, closed, attestation) throws; per-file outcomes come back as results.
export async function submitImages(uploads: SubmissionUpload[]): Promise<SubmissionFileResult[]> {
  const form = new FormData()
  form.append(
    'metadata',
    new Blob(
      [
        JSON.stringify({
          attestationAccepted: true,
          attestationVersion: ATTESTATION_VERSION,
          descriptions: uploads.map(({ description }) => description),
        }),
      ],
      { type: 'application/json' },
    ),
  )
  for (const { file } of uploads) {
    form.append('files', file, file.name)
  }
  const response = await fetch(`${baseUrl}/submissions`, { method: 'POST', credentials: 'include', body: form })
  const body = (await response.json().catch(() => null)) as { results?: SubmissionFileResult[]; error?: string } | null
  if (body?.results) {
    return body.results
  }
  throw new ApiError(body?.error ?? `Submission failed with ${response.status}`, response.status)
}

export function fetchSubmissions(limit: number, offset: number): Promise<SubmissionPage> {
  return apiGet<SubmissionPage>(`/submissions?limit=${limit}&offset=${offset}`)
}

export function deleteSubmission(id: string): Promise<void> {
  return apiDelete(`/submissions/${id}`)
}

export function deleteSubmissions(ids: string[]): Promise<{ deleted: string[] }> {
  return apiSend<{ deleted: string[] }>('POST', '/submissions/bulk-delete', { ids })
}
