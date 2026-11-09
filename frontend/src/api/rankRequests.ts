import type { RankRequest, RankRequestDecision } from '../types/rankRequests'
import { apiDelete, apiGet, apiSend } from './client'

export function fetchMyRankRequests(): Promise<RankRequest[]> {
  return apiGet<RankRequest[]>('/me/rank-requests')
}

export function createRankRequest(characterId: string, note: string): Promise<RankRequest> {
  return apiSend<RankRequest>('POST', `/me/characters/${characterId}/rank-requests`, { note })
}

export function cancelRankRequest(id: string): Promise<void> {
  return apiDelete(`/me/rank-requests/${id}`)
}

export function fetchRankRequests(): Promise<RankRequest[]> {
  return apiGet<RankRequest[]>('/rank-requests')
}

export function fetchOpenRankRequestCount(): Promise<{ open: number }> {
  return apiGet<{ open: number }>('/rank-requests/count')
}

export function resolveRankRequest(id: string, status: RankRequestDecision, note: string): Promise<RankRequest> {
  return apiSend<RankRequest>('PATCH', `/rank-requests/${id}`, { status, note })
}
