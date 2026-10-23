import { screen } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { Route, Routes } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { server } from '../../mocks/server'
import { renderWithProviders } from '../../testUtils'
import { OfficerSurveys } from './OfficerSurveys'

const officer = { id: 'user-1', username: 'Officer', avatarUrl: null, isOfficer: true }
const member = { id: 'user-2', username: 'Member', avatarUrl: null, isOfficer: false }

function signInAs(user: typeof officer) {
  server.use(http.get('/api/auth/me', () => HttpResponse.json(user)))
}

function renderPage() {
  return renderWithProviders(
    <Routes>
      <Route path="/officer/surveys" element={<OfficerSurveys />} />
      <Route path="/" element={<p>Home page</p>} />
    </Routes>,
    '/officer/surveys',
  )
}

describe('OfficerSurveys', () => {
  it('should link each survey to its manage page and offer a new survey when the officer opens the page', async () => {
    signInAs(officer)
    server.use(
      http.get('/api/surveys', () =>
        HttpResponse.json([
          { id: 'survey-2', title: 'Spring roster', closesAt: '2026-03-21T20:00:00Z', status: 'open' },
          { id: 'survey-1', title: 'Winter plans', closesAt: '2026-03-13T20:00:00Z', status: 'closed' },
        ]),
      ),
    )

    renderPage()

    const open = (await screen.findByRole('link', { name: 'Spring roster' })).closest('li')
    expect(screen.getByRole('link', { name: 'Spring roster' })).toHaveAttribute('href', '/officer/surveys/survey-2')
    expect(open).toHaveTextContent('Open')
    const closed = screen.getByRole('link', { name: 'Winter plans' }).closest('li')
    expect(screen.getByRole('link', { name: 'Winter plans' })).toHaveAttribute('href', '/officer/surveys/survey-1')
    expect(closed).toHaveTextContent('Closed')
    expect(screen.getByRole('link', { name: 'New survey' })).toHaveAttribute('href', '/officer/surveys/new')
  })

  it('should say there are no surveys yet when the list is empty', async () => {
    signInAs(officer)
    server.use(http.get('/api/surveys', () => HttpResponse.json([])))

    renderPage()

    expect(await screen.findByText('No surveys yet.')).toBeInTheDocument()
  })

  it('should show an error when the surveys fail to load', async () => {
    signInAs(officer)
    server.use(http.get('/api/surveys', () => new HttpResponse(null, { status: 500 })))

    renderPage()

    expect(await screen.findByRole('alert')).toHaveTextContent('The surveys could not be loaded.')
  })

  it('should send the visitor to the home page when they are not an officer', async () => {
    signInAs(member)

    renderPage()

    expect(await screen.findByText('Home page')).toBeInTheDocument()
  })
})
