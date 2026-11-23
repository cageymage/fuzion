// Jewelcrafting and Inscription are left out until Forever adds them.
export const primaryProfessions = [
  'Alchemy',
  'Blacksmithing',
  'Enchanting',
  'Engineering',
  'Herbalism',
  'Leatherworking',
  'Mining',
  'Skinning',
  'Tailoring',
] as const

export const secondaryProfessions = ['Cooking', 'Fishing', 'First Aid'] as const

export type PrimaryProfession = (typeof primaryProfessions)[number]

// Mirrors specializations in backend/internal/professions/specialization.go, which is the source of truth.
const specializations: Partial<Record<string, readonly string[]>> = {
  Blacksmithing: ['Armorsmith', 'Weaponsmith'],
  Engineering: ['Gnomish', 'Goblin'],
  Leatherworking: ['Dragonscale', 'Elemental', 'Tribal'],
}

export function specializationsOf(profession: string): readonly string[] {
  return specializations[profession] ?? []
}

export function isPrimaryProfession(name: string): name is PrimaryProfession {
  return (primaryProfessions as readonly string[]).includes(name)
}

export function isSecondaryProfession(name: string): boolean {
  return (secondaryProfessions as readonly string[]).includes(name)
}
