import { ApiError, baseUrl } from './client'

export interface UploadedImage {
  id: string
  url: string
}

export async function uploadImage(file: File): Promise<UploadedImage> {
  const form = new FormData()
  form.append('file', file)
  const response = await fetch(`${baseUrl}/images`, {
    method: 'POST',
    credentials: 'include',
    body: form,
  })
  if (!response.ok) {
    const problem = (await response.json().catch(() => null)) as { error?: string } | null
    throw new ApiError(problem?.error ?? `Upload failed with ${response.status}`, response.status)
  }
  return (await response.json()) as UploadedImage
}
