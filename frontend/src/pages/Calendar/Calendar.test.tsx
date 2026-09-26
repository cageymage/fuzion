import { screen, within } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '../../mocks/server'
import { renderWithProviders } from '../../testUtils'
import { Calendar } from './Calendar'

describe('Calendar', () => {
  it('should list upcoming raids soonest first', async () => {
    renderWithProviders(<Calendar />)

    const raids = await screen.findAllByRole('listitem')

    expect(raids).toHaveLength(3)
    expect(raids[0]).toHaveTextContent("Mythic · Nerub'ar Palace")
    expect(raids[0]).toHaveTextContent('8/8 Heroic cleared')
    expect(raids[1]).toHaveTextContent("Heroic · Nerub'ar Palace")
    expect(raids[2]).toHaveTextContent('Normal · Liberation of Undermine')
  })

  it('should group raids under a single date heading when they start on the same day', async () => {
    renderWithProviders(<Calendar />)

    const dayHeadings = await screen.findAllByRole('heading', { level: 2 })

    expect(dayHeadings).toHaveLength(2)
    const firstDay = dayHeadings[0].closest('section') as HTMLElement
    expect(within(firstDay).getAllByRole('listitem')).toHaveLength(2)
  })

  it('should show a loading message while the raids are loading', () => {
    renderWithProviders(<Calendar />)

    expect(screen.getByText('Loading the raid schedule…')).toBeInTheDocument()
  })

  it('should show an empty message when no raids are scheduled', async () => {
    server.use(http.get('/api/raids', () => HttpResponse.json([])))

    renderWithProviders(<Calendar />)

    expect(await screen.findByText('No raids scheduled.')).toBeInTheDocument()
  })

  it('should show an error message when the raids request fails', async () => {
    server.use(http.get('/api/raids', () => new HttpResponse(null, { status: 500 })))

    renderWithProviders(<Calendar />)

    expect(await screen.findByText('The raid schedule could not be loaded.')).toBeInTheDocument()
  })
})
