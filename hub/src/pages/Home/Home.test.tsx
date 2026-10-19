import { screen, within } from '@testing-library/react'
import { delay, http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { liveStream } from '../../mocks/handlers'
import { server } from '../../mocks/server'
import { renderWithProviders } from '../../testUtils'
import { Home } from './Home'

describe('Home', () => {
  it('should link to the Forever guild site', () => {
    renderWithProviders(<Home />)

    const guilds = screen.getByRole('region', { name: 'Our guilds' })
    expect(within(guilds).getByRole('link', { name: /Visit forever\.fuziongaming\.gg/ })).toHaveAttribute(
      'href',
      'https://forever.fuziongaming.gg',
    )
  })

  it.each([
    ['Twitch', 'https://twitch.tv/centrifuze'],
    ['YouTube', 'https://www.youtube.com/@Centrifuze'],
  ])('should link to Fuzion on %s in a new tab', (platform, url) => {
    renderWithProviders(<Home />)

    const link = within(screen.getByRole('region', { name: 'Find us' })).getByRole('link', {
      name: platform,
    })
    expect(link).toHaveAttribute('href', url)
    expect(link).toHaveAttribute('target', '_blank')
  })

  it('should say it is checking who is live when the live streams request is pending', () => {
    server.use(http.get('/api/streams/live', () => delay('infinite')))

    renderWithProviders(<Home />)

    expect(screen.getByText('Checking who is live…')).toBeInTheDocument()
  })

  it('should show every live stream when several people are streaming', async () => {
    server.use(
      http.get('/api/streams/live', () =>
        HttpResponse.json([
          liveStream,
          { ...liveStream, id: 'stream-2', streamerName: 'Moonveil', channelUrl: 'https://twitch.tv/moonveil' },
        ]),
      ),
    )

    renderWithProviders(<Home />)

    expect(await screen.findByText('Thundermane is live')).toBeInTheDocument()
    expect(screen.getByText('Moonveil is live')).toBeInTheDocument()
  })

  it('should title the section Live Now when someone is streaming', async () => {
    renderWithProviders(<Home />)

    await screen.findByText('Thundermane is live')
    expect(screen.getByRole('heading', { level: 2, name: 'Live Now' })).toBeInTheDocument()
  })

  it('should embed a suggested playlist video when no one is live', async () => {
    server.use(http.get('/api/streams/live', () => HttpResponse.json([])))

    renderWithProviders(<Home />)

    const titled = await screen.findAllByTitle('Queen Ansurek kill')
    const player = titled.find((element) => element.tagName === 'IFRAME')
    expect(player).toHaveAttribute('src', 'https://www.youtube.com/embed/vid-1')
  })

  it('should title the section From the Vault when a playlist video is suggested', async () => {
    server.use(http.get('/api/streams/live', () => HttpResponse.json([])))

    renderWithProviders(<Home />)

    expect(
      await screen.findByRole('heading', { level: 2, name: 'From the Vault' }),
    ).toBeInTheDocument()
  })

  it('should say nobody is streaming when no one is live and no video is available', async () => {
    server.use(
      http.get('/api/streams/live', () => HttpResponse.json([])),
      http.get('/api/streams/suggested-video', () => new HttpResponse(null, { status: 204 })),
    )

    renderWithProviders(<Home />)

    expect(await screen.findByText('Nobody is streaming right now.')).toBeInTheDocument()
  })

  it('should say nobody is streaming when no one is live and the suggested video request fails', async () => {
    server.use(
      http.get('/api/streams/live', () => HttpResponse.json([])),
      http.get('/api/streams/suggested-video', () => new HttpResponse(null, { status: 500 })),
    )

    renderWithProviders(<Home />)

    expect(await screen.findByText('Nobody is streaming right now.')).toBeInTheDocument()
  })

  it('should say live streams could not be loaded when the live streams request fails', async () => {
    server.use(http.get('/api/streams/live', () => new HttpResponse(null, { status: 500 })))

    renderWithProviders(<Home />)

    expect(await screen.findByText('Live streams could not be loaded.')).toBeInTheDocument()
  })
})
