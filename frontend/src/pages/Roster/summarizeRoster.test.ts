import { describe, expect, it } from 'vitest'
import type { Character } from '../../types/roster'
import { summarizeRoster } from './summarizeRoster'

const base: Character = {
  id: 'char-1',
  name: 'Ragnok',
  secondaryName: '',
  realm: 'Emberreach',
  class: 'Warrior',
  spec: 'Protection',
  role: 'tank',
  spec2: null,
  role2: null,
  isMain: true,
  raidTeam: null,
  race: null,
  level: null,
  faction: null,
  createdAt: '2026-01-01T00:00:00.000Z',
  professions: [],
}

describe('summarizeRoster', () => {
  it('should count characters by primary role', () => {
    const summary = summarizeRoster([
      base,
      { ...base, id: 'b', role: 'healer', spec2: 'Holy', role2: 'tank' },
      { ...base, id: 'c', role: 'dps' },
      { ...base, id: 'd', role: 'dps' },
    ])

    expect(summary).toMatchObject({ total: 4, tanks: 1, healers: 1, dps: 2 })
  })

  it('should count mains and alts separately', () => {
    const summary = summarizeRoster([
      base,
      { ...base, id: 'b', isMain: false },
      { ...base, id: 'c', isMain: false },
    ])

    expect(summary).toMatchObject({ mains: 1, alts: 2 })
  })

  it('should average only the characters that have a level, rounded to a whole level', () => {
    const summary = summarizeRoster([
      { ...base, level: 60 },
      { ...base, id: 'b', level: 57 },
      { ...base, id: 'c', level: null },
    ])

    expect(summary.averageLevel).toBe(59)
  })

  it('should have no average level when no character has a level', () => {
    expect(summarizeRoster([base]).averageLevel).toBeNull()
  })

  it('should be all zeros when the roster is empty', () => {
    expect(summarizeRoster([])).toEqual({
      total: 0,
      tanks: 0,
      healers: 0,
      dps: 0,
      mains: 0,
      alts: 0,
      averageLevel: null,
    })
  })
})
