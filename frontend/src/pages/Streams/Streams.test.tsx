import { screen, within } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '../../mocks/server'
import { renderWithProviders } from '../../testUtils'
import type { Stream } from '../../types/streams'
import { Streams } from './Streams'

const offlineStream: Stream = {
  id: 'stream-9',
  streamerName: 'Emberfist',
  gameName: '',
  title: '',
  viewerCount: 0,
  thumbnailUrl: null,
  avatarUrl: null,
  channelUrl: 'https://twitch.tv/emberfist',
  isLive: false,
  isLiveOtherGame: false,
}

const liveStream: Stream = {
  id: 'stream-8',
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

const otherGameStream: Stream = {
  id: 'stream-7',
  streamerName: 'Aelith',
  gameName: 'Call of Duty: Warzone',
  title: 'Warzone night',
  viewerCount: 87,
  thumbnailUrl: null,
  avatarUrl: null,
  channelUrl: 'https://twitch.tv/aelith',
  isLive: false,
  isLiveOtherGame: true,
}

describe('Streams', () => {
  it('should show a LIVE badge only on channels that are currently live', async () => {
    renderWithProviders(<Streams />)
    await screen.findByRole('link', { name: /Moonveil/ })

    expect(screen.getAllByText('LIVE')).toHaveLength(1)
    expect(screen.getByRole('link', { name: /Thundermane/ })).toHaveTextContent('LIVE')
    expect(screen.getByRole('link', { name: /Moonveil/ })).not.toHaveTextContent('LIVE')
  })

  it('should show the stream title on live channels', async () => {
    renderWithProviders(<Streams />)

    const thundermane = await screen.findByRole('link', { name: /Thundermane/ })
    expect(thundermane).toHaveTextContent('Mythic Queen Ansurek progress')
  })

  it('should show the viewer count only on channels that are currently live', async () => {
    renderWithProviders(<Streams />)
    await screen.findByRole('link', { name: /Moonveil/ })

    expect(screen.getByRole('link', { name: /Thundermane/ })).toHaveTextContent('1.2K')
    expect(screen.getByRole('link', { name: /Moonveil/ })).not.toHaveTextContent('0')
  })

  it('should link each card to the channel URL in a new tab', async () => {
    renderWithProviders(<Streams />)

    const thundermane = await screen.findByRole('link', { name: /Thundermane/ })
    const moonveil = screen.getByRole('link', { name: /Moonveil/ })
    expect(thundermane).toHaveAttribute('href', 'https://twitch.tv/thundermane')
    expect(thundermane).toHaveAttribute('target', '_blank')
    expect(moonveil).toHaveAttribute('href', 'https://twitch.tv/moonveil')
    expect(moonveil).toHaveAttribute('target', '_blank')
  })

  it('should list live channels under Live in World of Warcraft and offline channels under Offline', async () => {
    renderWithProviders(<Streams />)

    const live = within(await screen.findByRole('region', { name: 'Live in World of Warcraft' }))
    const offline = within(screen.getByRole('region', { name: 'Offline' }))
    expect(live.getAllByRole('link').map((link) => link.getAttribute('href'))).toEqual([
      'https://twitch.tv/thundermane',
    ])
    expect(offline.getAllByRole('link').map((link) => link.getAttribute('href'))).toEqual([
      'https://twitch.tv/moonveil',
      'https://twitch.tv/aelith',
    ])
  })

  it('should show the streamer avatar on an offline card when the streamer has one', async () => {
    renderWithProviders(<Streams />)

    const aelith = await screen.findByRole('link', { name: 'Aelith' })
    expect(aelith.querySelector('img')).toHaveAttribute('src', 'https://cdn.example/aelith.png')
  })

  it('should show the streamer initial on an offline card when the streamer has no avatar', async () => {
    renderWithProviders(<Streams />)

    const moonveil = await screen.findByRole('link', { name: 'Moonveil' })
    expect(moonveil.querySelector('img')).not.toBeInTheDocument()
    expect(within(moonveil).getByText('M')).toBeInTheDocument()
  })

  it('should show a nobody-is-live message when every channel is offline', async () => {
    server.use(http.get('/api/streams', () => HttpResponse.json([offlineStream])))

    renderWithProviders(<Streams />)

    expect(await screen.findByText('Nobody is live right now.')).toBeInTheDocument()
    expect(screen.queryByRole('region', { name: 'Live in World of Warcraft' })).not.toBeInTheDocument()
    expect(
      within(screen.getByRole('region', { name: 'Offline' })).getByRole('link', {
        name: 'Emberfist',
      }),
    ).toBeInTheDocument()
  })

  it('should not show the nobody-is-live message when a streamer is live in another game', async () => {
    server.use(http.get('/api/streams', () => HttpResponse.json([otherGameStream, offlineStream])))

    renderWithProviders(<Streams />)

    await screen.findByRole('region', { name: 'Live in another game' })
    expect(screen.queryByText('Nobody is live right now.')).not.toBeInTheDocument()
  })

  it('should not show the Offline section when every channel is live', async () => {
    server.use(http.get('/api/streams', () => HttpResponse.json([liveStream])))

    renderWithProviders(<Streams />)

    await screen.findByRole('region', { name: 'Live in World of Warcraft' })
    expect(screen.queryByRole('region', { name: 'Offline' })).not.toBeInTheDocument()
    expect(screen.queryByText('Nobody is live right now.')).not.toBeInTheDocument()
  })

  it('should show a streamer in the other-game section when they are live in a non-WoW game', async () => {
    server.use(http.get('/api/streams', () => HttpResponse.json([liveStream, otherGameStream, offlineStream])))

    renderWithProviders(<Streams />)

    const otherGame = within(await screen.findByRole('region', { name: 'Live in another game' }))
    expect(otherGame.getAllByRole('link').map((link) => link.getAttribute('href'))).toEqual([
      'https://twitch.tv/aelith',
    ])
    expect(otherGame.getByRole('link', { name: /Aelith/ })).toHaveTextContent('Call of Duty: Warzone')
    const offline = within(screen.getByRole('region', { name: 'Offline' }))
    expect(offline.queryByRole('link', { name: /Aelith/ })).not.toBeInTheDocument()
  })

  it('should hide the other-game section when nobody is playing another game', async () => {
    renderWithProviders(<Streams />)

    await screen.findByRole('region', { name: 'Live in World of Warcraft' })
    expect(screen.queryByRole('region', { name: 'Live in another game' })).not.toBeInTheDocument()
  })

  it('should list a streamer only under offline when they are not live', async () => {
    server.use(http.get('/api/streams', () => HttpResponse.json([otherGameStream, offlineStream])))

    renderWithProviders(<Streams />)

    const offline = within(await screen.findByRole('region', { name: 'Offline' }))
    expect(offline.getAllByRole('link').map((link) => link.getAttribute('href'))).toEqual([
      'https://twitch.tv/emberfist',
    ])
    expect(
      within(screen.getByRole('region', { name: 'Live in another game' })).queryByRole('link', {
        name: /Emberfist/,
      }),
    ).not.toBeInTheDocument()
  })

  it('should show a loading message when the request is still in flight', () => {
    renderWithProviders(<Streams />)

    expect(screen.getByText('Loading streams…')).toBeInTheDocument()
  })

  it('should show an empty message when no channels exist', async () => {
    server.use(http.get('/api/streams', () => HttpResponse.json([])))

    renderWithProviders(<Streams />)

    expect(await screen.findByText('No one has added a stream channel yet.')).toBeInTheDocument()
  })

  it('should show an error message when the streams request fails', async () => {
    server.use(http.get('/api/streams', () => new HttpResponse(null, { status: 500 })))

    renderWithProviders(<Streams />)

    expect(await screen.findByText('Streams could not be loaded.')).toBeInTheDocument()
  })
})
