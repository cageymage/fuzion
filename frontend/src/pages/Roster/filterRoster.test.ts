import { describe, expect, it } from 'vitest'
import type { Character } from '../../types/roster'
import { filterRoster } from './filterRoster'

const main: Character = {
  id: 'char-1',
  name: 'Ragnok',
  secondaryName: '',
  realm: 'Emberreach',
  class: 'Warrior',
  spec: 'Protection',
  role: 'tank',
  isMain: true,
  raidTeam: 'Team Alpha',
  createdAt: '2026-01-01T00:00:00.000Z',
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
}

const roster = [main, healer, alt]

describe('filterRoster', () => {
  it('should exclude alts when showAlts is false', () => {
    const result = filterRoster(roster, { role: 'all', className: 'all', showAlts: false })

    expect(result).toEqual([main, healer])
  })

  it('should include alts when showAlts is true', () => {
    const result = filterRoster(roster, { role: 'all', className: 'all', showAlts: true })

    expect(result).toEqual([main, healer, alt])
  })

  it('should only keep characters matching the selected role', () => {
    const result = filterRoster(roster, { role: 'healer', className: 'all', showAlts: false })

    expect(result).toEqual([healer])
  })

  it('should only keep characters matching the selected class', () => {
    const result = filterRoster(roster, { role: 'all', className: 'Priest', showAlts: false })

    expect(result).toEqual([healer])
  })

  it('should combine the role filter and the class filter', () => {
    const result = filterRoster(roster, { role: 'dps', className: 'Warlock', showAlts: true })

    expect(result).toEqual([alt])
  })
})
