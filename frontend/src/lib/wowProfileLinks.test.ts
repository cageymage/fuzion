import { describe, expect, it } from 'vitest'
import { armoryUrl, realmSlug, warcraftLogsUrl } from './wowProfileLinks'

describe('realmSlug', () => {
  it('should slugify a realm name with spaces into a hyphenated lowercase slug', () => {
    expect(realmSlug('Area 52')).toBe('area-52')
  })

  it('should drop apostrophes when the realm name has one', () => {
    expect(realmSlug("Mal'Ganis")).toBe('malganis')
  })
})

describe('armoryUrl', () => {
  it('should join the name and secondary name with a hyphen when the character has both', () => {
    expect(armoryUrl('Emberreach', 'Aeliana', 'Dawnsong')).toBe(
      'https://worldofwarcraft.blizzard.com/en-us/forever/us/armory/character/emberreach/aeliana-dawnsong',
    )
  })

  it('should build an armory URL with a lowercased character name when there is no secondary name', () => {
    expect(armoryUrl('Emberreach', 'Ragnok', '')).toBe(
      'https://worldofwarcraft.blizzard.com/en-us/forever/us/armory/character/emberreach/ragnok',
    )
  })

  it('should percent-encode the character name when it has accented letters', () => {
    expect(armoryUrl('Emberreach', 'Zuljïn', '')).toBe(
      'https://worldofwarcraft.blizzard.com/en-us/forever/us/armory/character/emberreach/zulj%C3%AFn',
    )
  })
})

describe('warcraftLogsUrl', () => {
  it('should build a warcraft logs URL with a lowercased character name', () => {
    expect(warcraftLogsUrl('Area 52', 'Ragnok', '')).toBe('https://www.warcraftlogs.com/character/us/area-52/ragnok')
  })

  it('should join the name and secondary name with a hyphen when the character has both', () => {
    expect(warcraftLogsUrl('Emberreach', 'Aeliana', 'Dawnsong')).toBe(
      'https://www.warcraftlogs.com/character/us/emberreach/aeliana-dawnsong',
    )
  })
})
