export const baseUrl = import.meta.env.VITE_API_BASE_URL ?? '/api'

export async function apiGet<T>(path: string): Promise<T> {
  const response = await fetch(`${baseUrl}${path}`, { credentials: 'include' })
  if (!response.ok) {
    throw new Error(`GET ${path} failed with ${response.status}`)
  }
  return (await response.json()) as T
}

export async function apiPost(path: string): Promise<void> {
  const response = await fetch(`${baseUrl}${path}`, { method: 'POST', credentials: 'include' })
  if (!response.ok) {
    throw new Error(`POST ${path} failed with ${response.status}`)
  }
}
