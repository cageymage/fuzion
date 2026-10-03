import type { Character, CreateCharacterRequest, UpdateCharacterRequest } from '../types/roster'
import { apiDelete, apiGet, apiSend } from './client'

export function fetchRoster(): Promise<Character[]> {
  return apiGet<Character[]>('/roster')
}

export function createCharacter(request: CreateCharacterRequest): Promise<Character> {
  return apiSend<Character>('POST', '/roster', request)
}

export function updateCharacter(id: string, request: UpdateCharacterRequest): Promise<Character> {
  return apiSend<Character>('PATCH', `/roster/${id}`, request)
}

export function deleteCharacter(id: string): Promise<void> {
  return apiDelete(`/roster/${id}`)
}
