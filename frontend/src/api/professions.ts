import type { ProfessionEntry, Recipe } from '../types/professions'
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

export interface RecipeFilter {
  query: string
  characterId: string
}

export function fetchRecipes({ query, characterId }: RecipeFilter): Promise<Recipe[]> {
  const params = new URLSearchParams()
  if (query.trim()) {
    params.set('q', query.trim())
  }
  if (characterId) {
    params.set('characterId', characterId)
  }
  const suffix = params.size > 0 ? `?${params}` : ''
  return apiGet<Recipe[]>(`/recipes${suffix}`)
}
