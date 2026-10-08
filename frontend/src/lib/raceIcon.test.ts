import { describe, expect, it } from 'vitest'
import { raceIcon } from './raceIcon'

describe('raceIcon', () => {
  it('should return an icon url when the race is known', () => {
    expect(raceIcon('Night Elf')).toBe(
      'https://render.worldofwarcraft.com/us/icons/36/achievement_character_nightelf_male.jpg',
    )
  })

  it('should return null when the race is not set', () => {
    expect(raceIcon(null)).toBeNull()
  })

  it.each(['Skyborne (High Order)', 'Skyborne (Windshaper)'])(
    'should return null when the race is %s, which has no Blizzard icon',
    (race) => {
      expect(raceIcon(race)).toBeNull()
    },
  )

  it('should return null when the race is not playable', () => {
    expect(raceIcon('Murloc')).toBeNull()
  })
})
