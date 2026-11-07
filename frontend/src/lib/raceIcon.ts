import { isRace, type Race } from './races'

const iconBaseUrl = 'https://render.worldofwarcraft.com/us/icons/36'

// Icon names without the gender suffix. Blizzard has no icon for Skyborne yet, so Windshaper
// borrows the blood elf icon as a stand-in and High Order has none.
const iconNames: Partial<Record<Race, string>> = {
  Human: 'achievement_character_human',
  Dwarf: 'achievement_character_dwarf',
  'Night Elf': 'achievement_character_nightelf',
  Gnome: 'achievement_character_gnome',
  Orc: 'achievement_character_orc',
  Undead: 'achievement_character_undead',
  Tauren: 'achievement_character_tauren',
  Troll: 'achievement_character_troll',
  'Skyborne (Windshaper)': 'achievement_character_bloodelf',
}

export function raceIcon(race: string | null, gender: 'male' | 'female' | null = null): string | null {
  const name = race !== null && isRace(race) ? iconNames[race] : undefined
  return name ? `${iconBaseUrl}/${name}_${gender ?? 'male'}.jpg` : null
}
