import { describe, expect, it } from 'vitest'
import { armoryUrl } from './wowProfileLinks'

describe('armoryUrl', () => {
  it('should join the name and secondary name with a hyphen when the character has both', () => {
    expect(armoryUrl('Psyence', 'Spiritbound')).toBe(
      'https://worldofwarcraft.blizzard.com/en-us/forever/us/armory/character/psyence-spiritbound',
    )
  })

  it('should build an armory URL with a lowercased character name when there is no secondary name', () => {
    expect(armoryUrl('Ragnok', '')).toBe(
      'https://worldofwarcraft.blizzard.com/en-us/forever/us/armory/character/ragnok',
    )
  })

  it('should percent-encode the character name when it has accented letters', () => {
    expect(armoryUrl('Zuljïn', '')).toBe(
      'https://worldofwarcraft.blizzard.com/en-us/forever/us/armory/character/zulj%C3%AFn',
    )
  })
})
