import type { Character } from '../types/roster'
import { apiGet } from './client'

export function fetchRoster(): Promise<Character[]> {
  return apiGet<Character[]>('/roster')
}
