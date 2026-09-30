import { describe, expect, it } from 'vitest'
import { twitchChannelFromUrl } from './twitch'

describe('twitchChannelFromUrl', () => {
  it('should return the channel login when the URL is a plain twitch.tv channel link', () => {
    expect(twitchChannelFromUrl('https://twitch.tv/thundermane')).toBe('thundermane')
  })

  it('should return the channel login when the host is www.twitch.tv', () => {
    expect(twitchChannelFromUrl('https://www.twitch.tv/centrifuze')).toBe('centrifuze')
  })

  it('should return the channel login when the URL has a trailing slash', () => {
    expect(twitchChannelFromUrl('https://twitch.tv/moonveil/')).toBe('moonveil')
  })

  it('should return null when the URL has no path', () => {
    expect(twitchChannelFromUrl('https://twitch.tv/')).toBeNull()
  })

  it('should return null when the host is not twitch', () => {
    expect(twitchChannelFromUrl('https://example.com/thundermane')).toBeNull()
  })

  it('should return null when the host only ends with twitch.tv', () => {
    expect(twitchChannelFromUrl('https://nottwitch.tv/thundermane')).toBeNull()
  })

  it('should return null when the first path segment is not a valid login', () => {
    expect(twitchChannelFromUrl('https://twitch.tv/some%20thing')).toBeNull()
  })

  it('should return null when the value is not a URL', () => {
    expect(twitchChannelFromUrl('not a url')).toBeNull()
  })
})
