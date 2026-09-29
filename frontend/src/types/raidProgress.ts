export interface RaidProgressBoss {
  id: string
  name: string
  killedAt: string | null
}

export interface RaidProgress {
  tier: { name: string }
  bosses: RaidProgressBoss[]
  killed: number
  total: number
}

export interface RaidTier {
  id: string
  name: string
  isCurrent: boolean
  bosses: RaidProgressBoss[]
}
