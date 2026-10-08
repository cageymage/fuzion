import type { Character, Role } from '../../types/roster'

export type RoleFilter = Role | 'all'
export type ClassFilter = string | 'all'

export interface RosterFilters {
  role: RoleFilter
  className: ClassFilter
  showAlts: boolean
  search: string
}

export type SortKey = 'name' | 'race' | 'level' | 'class' | 'role' | 'raidTeam'
export type SortDirection = 'asc' | 'desc'

export interface RosterSort {
  key: SortKey
  direction: SortDirection
}

export function filterRoster(characters: Character[], filters: RosterFilters): Character[] {
  const search = filters.search.trim().toLowerCase()
  return characters.filter((character) => {
    if (!filters.showAlts && !character.isMain) return false
    if (filters.role !== 'all' && character.role !== filters.role && character.role2 !== filters.role) {
      return false
    }
    if (filters.className !== 'all' && character.class !== filters.className) return false
    if (
      search !== '' &&
      !character.name.toLowerCase().includes(search) &&
      !character.secondaryName.toLowerCase().includes(search)
    ) {
      return false
    }
    return true
  })
}

function compareValues(a: string | number | null, b: string | number | null): number {
  if (typeof a === 'number' && typeof b === 'number') return a - b
  return String(a).localeCompare(String(b))
}

function sortValue(character: Character, key: SortKey): string | number | null {
  switch (key) {
    case 'name':
      return `${character.name} ${character.secondaryName}`
    case 'race':
      return character.race
    case 'level':
      return character.level
    case 'class':
      return character.class
    case 'role':
      return character.role
    case 'raidTeam':
      return character.raidTeam
  }
}

// Characters with no value for the sorted column stay at the bottom in both directions.
export function sortRoster(characters: Character[], sort: RosterSort | null): Character[] {
  if (sort === null) return characters
  const sign = sort.direction === 'asc' ? 1 : -1
  return [...characters].sort((a, b) => {
    const left = sortValue(a, sort.key)
    const right = sortValue(b, sort.key)
    if (left === null && right === null) return 0
    if (left === null) return 1
    if (right === null) return -1
    return sign * compareValues(left, right)
  })
}
