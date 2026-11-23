import type { ProfessionEntry, Recipe } from '../types/professions'
import { apiGet, apiSend } from './client'

export function fetchProfessions(): Promise<ProfessionEntry[]> {
  return apiGet<ProfessionEntry[]>('/professions')
}

export function setSpecialization(professionId: string, specialization: string | null): Promise<ProfessionEntry> {
  return apiSend<ProfessionEntry>('PATCH', `/professions/${professionId}/specialization`, { specialization })
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
