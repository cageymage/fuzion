import { screen } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '../../mocks/server'
import { renderWithProviders } from '../../testUtils'
import { Surveys } from './Surveys'

describe('Surveys', () => {
  it('should link each survey to its page with an open or closed label when surveys exist', async () => {
    server.use(
      http.get('/api/surveys', () =>
        HttpResponse.json([
          { id: 'survey-2', title: 'Spring roster', closesAt: '2026-03-21T20:00:00Z', status: 'open' },
          { id: 'survey-1', title: 'Winter plans', closesAt: '2026-03-13T20:00:00Z', status: 'closed' },
        ]),
      ),
    )

    renderWithProviders(<Surveys />, '/surveys')

    const open = (await screen.findByRole('link', { name: 'Spring roster' })).closest('li')
    expect(screen.getByRole('link', { name: 'Spring roster' })).toHaveAttribute('href', '/surveys/survey-2')
    expect(open).toHaveTextContent('Open')
    expect(open).toHaveTextContent('Closes')
    const closed = screen.getByRole('link', { name: 'Winter plans' }).closest('li')
    expect(screen.getByRole('link', { name: 'Winter plans' })).toHaveAttribute('href', '/surveys/survey-1')
    expect(closed).toHaveTextContent('Closed')
  })

  it('should say there are no surveys yet when the list is empty', async () => {
    server.use(http.get('/api/surveys', () => HttpResponse.json([])))

    renderWithProviders(<Surveys />, '/surveys')

    expect(await screen.findByText('No surveys yet.')).toBeInTheDocument()
  })

  it('should show an error when the surveys fail to load', async () => {
    server.use(http.get('/api/surveys', () => new HttpResponse(null, { status: 500 })))

    renderWithProviders(<Surveys />, '/surveys')

    expect(await screen.findByRole('alert')).toHaveTextContent('The surveys could not be loaded.')
  })
})
