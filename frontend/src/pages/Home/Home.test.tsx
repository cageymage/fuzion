import { screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '../../mocks/server'
import { renderWithProviders } from '../../testUtils'
import { Home } from './Home'

describe('Home', () => {
  it('should show the guild name, realm and tagline', async () => {
    renderWithProviders(<Home />)

    expect(await screen.findByRole('heading', { name: 'Fuzion', level: 1 })).toBeInTheDocument()
    expect(screen.getByText('Emberreach · US')).toBeInTheDocument()
    expect(screen.getByText('Raiding · Mythic+ · PvP · Crafting')).toBeInTheDocument()
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

  it('should show a nobody-streaming message when no one is live', async () => {
    server.use(http.get('/api/streams/live', () => HttpResponse.json([])))

    renderWithProviders(<Home />)

    expect(await screen.findByText('Nobody is streaming right now.')).toBeInTheDocument()
  })

  it('should show empty-news copy when the guild has posted no news', async () => {
    server.use(http.get('/api/news', () => HttpResponse.json([])))

    renderWithProviders(<Home />)

    expect(await screen.findByText('No news posted yet.')).toBeInTheDocument()
    await waitFor(() =>
      expect(screen.queryByText('Loading the latest news…')).not.toBeInTheDocument(),
    )
  })
})
