// Mirrors the races the backend accepts. Skyborne can join either faction, so each
// faction's variant is its own race.
export const races = [
  'Human',
  'Dwarf',
  'Night Elf',
  'Gnome',
  'Skyborne (High Order)',
  'Orc',
  'Undead',
  'Tauren',
  'Troll',
  'Skyborne (Windshaper)',
] as const

export type Race = (typeof races)[number]

export const maxLevel = 60

export function isRace(name: string): name is Race {
  return (races as readonly string[]).includes(name)
}
