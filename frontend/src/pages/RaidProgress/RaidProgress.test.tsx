import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { formatRaidDay } from '../../lib/format'
import { server } from '../../mocks/server'
import { renderWithProviders } from '../../testUtils'
import { RaidProgress } from './RaidProgress'

describe('RaidProgress', () => {
  it('should show the kill count and progress bar for the current tier', async () => {
    // no raid history in this test, so the only progress bar on the page is the current tier's
    server.use(http.get('/api/raid-tiers', () => HttpResponse.json([])))

    renderWithProviders(<RaidProgress />)

    expect(await screen.findByRole('heading', { name: 'Molten Depths' })).toBeInTheDocument()
    expect(screen.getByText('1 / 3 bosses defeated')).toBeInTheDocument()

    const bar = screen.getByRole('progressbar')
    expect(bar).toHaveAttribute('value', '1')
    expect(bar).toHaveAttribute('max', '3')
  })

  it('should list bosses in order with kill dates for killed ones', async () => {
    const killedAt = '2026-03-01T20:00:00Z'
    server.use(
      http.get('/api/raid-progress', () =>
        HttpResponse.json([
          {
            tier: { name: 'Molten Depths' },
            bosses: [
              { id: 'boss-1', name: 'Grimjaw', killedAt },
              { id: 'boss-2', name: 'Ashveil', killedAt: null },
              { id: 'boss-3', name: 'Pyrelord', killedAt: null },
            ],
            killed: 1,
            total: 3,
          },
        ]),
      ),
      // no raid history in this test, so the only list items on the page are these bosses
      http.get('/api/raid-tiers', () => HttpResponse.json([])),
    )

    renderWithProviders(<RaidProgress />)

    const bosses = await screen.findAllByRole('listitem')
    expect(bosses).toHaveLength(3)
    expect(bosses[0]).toHaveTextContent('Grimjaw')
    expect(bosses[0]).toHaveTextContent(formatRaidDay(killedAt))
    expect(bosses[1]).toHaveTextContent('Ashveil')
    expect(bosses[1]).not.toHaveTextContent(formatRaidDay(killedAt))
    expect(bosses[2]).toHaveTextContent('Pyrelord')
  })

  it('should show a card for every raid that is current at once', async () => {
    server.use(
      http.get('/api/raid-progress', () =>
        HttpResponse.json([
          { tier: { name: "Onyxia's Lair" }, bosses: [{ id: 'boss-onyxia', name: 'Onyxia', killedAt: null }], killed: 0, total: 1 },
          { tier: { name: 'Barrow Deeps' }, bosses: [], killed: 0, total: 0 },
        ]),
      ),
    )

    renderWithProviders(<RaidProgress />)

    expect(await screen.findByRole('heading', { name: "Onyxia's Lair" })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Barrow Deeps' })).toBeInTheDocument()
    expect(screen.getByText('0 / 1 bosses defeated')).toBeInTheDocument()
    expect(screen.getByText('0 / 0 bosses defeated')).toBeInTheDocument()
  })

  it('should show a not-set-up message when there is no current tier', async () => {
    server.use(
      http.get('/api/raid-progress', () =>
        HttpResponse.json({ error: 'no current raid tier' }, { status: 404 }),
      ),
    )

    renderWithProviders(<RaidProgress />)

    expect(await screen.findByText('No tier set up yet.')).toBeInTheDocument()
  })

  it('should collapse raid history by default', async () => {
    renderWithProviders(<RaidProgress />)

    const summary = await screen.findByText('Raid History')
    expect(summary.closest('details')).not.toHaveAttribute('open')
  })

  it('should list past tiers as raid history after it is expanded', async () => {
    renderWithProviders(<RaidProgress />)

    await userEvent.click(await screen.findByText('Raid History'))

    expect(screen.getByRole('heading', { name: 'Shattered Spire' })).toBeInTheDocument()
    expect(screen.getByText('2 / 2 bosses defeated')).toBeInTheDocument()
    expect(screen.getByText('Voidshard Sentinel')).toBeInTheDocument()
    expect(screen.getByText('Thornqueen Ilyra')).toBeInTheDocument()
  })

  it('should not show a raid history section when there are no past tiers', async () => {
    server.use(http.get('/api/raid-tiers', () => HttpResponse.json([])))

    renderWithProviders(<RaidProgress />)

    await screen.findByRole('heading', { name: 'Molten Depths' })
    expect(screen.queryByText('Raid History')).not.toBeInTheDocument()
  })
})
