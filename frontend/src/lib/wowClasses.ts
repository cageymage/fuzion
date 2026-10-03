export const wowClasses = [
  'Warrior',
  'Paladin',
  'Hunter',
  'Rogue',
  'Priest',
  'Shaman',
  'Mage',
  'Warlock',
  'Druid',
] as const

export type WowClass = (typeof wowClasses)[number]

// Mirrors classSpecs in backend/internal/roster/service.go, which is the source of truth.
export const classSpecs: Record<WowClass, readonly string[]> = {
  Warrior: ['Arms', 'Fury', 'Protection'],
  Paladin: ['Holy', 'Protection', 'Retribution'],
  Hunter: ['Beast Mastery', 'Marksmanship', 'Survival'],
  Rogue: ['Assassination', 'Combat', 'Subtlety'],
  Priest: ['Discipline', 'Holy', 'Shadow'],
  Shaman: ['Elemental', 'Enhancement', 'Restoration'],
  Mage: ['Arcane', 'Fire', 'Frost'],
  Warlock: ['Affliction', 'Demonology', 'Destruction'],
  Druid: ['Balance', 'Feral', 'Restoration'],
}
