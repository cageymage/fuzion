import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { liveStream } from '../../mocks/handlers'
import { LiveStreamCard } from './LiveStreamCard'

describe('LiveStreamCard', () => {
  it('should show the streamer, stream title and viewer count', () => {
    render(<LiveStreamCard stream={liveStream} />)

    expect(screen.getByText('Thundermane is live')).toBeInTheDocument()
    expect(screen.getByText('Mythic Queen Ansurek progress')).toBeInTheDocument()
    expect(screen.getByText('1.2K')).toBeInTheDocument()
  })

  it('should link to the channel on Twitch when the stream is on Twitch', () => {
    render(<LiveStreamCard stream={liveStream} />)

    expect(screen.getByRole('link', { name: 'Watch Thundermane on Twitch →' })).toHaveAttribute(
      'href',
      'https://twitch.tv/thundermane',
    )
  })

  it('should link to the channel on YouTube when the stream is on YouTube', () => {
    render(
      <LiveStreamCard
        stream={{
          ...liveStream,
          platform: 'youtube',
          channelUrl: 'https://www.youtube.com/@thundermane',
        }}
      />,
    )

    expect(screen.getByRole('link', { name: 'Watch Thundermane on YouTube →' })).toHaveAttribute(
      'href',
      'https://www.youtube.com/@thundermane',
    )
  })
})
