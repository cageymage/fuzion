import type { RaidProgress, RaidTier } from '../types/raidProgress'
import { apiGet, baseUrl } from './client'

// A guild between tiers, or one where nothing is current yet, is a normal
// state, not an error, so a missing current tier (404) resolves to an empty
// list rather than throwing.
export async function fetchRaidProgress(): Promise<RaidProgress[]> {
  const response = await fetch(`${baseUrl}/raid-progress`, { credentials: 'include' })
  if (response.status === 404) {
    return []
  }
  if (!response.ok) {
    throw new Error(`GET /raid-progress failed with ${response.status}`)
  }
  return (await response.json()) as RaidProgress[]
}

export function fetchRaidTiers(): Promise<RaidTier[]> {
  return apiGet<RaidTier[]>('/raid-tiers')
}
