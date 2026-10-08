import type { Character } from '../../types/roster'

export interface RosterSummary {
  total: number
  tanks: number
  healers: number
  dps: number
  mains: number
  alts: number
  averageLevel: number | null
}

// Roles are counted by each character's primary role, so a tank/healer is one tank.
export function summarizeRoster(characters: Character[]): RosterSummary {
  const levels = characters.flatMap((character) => (character.level === null ? [] : [character.level]))
  const mains = characters.filter((character) => character.isMain).length
  return {
    total: characters.length,
    tanks: characters.filter((character) => character.role === 'tank').length,
    healers: characters.filter((character) => character.role === 'healer').length,
    dps: characters.filter((character) => character.role === 'dps').length,
    mains,
    alts: characters.length - mains,
    averageLevel:
      levels.length === 0 ? null : Math.round(levels.reduce((sum, level) => sum + level, 0) / levels.length),
  }
}
