import { screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { newsPage } from '../../mocks/handlers'
import { server } from '../../mocks/server'
import { renderWithProviders } from '../../testUtils'
import type { Stream } from '../../types/streams'
import { Home } from './Home'

const centrifuze: Stream = {
  id: 'stream-2',
  streamerName: 'Centrifuze',
  gameName: 'World of Warcraft',
  title: 'Raid night prep',
  viewerCount: 87,
  thumbnailUrl: null,
  avatarUrl: null,
  channelUrl: 'https://www.twitch.tv/centrifuze',
  isLive: true,
  isLiveOtherGame: false,
}

describe('Home', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('should show the guild name, realm and tagline', async () => {
    renderWithProviders(<Home />)

    expect(await screen.findByRole('heading', { name: 'Fuzion', level: 1 })).toBeInTheDocument()
    expect(screen.getByText('PvE · US')).toBeInTheDocument()
    expect(screen.getByText('Raiding · Dungeons · Community')).toBeInTheDocument()
  })

  it('should show the hero image as decorative with an empty alt text', async () => {
    const { container } = renderWithProviders(<Home />)

    await screen.findByRole('heading', { name: 'Fuzion', level: 1 })
    const hero = container.querySelector('picture img')
    expect(hero).toHaveAttribute('alt', '')
    expect(hero).toHaveAttribute('width', '2400')
    expect(hero).toHaveAttribute('height', '900')
  })

  it('should use the first image in the newest post body as the featured thumbnail', async () => {
    server.use(
      http.get('/api/news', () =>
        HttpResponse.json(
          newsPage([
            {
              id: 'post-1',
              title: 'Server first',
              excerpt: 'We did it.',
              category: 'raid-progress',
              imageUrl: null,
              authorName: 'Officer',
              publishedAt: new Date().toISOString(),
              body: 'Intro\n\n![Kill](https://cdn.example/kill.png)',
            },
          ]),
        ),
      ),
    )

    const { container } = renderWithProviders(<Home />)

    await screen.findByRole('heading', { name: 'Server first' })
    expect(container.querySelector('img[src="https://cdn.example/kill.png"]')).not.toBeNull()
  })

  it('should keep the placeholder graph as the featured thumbnail when the post has no image', async () => {
    const { container } = renderWithProviders(<Home />)

    await screen.findByRole('heading', { name: 'Fuzion defeated Queen Ansurek on Mythic' })
    const placeholderGraph = container.querySelector('svg[viewBox="0 0 210 130"]')
    expect(placeholderGraph).not.toBeNull()
    expect(placeholderGraph?.parentElement?.querySelector('img')).toBeNull()
  })

  it('should show the countdown to the next raid when one is scheduled', async () => {
    renderWithProviders(<Home />)

    expect(await screen.findByText('2h 14m')).toBeInTheDocument()
    expect(screen.getByText("Mythic · Nerub'ar Palace")).toBeInTheDocument()
  })

  it('should show a no-raid message when no raid is scheduled', async () => {
    server.use(http.get('/api/raids/next', () => HttpResponse.json(null)))

    renderWithProviders(<Home />)

    expect(await screen.findByText('No raid scheduled yet.')).toBeInTheDocument()
  })

  it('should feature the newest news post above the remaining posts', async () => {
    renderWithProviders(<Home />)

    expect(
      await screen.findByRole('heading', { name: 'Fuzion defeated Queen Ansurek on Mythic' }),
    ).toBeInTheDocument()
    expect(screen.getByText(/a huge night for the team/)).toBeInTheDocument()
    expect(
      screen.getByRole('heading', { name: 'Now recruiting: Restoration Druid & Fire Mage' }),
    ).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Welcome our newest officers' })).toBeInTheDocument()
  })

  it('should request only the three most recent news posts', async () => {
    let requestedLimit: string | null = null
    server.use(
      http.get('/api/news', ({ request }) => {
        requestedLimit = new URL(request.url).searchParams.get('limit')
        return HttpResponse.json(newsPage([]))
      }),
    )

    renderWithProviders(<Home />)

    expect(await screen.findByText('No news posted yet.')).toBeInTheDocument()
    expect(requestedLimit).toBe('3')
  })

  it('should link the news section to the full news page', async () => {
    renderWithProviders(<Home />)

    expect(await screen.findByRole('link', { name: 'View all →' })).toHaveAttribute(
      'href',
      '/news',
    )
  })

  it('should show an error message when the news request fails', async () => {
    server.use(http.get('/api/news', () => new HttpResponse(null, { status: 500 })))

    renderWithProviders(<Home />)

    expect(await screen.findByText('Latest news could not be loaded.')).toBeInTheDocument()
  })

  it('should show the live stream with its viewer count when someone is streaming', async () => {
    renderWithProviders(<Home />)

    expect(await screen.findByText('Thundermane is live')).toBeInTheDocument()
    expect(screen.getByText('1.2K')).toBeInTheDocument()
  })

  it('should show only the first live stream when the random pick lands on the first', async () => {
    vi.spyOn(Math, 'random').mockReturnValue(0)
    server.use(
      http.get('/api/streams/live', () =>
        HttpResponse.json([
          { ...centrifuze, id: 'stream-1', streamerName: 'Thundermane' },
          centrifuze,
        ]),
      ),
    )

    renderWithProviders(<Home />)

    expect(await screen.findByText('Thundermane is live')).toBeInTheDocument()
    expect(screen.queryByText('Centrifuze is live')).not.toBeInTheDocument()
  })

  it('should show only the last live stream when the random pick lands on the last', async () => {
    vi.spyOn(Math, 'random').mockReturnValue(0.99)
    server.use(
      http.get('/api/streams/live', () =>
        HttpResponse.json([
          { ...centrifuze, id: 'stream-1', streamerName: 'Thundermane' },
          centrifuze,
        ]),
      ),
    )

    renderWithProviders(<Home />)

    expect(await screen.findByText('Centrifuze is live')).toBeInTheDocument()
    expect(screen.queryByText('Thundermane is live')).not.toBeInTheDocument()
  })

  it('should show the title of the featured live stream', async () => {
    server.use(http.get('/api/streams/live', () => HttpResponse.json([centrifuze])))

    renderWithProviders(<Home />)

    expect(await screen.findByText('Raid night prep')).toBeInTheDocument()
  })

  it('should embed the featured live stream muted and autoplaying when someone is live', async () => {
    server.use(http.get('/api/streams/live', () => HttpResponse.json([centrifuze])))

    renderWithProviders(<Home />)

    const player = await screen.findByTitle('Centrifuze live stream')
    const playerUrl = new URL(player.getAttribute('src') ?? '')
    expect(playerUrl.origin + playerUrl.pathname).toBe('https://player.twitch.tv/')
    expect(Object.fromEntries(playerUrl.searchParams)).toEqual({
      channel: 'centrifuze',
      parent: 'localhost',
      muted: 'true',
      autoplay: 'true',
    })
  })

  it('should keep a link to the channel on Twitch when the stream is embedded', async () => {
    server.use(http.get('/api/streams/live', () => HttpResponse.json([centrifuze])))

    renderWithProviders(<Home />)

    expect(await screen.findByRole('link', { name: 'Watch on Twitch →' })).toHaveAttribute(
      'href',
      'https://www.twitch.tv/centrifuze',
    )
  })

  it('should fall back to the stream card when the channel URL has no channel name', async () => {
    server.use(
      http.get('/api/streams/live', () =>
        HttpResponse.json([{ ...centrifuze, channelUrl: 'https://www.twitch.tv/' }]),
      ),
    )

    renderWithProviders(<Home />)

    expect(await screen.findByText('Centrifuze is live')).toBeInTheDocument()
    expect(screen.queryByTitle('Centrifuze live stream')).not.toBeInTheDocument()
  })

  it('should suggest a playlist video as a click-to-play embed when no one is live', async () => {
    server.use(http.get('/api/streams/live', () => HttpResponse.json([])))

    renderWithProviders(<Home />)

    const titled = await screen.findAllByTitle('Queen Ansurek kill')
    const player = titled.find((element) => element.tagName === 'IFRAME')
    expect(player).toHaveAttribute('src', 'https://www.youtube.com/embed/vid-1')
    expect(screen.queryByText('Nobody is streaming right now.')).not.toBeInTheDocument()
  })

  it('should title the section From the Vault when a playlist video is suggested', async () => {
    server.use(http.get('/api/streams/live', () => HttpResponse.json([])))

    renderWithProviders(<Home />)

    expect(await screen.findByRole('heading', { name: 'From the Vault' })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Live Now' })).not.toBeInTheDocument()
  })

  it('should title the section Live Now when someone is streaming', async () => {
    renderWithProviders(<Home />)

    expect(await screen.findByText('Thundermane is live')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Live Now' })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'From the Vault' })).not.toBeInTheDocument()
  })

  it('should title the section Live Now when no one is live and no video is available', async () => {
    server.use(
      http.get('/api/streams/live', () => HttpResponse.json([])),
      http.get('/api/streams/suggested-video', () => new HttpResponse(null, { status: 204 })),
    )

    renderWithProviders(<Home />)

    expect(await screen.findByText('Nobody is streaming right now.')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Live Now' })).toBeInTheDocument()
  })

  it('should link the suggested video to YouTube when no one is live', async () => {
    server.use(http.get('/api/streams/live', () => HttpResponse.json([])))

    renderWithProviders(<Home />)

    expect(await screen.findByRole('link', { name: 'Watch on YouTube →' })).toHaveAttribute(
      'href',
      'https://www.youtube.com/watch?v=vid-1',
    )
  })

  it('should not suggest a video when someone is live', async () => {
    let suggestedVideoRequested = false
    server.use(
      http.get('/api/streams/suggested-video', () => {
        suggestedVideoRequested = true
        return HttpResponse.json({ id: 'vid-1', title: 'Queen Ansurek kill' })
      }),
    )

    renderWithProviders(<Home />)

    expect(await screen.findByText('Thundermane is live')).toBeInTheDocument()
    expect(screen.queryByTitle('Queen Ansurek kill')).not.toBeInTheDocument()
    expect(suggestedVideoRequested).toBe(false)
  })

  it('should show a nobody-streaming message when no one is live and the playlist has no videos', async () => {
    server.use(
      http.get('/api/streams/live', () => HttpResponse.json([])),
      http.get('/api/streams/suggested-video', () => new HttpResponse(null, { status: 204 })),
    )

    renderWithProviders(<Home />)

    expect(await screen.findByText('Nobody is streaming right now.')).toBeInTheDocument()
  })

  it('should show a nobody-streaming message when no one is live and the suggested video request fails', async () => {
    server.use(
      http.get('/api/streams/live', () => HttpResponse.json([])),
      http.get('/api/streams/suggested-video', () => new HttpResponse(null, { status: 502 })),
    )

    renderWithProviders(<Home />)

    expect(await screen.findByText('Nobody is streaming right now.')).toBeInTheDocument()
  })

  it('should show empty-news copy when the guild has posted no news', async () => {
    server.use(http.get('/api/news', () => HttpResponse.json(newsPage([]))))

    renderWithProviders(<Home />)

    expect(await screen.findByText('No news posted yet.')).toBeInTheDocument()
    await waitFor(() =>
      expect(screen.queryByText('Loading the latest news…')).not.toBeInTheDocument(),
    )
  })
})
