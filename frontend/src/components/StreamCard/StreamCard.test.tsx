import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import type { Stream } from '../../types/streams'
import { StreamCard } from './StreamCard'

const liveStream: Stream = {
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

describe('StreamCard', () => {
  it('should show the LIVE badge and viewer count', () => {
    render(<StreamCard stream={liveStream} />)

    expect(screen.getByText('LIVE')).toBeInTheDocument()
    expect(screen.getByText('1.2K')).toBeInTheDocument()
  })

  it('should not show the game name when the streamer is live in WoW', () => {
    render(<StreamCard stream={liveStream} />)

    expect(screen.queryByText('World of Warcraft: Forever')).not.toBeInTheDocument()
  })

  it('should show the game name when the streamer is live in another game', () => {
    render(
      <StreamCard
        stream={{ ...liveStream, isLive: false, isLiveOtherGame: true, gameName: 'Call of Duty: Warzone' }}
      />,
    )

    expect(screen.getByText('Call of Duty: Warzone')).toBeInTheDocument()
  })

  it('should show the stream title when the channel is live', () => {
    render(<StreamCard stream={liveStream} />)

    expect(screen.getByText('Mythic Queen Ansurek progress')).toBeInTheDocument()
  })

  it('should link to the channel in a new tab', () => {
    render(<StreamCard stream={liveStream} />)

    const link = screen.getByRole('link', { name: /Thundermane/ })
    expect(link).toHaveAttribute('href', 'https://twitch.tv/thundermane')
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveAttribute('rel', 'noreferrer')
  })

  it('should show the preview thumbnail when the stream has one', () => {
    const { container } = render(
      <StreamCard stream={{ ...liveStream, thumbnailUrl: 'https://cdn.example/preview.jpg' }} />,
    )

    expect(container.querySelector('img')).toHaveAttribute('src', 'https://cdn.example/preview.jpg')
  })

  it('should show the streamer avatar next to the name when the streamer has one', () => {
    const { container } = render(
      <StreamCard stream={{ ...liveStream, avatarUrl: 'https://cdn.example/thundermane.png' }} />,
    )

    expect(container.querySelector('img')).toHaveAttribute(
      'src',
      'https://cdn.example/thundermane.png',
    )
  })
})
