import type { CharacterProfession } from '../types/roster'
import { isPrimaryProfession, type PrimaryProfession } from './professions'

const iconBaseUrl = 'https://render.worldofwarcraft.com/us/icons/36'
const maxPrimaryProfessions = 2

const primaryIconNames: Record<PrimaryProfession, string> = {
  Alchemy: 'trade_alchemy',
  Blacksmithing: 'trade_blacksmithing',
  Enchanting: 'trade_engraving',
  Engineering: 'trade_engineering',
  Herbalism: 'trade_herbalism',
  Leatherworking: 'inv_misc_armorkit_17',
  Mining: 'trade_mining',
  Skinning: 'inv_misc_pelt_wolf_01',
  Tailoring: 'trade_tailoring',
}

export interface ProfessionIcon {
  profession: string
  iconUrl: string
}

export function professionIcon(profession: string): string | null {
  const name = isPrimaryProfession(profession) ? primaryIconNames[profession] : null
  return name ? `${iconBaseUrl}/${name}.jpg` : null
}

export function primaryProfessionIcons(professions: CharacterProfession[]): ProfessionIcon[] {
  const icons: ProfessionIcon[] = []
  for (const { profession } of professions) {
    const iconUrl = professionIcon(profession)
    if (iconUrl) icons.push({ profession, iconUrl })
  }
  return icons.slice(0, maxPrimaryProfessions)
}
