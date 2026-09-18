import type { NextRaid } from '../types/raids'
import { apiGet } from './client'

export function fetchNextRaid(): Promise<NextRaid | null> {
  return apiGet<NextRaid | null>('/raids/next')
}
