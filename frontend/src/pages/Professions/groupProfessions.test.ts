import { describe, expect, it } from 'vitest'
import type { ProfessionEntry } from '../../types/professions'
import { groupProfessions } from './groupProfessions'

function entry(
  id: string,
  profession: string,
  skillLevel: number,
  name: string,
  secondaryName = '',
): ProfessionEntry {
  return {
    id,
    profession,
    skillLevel,
    character: { id: `char-${name}`, name, secondaryName, class: 'Priest' },
  }
}

const alchemyAeliana = entry('p1', 'Alchemy', 300, 'Aeliana', 'Dawnsong')
const alchemyZephyrion = entry('p2', 'Alchemy', 225, 'Zephyrion', 'Ironhide')
const smithingZephyrion = entry('p3', 'Blacksmithing', 300, 'Zephyrion', 'Ironhide')
const rows = [alchemyAeliana, alchemyZephyrion, smithingZephyrion]

describe('groupProfessions', () => {
  it('should group rows by profession and keep the api order when the query is empty', () => {
    expect(groupProfessions(rows, '')).toEqual([
      { profession: 'Alchemy', entries: [alchemyAeliana, alchemyZephyrion] },
      { profession: 'Blacksmithing', entries: [smithingZephyrion] },
    ])
  })

  it('should keep every row of a profession when the query matches the profession name', () => {
    expect(groupProfessions(rows, 'alch')).toEqual([
      { profession: 'Alchemy', entries: [alchemyAeliana, alchemyZephyrion] },
    ])
  })

  it('should keep only the matching rows across professions when the query matches a character name', () => {
    expect(groupProfessions(rows, 'ZEPHYRION')).toEqual([
      { profession: 'Alchemy', entries: [alchemyZephyrion] },
      { profession: 'Blacksmithing', entries: [smithingZephyrion] },
    ])
  })

  it('should match the secondary name when the query matches only the secondary name', () => {
    expect(groupProfessions(rows, 'dawn')).toEqual([
      { profession: 'Alchemy', entries: [alchemyAeliana] },
    ])
  })

  it('should ignore surrounding whitespace when the query has leading or trailing spaces', () => {
    expect(groupProfessions(rows, '  smith  ')).toEqual([
      { profession: 'Blacksmithing', entries: [smithingZephyrion] },
    ])
  })

  it('should return no groups when nothing matches the query', () => {
    expect(groupProfessions(rows, 'tailoring')).toEqual([])
  })

  it('should return no groups when there are no rows', () => {
    expect(groupProfessions([], '')).toEqual([])
  })
})
