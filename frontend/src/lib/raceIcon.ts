import { isRace, type Race } from './races'

const iconBaseUrl = 'https://render.worldofwarcraft.com/us/icons/36'

// Blizzard has no icon for Skyborne yet, so those races have none here.
const iconNames: Partial<Record<Race, string>> = {
  Human: 'achievement_character_human_male',
  Dwarf: 'achievement_character_dwarf_male',
  'Night Elf': 'achievement_character_nightelf_male',
  Gnome: 'achievement_character_gnome_male',
  Orc: 'achievement_character_orc_male',
  Undead: 'achievement_character_undead_male',
  Tauren: 'achievement_character_tauren_male',
  Troll: 'achievement_character_troll_male',
}

export function raceIcon(race: string | null): string | null {
  const name = race !== null && isRace(race) ? iconNames[race] : undefined
  return name ? `${iconBaseUrl}/${name}.jpg` : null
}
