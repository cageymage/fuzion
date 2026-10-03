import type { Character, Role } from '../../types/roster'

export const roleLabels: Record<Role, string> = {
  tank: 'Tank',
  healer: 'Healer',
  dps: 'DPS',
}

export function specLabel(character: Character): string {
  if (character.spec && character.spec2) return `${character.spec}/${character.spec2}`
  return character.spec ?? '—'
}

export function roleLabel(character: Character): string {
  if (character.role2 && character.role2 !== character.role) {
    return `${roleLabels[character.role]}/${roleLabels[character.role2]}`
  }
  return roleLabels[character.role]
}
