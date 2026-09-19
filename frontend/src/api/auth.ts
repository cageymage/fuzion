import type { AuthUser } from '../types/auth'
import { apiPost, baseUrl } from './client'

export const loginUrl = `${baseUrl}/auth/login`

// A 401 here just means "nobody is logged in", which is a normal state, not an error.
export async function fetchCurrentUser(): Promise<AuthUser | null> {
  const response = await fetch(`${baseUrl}/auth/me`, { credentials: 'include' })
  if (response.status === 401) {
    return null
  }
  if (!response.ok) {
    throw new Error(`GET /auth/me failed with ${response.status}`)
  }
  return (await response.json()) as AuthUser
}

export function logout(): Promise<void> {
  return apiPost('/auth/logout')
}
