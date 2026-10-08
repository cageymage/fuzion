import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import type { Stream } from '../../types/streams'
import { LiveStreamCard } from './LiveStreamCard'

const stream: Stream = {
  id: 'stream-1',
  streamerName: 'Thundermane',
  gameName: 'World of Warcraft: Forever',
  title: 'Mythic Queen Ansurek progress',
  viewerCount: 1_240,
  thumbnailUrl: null,
  avatarUrl: null,
  channelUrl: 'https://twitch.tv/thundermane',
  isLive: true,
  isLiveOtherGame: false,
}

describe('LiveStreamCard', () => {
  it('should name the streamer', () => {
    render(<LiveStreamCard stream={stream} />)

    expect(screen.getByText('Thundermane is live')).toBeInTheDocument()
  })

  it('should not show the game name', () => {
    render(<LiveStreamCard stream={stream} />)

    expect(screen.queryByText('World of Warcraft: Forever')).not.toBeInTheDocument()
  })

  it('should show the stream title', () => {
    render(<LiveStreamCard stream={stream} />)

    expect(screen.getByText('Mythic Queen Ansurek progress')).toBeInTheDocument()
  })

  it('should show the viewer count abbreviated in thousands', () => {
    render(<LiveStreamCard stream={stream} />)

    expect(screen.getByText('1.2K')).toBeInTheDocument()
  })

  it('should show the streamer avatar next to the name when the streamer has one', () => {
    const { container } = render(
      <LiveStreamCard stream={{ ...stream, avatarUrl: 'https://cdn.example/thundermane.png' }} />,
    )

    expect(container.querySelector('img')).toHaveAttribute(
      'src',
      'https://cdn.example/thundermane.png',
    )
  })

  it('should show the streamer initial when the streamer has no avatar', () => {
    const { container } = render(<LiveStreamCard stream={stream} />)

    expect(container.querySelector('img')).not.toBeInTheDocument()
    expect(screen.getByText('T')).toBeInTheDocument()
  })

  it('should link to the channel in a new tab', () => {
    render(<LiveStreamCard stream={stream} />)

    const link = screen.getByRole('link', { name: 'Watch on Twitch →' })
    expect(link).toHaveAttribute('href', 'https://twitch.tv/thundermane')
    expect(link).toHaveAttribute('target', '_blank')
  })
})
