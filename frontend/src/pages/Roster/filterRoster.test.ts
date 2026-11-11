import { describe, expect, it } from 'vitest'
import type { Character } from '../../types/roster'
import { filterRoster, sortRoster } from './filterRoster'

const main: Character = {
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
  rank: null,
  rankOrder: null,
  raidTeam: 'Team Alpha',
  race: null,
  level: null,
  faction: null,
  gender: null,
  createdAt: '2026-01-01T00:00:00.000Z',
  professions: [],
}

const healer: Character = {
  ...main,
  id: 'char-2',
  name: 'Selene',
  class: 'Priest',
  spec: 'Holy',
  role: 'healer',
}

const alt: Character = {
  ...main,
  id: 'char-3',
  name: 'Zaldrix',
  class: 'Warlock',
  spec: 'Affliction',
  role: 'dps',
  isMain: false,
  rank: null,
  rankOrder: null,
}

const roster = [main, healer, alt]

describe('filterRoster', () => {
  it('should exclude alts when showAlts is false', () => {
    const result = filterRoster(roster, { role: 'all', className: 'all', showAlts: false, search: '' })

    expect(result).toEqual([main, healer])
  })

  it('should include alts when showAlts is true', () => {
    const result = filterRoster(roster, { role: 'all', className: 'all', showAlts: true, search: '' })

    expect(result).toEqual([main, healer, alt])
  })

  it('should only keep characters matching the selected role', () => {
    const result = filterRoster(roster, { role: 'healer', className: 'all', showAlts: false, search: '' })

    expect(result).toEqual([healer])
  })

  it('should keep a character whose second role matches the selected role', () => {
    const tankHealer: Character = { ...main, id: 'char-4', name: 'Aeliana', spec2: 'Holy', role2: 'healer' }

    const result = filterRoster([main, tankHealer], { role: 'healer', className: 'all', showAlts: false, search: '' })

    expect(result).toEqual([tankHealer])
  })

  it('should keep a character whose first role matches the selected role when it also has a second role', () => {
    const tankHealer: Character = { ...main, id: 'char-4', name: 'Aeliana', spec2: 'Holy', role2: 'healer' }

    const result = filterRoster([main, tankHealer], { role: 'tank', className: 'all', showAlts: false, search: '' })

    expect(result).toEqual([main, tankHealer])
  })

  it('should drop a character whose neither role matches the selected role', () => {
    const tankHealer: Character = { ...main, id: 'char-4', name: 'Aeliana', spec2: 'Holy', role2: 'healer' }

    const result = filterRoster([tankHealer], { role: 'dps', className: 'all', showAlts: false, search: '' })

    expect(result).toEqual([])
  })

  it('should only keep characters matching the selected class', () => {
    const result = filterRoster(roster, { role: 'all', className: 'Priest', showAlts: false, search: '' })

    expect(result).toEqual([healer])
  })

  it('should combine the role filter and the class filter', () => {
    const result = filterRoster(roster, { role: 'dps', className: 'Warlock', showAlts: true, search: '' })

    expect(result).toEqual([alt])
  })
})

const noFilters = { role: 'all', className: 'all', showAlts: true, search: '' } as const

describe('filterRoster name search', () => {
  it('should match secondary names when searching by name', () => {
    const withSecondary: Character = { ...main, id: 'char-4', name: 'Aeliana', secondaryName: 'Dawnsong' }

    const result = filterRoster([main, withSecondary], { ...noFilters, search: 'dawn' })

    expect(result).toEqual([withSecondary])
  })

  it('should ignore case and surrounding spaces when searching by name', () => {
    const result = filterRoster(roster, { ...noFilters, search: '  RAGN ' })

    expect(result).toEqual([main])
  })

  it('should combine the name search with the role filter', () => {
    const result = filterRoster(roster, { ...noFilters, role: 'healer', search: 'l' })

    expect(result).toEqual([healer])
  })

  it('should return nobody when no name matches', () => {
    expect(filterRoster(roster, { ...noFilters, search: 'zzz' })).toEqual([])
  })
})

describe('sortRoster', () => {
  const orc: Character = { ...main, id: 'a', name: 'Bravo', race: 'Orc', level: 60, raidTeam: 'Team 2' }
  const dwarf: Character = { ...main, id: 'b', name: 'Alpha', race: 'Dwarf', level: 9, raidTeam: 'Team 1' }
  const unset: Character = { ...main, id: 'c', name: 'Charlie', race: null, level: null, raidTeam: null }

  it('should keep the given order when no sort is chosen', () => {
    expect(sortRoster([orc, dwarf, unset], null)).toEqual([orc, dwarf, unset])
  })

  it('should sort by level numerically when ascending', () => {
    const result = sortRoster([orc, dwarf, unset], { key: 'level', direction: 'asc' })

    expect(result.map((c) => c.id)).toEqual(['b', 'a', 'c'])
  })

  it('should keep characters without a level last when sorting descending', () => {
    const result = sortRoster([unset, dwarf, orc], { key: 'level', direction: 'desc' })

    expect(result.map((c) => c.id)).toEqual(['a', 'b', 'c'])
  })

  it('should sort by race name when sorting by race', () => {
    const result = sortRoster([orc, unset, dwarf], { key: 'race', direction: 'asc' })

    expect(result.map((c) => c.id)).toEqual(['b', 'a', 'c'])
  })

  it('should sort by raid team with unassigned characters last', () => {
    const result = sortRoster([unset, orc, dwarf], { key: 'raidTeam', direction: 'asc' })

    expect(result.map((c) => c.id)).toEqual(['b', 'a', 'c'])
  })

  it('should sort by name then secondary name when sorting by name', () => {
    const second: Character = { ...dwarf, id: 'd', secondaryName: 'Zed' }

    const result = sortRoster([orc, second, dwarf], { key: 'name', direction: 'asc' })

    expect(result.map((c) => c.id)).toEqual(['b', 'd', 'a'])
  })

  describe('by rank', () => {
    const guildMaster: Character = { ...main, id: 'gm', rank: 'Guild Master', rankOrder: 0 }
    const veteran: Character = { ...main, id: 'vet', rank: 'Veteran', rankOrder: 4 }
    const member: Character = { ...main, id: 'mem', rank: 'Member', rankOrder: 6 }
    const unranked: Character = { ...main, id: 'none', rank: null, rankOrder: null }

    it('should order by rank number rather than rank name when ascending', () => {
      const result = sortRoster([member, veteran, guildMaster], { key: 'rank', direction: 'asc' })

      expect(result.map((c) => c.id)).toEqual(['gm', 'vet', 'mem'])
    })

    it('should put the highest rank number first when descending', () => {
      const result = sortRoster([guildMaster, member, veteran], { key: 'rank', direction: 'desc' })

      expect(result.map((c) => c.id)).toEqual(['mem', 'vet', 'gm'])
    })

    it('should keep characters without a rank last in both directions', () => {
      const ascending = sortRoster([unranked, member, guildMaster], { key: 'rank', direction: 'asc' })
      const descending = sortRoster([unranked, member, guildMaster], { key: 'rank', direction: 'desc' })

      expect(ascending.map((c) => c.id)).toEqual(['gm', 'mem', 'none'])
      expect(descending.map((c) => c.id)).toEqual(['mem', 'gm', 'none'])
    })
  })
})
