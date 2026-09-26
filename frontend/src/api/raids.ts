import type { Raid } from '../types/raids'
import { apiGet } from './client'

export function fetchNextRaid(): Promise<Raid | null> {
  return apiGet<Raid | null>('/raids/next')
}

export function fetchUpcomingRaids(): Promise<Raid[]> {
  return apiGet<Raid[]>('/raids')
}
