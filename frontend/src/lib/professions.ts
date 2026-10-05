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

export function isPrimaryProfession(name: string): name is PrimaryProfession {
  return (primaryProfessions as readonly string[]).includes(name)
}

export function isSecondaryProfession(name: string): boolean {
  return (secondaryProfessions as readonly string[]).includes(name)
}
