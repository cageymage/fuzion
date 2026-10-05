import type { CharacterProfession } from '../types/roster'

const iconBaseUrl = 'https://render.worldofwarcraft.com/us/icons/36'
const maxPrimaryProfessions = 2

// Jewelcrafting and Inscription are left out until Forever adds them.
const primaryIconNames: Record<string, string> = {
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
  const name = primaryIconNames[profession]
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
