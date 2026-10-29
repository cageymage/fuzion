import type { ProfessionEntry } from '../types/professions'
import { apiDelete, apiGet, apiSend } from './client'

export function fetchProfessions(): Promise<ProfessionEntry[]> {
  return apiGet<ProfessionEntry[]>('/professions')
}

export interface MyProfessionRequest {
  profession: string
  skillLevel: number
}

export function addMyProfession(characterId: string, request: MyProfessionRequest): Promise<ProfessionEntry> {
  return apiSend<ProfessionEntry>('POST', `/me/characters/${characterId}/professions`, request)
}

export function updateMyProfession(id: string, request: Partial<MyProfessionRequest>): Promise<ProfessionEntry> {
  return apiSend<ProfessionEntry>('PATCH', `/me/professions/${id}`, request)
}

export function deleteMyProfession(id: string): Promise<void> {
  return apiDelete(`/me/professions/${id}`)
}
