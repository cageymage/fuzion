import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import type { LiveStream } from '../../types/streams'
import { LiveStreamCard } from './LiveStreamCard'

const stream: LiveStream = {
  id: 'stream-1',
  streamerName: 'Thundermane',
  gameName: 'World of Warcraft: Forever',
  viewerCount: 1_240,
  thumbnailUrl: null,
  channelUrl: 'https://twitch.tv/thundermane',
}

describe('LiveStreamCard', () => {
  it('should name the streamer and the game being played', () => {
    render(<LiveStreamCard stream={stream} />)

    expect(screen.getByText('Thundermane is live')).toBeInTheDocument()
    expect(screen.getByText('World of Warcraft: Forever')).toBeInTheDocument()
  })

  it('should show the viewer count abbreviated in thousands', () => {
    render(<LiveStreamCard stream={stream} />)

    expect(screen.getByText('1.2K')).toBeInTheDocument()
  })

  it('should link to the channel in a new tab', () => {
    render(<LiveStreamCard stream={stream} />)

    const link = screen.getByRole('link', { name: 'Watch on Twitch →' })
    expect(link).toHaveAttribute('href', 'https://twitch.tv/thundermane')
    expect(link).toHaveAttribute('target', '_blank')
  })
})
