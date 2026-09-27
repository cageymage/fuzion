import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import type { Stream } from '../../types/streams'
import { StreamCard } from './StreamCard'

const liveStream: Stream = {
  id: 'stream-1',
  streamerName: 'Thundermane',
  gameName: 'World of Warcraft: Forever',
  viewerCount: 1_240,
  thumbnailUrl: null,
  channelUrl: 'https://twitch.tv/thundermane',
  isLive: true,
}

describe('StreamCard', () => {
  it('should show the LIVE badge, viewer count and game when the stream is live', () => {
    render(<StreamCard stream={liveStream} />)

    expect(screen.getByText('LIVE')).toBeInTheDocument()
    expect(screen.getByText('1.2K')).toBeInTheDocument()
    expect(screen.getByText('World of Warcraft: Forever')).toBeInTheDocument()
  })

  it('should hide the LIVE badge and viewer count when the stream is offline', () => {
    render(<StreamCard stream={{ ...liveStream, isLive: false, viewerCount: 0 }} />)

    expect(screen.queryByText('LIVE')).not.toBeInTheDocument()
    expect(screen.queryByText('0')).not.toBeInTheDocument()
  })

  it('should link to the channel in a new tab', () => {
    render(<StreamCard stream={liveStream} />)

    const link = screen.getByRole('link', { name: /Thundermane/ })
    expect(link).toHaveAttribute('href', 'https://twitch.tv/thundermane')
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveAttribute('rel', 'noreferrer')
  })

  it('should show the preview thumbnail when the stream is live and has one', () => {
    const { container } = render(
      <StreamCard stream={{ ...liveStream, thumbnailUrl: 'https://cdn.example/preview.jpg' }} />,
    )

    expect(container.querySelector('img')).toHaveAttribute('src', 'https://cdn.example/preview.jpg')
  })

  it('should not show a thumbnail when the stream is offline', () => {
    const { container } = render(
      <StreamCard
        stream={{ ...liveStream, isLive: false, thumbnailUrl: 'https://cdn.example/preview.jpg' }}
      />,
    )

    expect(container.querySelector('img')).not.toBeInTheDocument()
  })
})
