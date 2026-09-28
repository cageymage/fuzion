import type { Character, Role } from '../../types/roster'

export type RoleFilter = Role | 'all'
export type ClassFilter = string | 'all'

export interface RosterFilters {
  role: RoleFilter
  className: ClassFilter
  showAlts: boolean
}

export function filterRoster(characters: Character[], filters: RosterFilters): Character[] {
  return characters.filter((character) => {
    if (!filters.showAlts && !character.isMain) return false
    if (filters.role !== 'all' && character.role !== filters.role) return false
    if (filters.className !== 'all' && character.class !== filters.className) return false
    return true
  })
}
