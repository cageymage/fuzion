export const baseUrl = import.meta.env.VITE_API_BASE_URL ?? '/api'

export async function apiGet<T>(path: string): Promise<T> {
  const response = await fetch(`${baseUrl}${path}`, { credentials: 'include' })
  if (!response.ok) {
    throw new Error(`GET ${path} failed with ${response.status}`)
  }
  return (await response.json()) as T
}

export async function apiGetOptional<T>(path: string): Promise<T | null> {
  const response = await fetch(`${baseUrl}${path}`, { credentials: 'include' })
  if (response.status === 204) {
    return null
  }
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

export async function apiDelete(path: string): Promise<void> {
  const response = await fetch(`${baseUrl}${path}`, { method: 'DELETE', credentials: 'include' })
  if (!response.ok) {
    const problem = (await response.json().catch(() => null)) as { error?: string } | null
    throw new Error(problem?.error ?? `DELETE ${path} failed with ${response.status}`)
  }
}

export async function apiPut(path: string, body: unknown): Promise<void> {
  const response = await fetch(`${baseUrl}${path}`, {
    method: 'PUT',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!response.ok) {
    const problem = (await response.json().catch(() => null)) as { error?: string } | null
    throw new Error(problem?.error ?? `PUT ${path} failed with ${response.status}`)
  }
}

export async function apiSend<T>(method: 'POST' | 'PATCH', path: string, body: unknown): Promise<T> {
  const response = await fetch(`${baseUrl}${path}`, {
    method,
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!response.ok) {
    const problem = (await response.json().catch(() => null)) as { error?: string } | null
    throw new Error(problem?.error ?? `${method} ${path} failed with ${response.status}`)
  }
  return (await response.json()) as T
}
