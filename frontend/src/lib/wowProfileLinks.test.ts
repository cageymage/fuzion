import { describe, expect, it } from 'vitest'
import { armoryUrl, realmSlug } from './wowProfileLinks'

describe('realmSlug', () => {
  it('should slugify a realm name with spaces into a hyphenated lowercase slug', () => {
    expect(realmSlug('Area 52')).toBe('area-52')
  })

  it('should drop apostrophes when the realm name has one', () => {
    expect(realmSlug("Mal'Ganis")).toBe('malganis')
  })
})

describe('armoryUrl', () => {
  it('should build an armory URL with a lowercased character name', () => {
    expect(armoryUrl('Emberreach', 'Ragnok')).toBe(
      'https://worldofwarcraft.blizzard.com/en-us/forever/us/armory/character/emberreach/ragnok',
    )
  })

  it('should percent-encode the character name when it has accented letters', () => {
    expect(armoryUrl('Emberreach', 'Zuljïn')).toBe(
      'https://worldofwarcraft.blizzard.com/en-us/forever/us/armory/character/emberreach/zulj%C3%AFn',
    )
  })
})
