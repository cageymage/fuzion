import type {
  CreateRaidTierRequest,
  RaidProgress,
  RaidProgressBoss,
  RaidTier,
} from '../types/raidProgress'
import { apiDelete, apiGet, apiSend, baseUrl } from './client'

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

export function createRaidTier(request: CreateRaidTierRequest): Promise<RaidTier> {
  return apiSend<RaidTier>('POST', '/raid-tiers', request)
}

export function setRaidTierCurrent(id: string, isCurrent: boolean): Promise<RaidTier> {
  return apiSend<RaidTier>('PATCH', `/raid-tiers/${id}`, { isCurrent })
}

export function setRaidBossKilled(id: string, killed: boolean): Promise<RaidProgressBoss> {
  return apiSend<RaidProgressBoss>('PATCH', `/raid-bosses/${id}`, { killed })
}

export function deleteRaidTier(id: string): Promise<void> {
  return apiDelete(`/raid-tiers/${id}`)
}

export function deleteRaidBoss(id: string): Promise<void> {
  return apiDelete(`/raid-bosses/${id}`)
}
