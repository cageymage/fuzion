import { describe, expect, it } from 'vitest'
import { raceIcon } from './raceIcon'

describe('raceIcon', () => {
  it('should return an icon url when the race is known', () => {
    expect(raceIcon('Night Elf')).toBe(
      'https://render.worldofwarcraft.com/us/icons/36/achievement_character_nightelf_male.jpg',
    )
  })

  it('should return the female icon when the gender is female', () => {
    expect(raceIcon('Night Elf', 'female')).toBe(
      'https://render.worldofwarcraft.com/us/icons/36/achievement_character_nightelf_female.jpg',
    )
  })

  it('should return the male icon when the gender is unknown', () => {
    expect(raceIcon('Troll', null)).toBe(
      'https://render.worldofwarcraft.com/us/icons/36/achievement_character_troll_male.jpg',
    )
  })

  it('should return null when the race is not set', () => {
    expect(raceIcon(null)).toBeNull()
  })

  it('should return the blood elf icon when the race is Skyborne (Windshaper)', () => {
    expect(raceIcon('Skyborne (Windshaper)')).toBe(
      'https://render.worldofwarcraft.com/us/icons/36/achievement_character_bloodelf_male.jpg',
    )
  })

  it('should return null when the race is Skyborne (High Order), which has no icon yet', () => {
    expect(raceIcon('Skyborne (High Order)')).toBeNull()
  })

  it('should return null when the race is not playable', () => {
    expect(raceIcon('Murloc')).toBeNull()
  })
})
