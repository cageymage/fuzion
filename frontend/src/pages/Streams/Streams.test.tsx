import { screen } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '../../mocks/server'
import { renderWithProviders } from '../../testUtils'
import { Streams } from './Streams'

describe('Streams', () => {
  it('should show a LIVE badge only on channels that are currently live', async () => {
    renderWithProviders(<Streams />)
    await screen.findByRole('link', { name: /Moonveil/ })

    expect(screen.getAllByText('LIVE')).toHaveLength(1)
    expect(screen.getByRole('link', { name: /Thundermane/ })).toHaveTextContent('LIVE')
    expect(screen.getByRole('link', { name: /Moonveil/ })).not.toHaveTextContent('LIVE')
  })

  it('should show the viewer count only on channels that are currently live', async () => {
    renderWithProviders(<Streams />)
    await screen.findByRole('link', { name: /Moonveil/ })

    expect(screen.getByRole('link', { name: /Thundermane/ })).toHaveTextContent('1.2K')
    expect(screen.getByRole('link', { name: /Moonveil/ })).not.toHaveTextContent('0')
  })

  it('should link each card to the channel URL in a new tab', async () => {
    renderWithProviders(<Streams />)

    const thundermane = await screen.findByRole('link', {
      name: /Thundermane/,
    })
    const moonveil = screen.getByRole('link', { name: /Moonveil/ })
    expect(thundermane).toHaveAttribute('href', 'https://twitch.tv/thundermane')
    expect(thundermane).toHaveAttribute('target', '_blank')
    expect(moonveil).toHaveAttribute('href', 'https://twitch.tv/moonveil')
    expect(moonveil).toHaveAttribute('target', '_blank')
  })

  it('should list channels in the order the API returns them', async () => {
    renderWithProviders(<Streams />)

    const links = await screen.findAllByRole('link')
    expect(links.map((link) => link.getAttribute('href'))).toEqual([
      'https://twitch.tv/thundermane',
      'https://twitch.tv/moonveil',
    ])
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
