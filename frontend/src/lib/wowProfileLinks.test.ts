import { describe, expect, it } from 'vitest'
import { armoryUrl, warcraftLogsUrl } from './wowProfileLinks'

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

  it('should percent-encode dots when the name is only dots', () => {
    expect(armoryUrl('..', '')).toBe('https://worldofwarcraft.blizzard.com/en-us/forever/us/armory/character/%2E%2E')
  })
})

describe('warcraftLogsUrl', () => {
  it('should join the name and secondary name with a space under the normal segment when the character has both', () => {
    expect(warcraftLogsUrl('Terrorskull', 'Gloomstep')).toBe(
      'https://forever.warcraftlogs.com/character/us/normal/terrorskull%20gloomstep',
    )
  })

  it('should build a warcraft logs URL with a lowercased character name when there is no secondary name', () => {
    expect(warcraftLogsUrl('Ragnok', '')).toBe('https://forever.warcraftlogs.com/character/us/normal/ragnok')
  })

  it('should percent-encode the character name when it has accented letters', () => {
    expect(warcraftLogsUrl('Zuljïn', '')).toBe('https://forever.warcraftlogs.com/character/us/normal/zulj%C3%AFn')
  })

  it('should percent-encode dots when the name is only dots', () => {
    expect(warcraftLogsUrl('..', '')).toBe('https://forever.warcraftlogs.com/character/us/normal/%2E%2E')
  })
})
