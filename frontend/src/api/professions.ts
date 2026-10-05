import type { ProfessionEntry } from '../types/professions'
import { apiGet } from './client'

export function fetchProfessions(): Promise<ProfessionEntry[]> {
  return apiGet<ProfessionEntry[]>('/professions')
}
