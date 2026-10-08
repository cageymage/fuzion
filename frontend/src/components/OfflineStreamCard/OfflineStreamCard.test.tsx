import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import type { Stream } from '../../types/streams'
import { OfflineStreamCard } from './OfflineStreamCard'

const offlineStream: Stream = {
  id: 'stream-2',
  streamerName: 'Moonveil',
  gameName: '',
  title: '',
  viewerCount: 0,
  thumbnailUrl: null,
  avatarUrl: 'https://cdn.example/moonveil.png',
  channelUrl: 'https://twitch.tv/moonveil',
  isLive: false,
  isLiveOtherGame: false,
}

describe('OfflineStreamCard', () => {
  it('should link to the channel in a new tab', () => {
    render(<OfflineStreamCard stream={offlineStream} />)

    const link = screen.getByRole('link', { name: 'Moonveil' })
    expect(link).toHaveAttribute('href', 'https://twitch.tv/moonveil')
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveAttribute('rel', 'noreferrer')
  })

  it('should show the streamer avatar when one exists', () => {
    const { container } = render(<OfflineStreamCard stream={offlineStream} />)

    expect(container.querySelector('img')).toHaveAttribute('src', 'https://cdn.example/moonveil.png')
  })

  it('should not show a LIVE badge or viewer count', () => {
    render(<OfflineStreamCard stream={offlineStream} />)

    expect(screen.queryByText('LIVE')).not.toBeInTheDocument()
    expect(screen.queryByText('0')).not.toBeInTheDocument()
  })
})
