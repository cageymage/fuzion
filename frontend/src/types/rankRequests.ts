export type RankRequestDirection = 'to_main' | 'to_alt'

// Cancelled requests are never returned, so they have no status here.
export type RankRequestStatus = 'open' | 'done' | 'declined'

export type RankRequestDecision = 'done' | 'declined'

export interface RankRequest {
  id: string
  characterId: string
  characterName: string
  characterSecondaryName: string
  characterClass: string
  direction: RankRequestDirection
  status: RankRequestStatus
  note: string
  resolutionNote: string
  requestedBy: string
  createdAt: string
  resolvedAt: string | null
  // null on a resolved request means the roster sync saw the new rank and closed it.
  resolvedBy: string | null
}
