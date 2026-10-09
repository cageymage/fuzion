import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { acceptedApplication, pendingApplication } from '../../mocks/handlers'
import { server } from '../../mocks/server'
import { renderWithProviders } from '../../testUtils'
import { OfficerApplications } from './OfficerApplications'

describe('OfficerApplications', () => {
  it('should list pending applications by default when an officer opens the page', async () => {
    renderWithProviders(<OfficerApplications />, '/officer/applications')

    const row = await screen.findByRole('button', { name: /Mira/ })
    expect(row).toHaveTextContent('Thornleaf')
    expect(row).toHaveTextContent('Druid · Healer')
    expect(row).toHaveTextContent('Pending')
    expect(screen.queryByRole('button', { name: /Brek/ })).not.toBeInTheDocument()
  })

  it('should request the selected status when the officer changes the status filter', async () => {
    const user = userEvent.setup()
    const requestedStatuses: (string | null)[] = []
    server.use(
      http.get('/api/applications', ({ request }) => {
        requestedStatuses.push(new URL(request.url).searchParams.get('status'))
        return HttpResponse.json([acceptedApplication])
      }),
    )
    renderWithProviders(<OfficerApplications />, '/officer/applications')
    await screen.findByRole('button', { name: /Brek/ })

    await user.selectOptions(screen.getByLabelText('Status'), 'Accepted')

    await screen.findByRole('button', { name: /Brek/ })
    expect(requestedStatuses).toEqual(['pending', 'accepted'])
  })

  it('should request every application without a status when the officer picks the all filter', async () => {
    const user = userEvent.setup()
    const requestedStatuses: (string | null)[] = []
    server.use(
      http.get('/api/applications', ({ request }) => {
        requestedStatuses.push(new URL(request.url).searchParams.get('status'))
        return HttpResponse.json([pendingApplication, acceptedApplication])
      }),
    )
    renderWithProviders(<OfficerApplications />, '/officer/applications')
    await screen.findByRole('button', { name: /Mira/ })

    await user.selectOptions(screen.getByLabelText('Status'), 'All')

    expect(await screen.findByRole('button', { name: /Brek/ })).toBeInTheDocument()
    expect(requestedStatuses).toEqual(['pending', null])
  })

  it('should show the full application including notes when the officer opens a row', async () => {
    const user = userEvent.setup()
    renderWithProviders(<OfficerApplications />, '/officer/applications')

    await user.click(await screen.findByRole('button', { name: /Mira/ }))

    expect(screen.getByText('Tue/Thu 8-11pm ET')).toBeInTheDocument()
    expect(screen.getByText('mira.heals')).toBeInTheDocument()
    expect(screen.getByText(/Raided Karazhan on my old realm\./)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Accept' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Decline' })).toBeInTheDocument()
  })

  it('should show the earlier review note when the officer opens an already reviewed application', async () => {
    const user = userEvent.setup()
    server.use(http.get('/api/applications', () => HttpResponse.json([acceptedApplication])))
    renderWithProviders(<OfficerApplications />, '/officer/applications')

    await user.click(await screen.findByRole('button', { name: /Brek/ }))

    expect(screen.getByRole('textbox', { name: 'Review note (optional)' })).toHaveValue('Great fit for tank spot')
    expect(screen.getByRole('button', { name: 'Accept' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Decline' })).toBeInTheDocument()
  })

  it('should send the accepted status and review note when the officer accepts an application', async () => {
    const user = userEvent.setup()
    let requestBody: unknown
    server.use(
      http.patch('/api/applications/:id', async ({ request }) => {
        requestBody = await request.json()
        return HttpResponse.json({ ...pendingApplication, status: 'accepted' })
      }),
    )
    renderWithProviders(<OfficerApplications />, '/officer/applications')
    await user.click(await screen.findByRole('button', { name: /Mira/ }))

    await user.type(screen.getByRole('textbox', { name: 'Review note (optional)' }), 'Welcome aboard')
    await user.click(screen.getByRole('button', { name: 'Accept' }))

    expect(await screen.findByText('Thornleaf was accepted.')).toBeInTheDocument()
    expect(requestBody).toEqual({ status: 'accepted', reviewNote: 'Welcome aboard' })
  })

  it('should show the declined status in the list when the officer declines an application', async () => {
    const user = userEvent.setup()
    let declined = false
    server.use(
      http.get('/api/applications', () =>
        HttpResponse.json([declined ? { ...pendingApplication, status: 'declined' } : pendingApplication]),
      ),
      http.patch('/api/applications/:id', () => {
        declined = true
        return HttpResponse.json({ ...pendingApplication, status: 'declined' })
      }),
    )
    renderWithProviders(<OfficerApplications />, '/officer/applications')
    await user.click(await screen.findByRole('button', { name: /Mira/ }))

    await user.click(screen.getByRole('button', { name: 'Decline' }))

    const row = await screen.findByRole('button', { name: /Mira.*Declined/ })
    expect(within(row).getByText('Declined')).toBeInTheDocument()
  })

  it('should show the server message when the review is rejected', async () => {
    const user = userEvent.setup()
    server.use(
      http.patch('/api/applications/:id', () =>
        HttpResponse.json({ error: 'reviewNote: must be at most 2000 characters' }, { status: 400 }),
      ),
    )
    renderWithProviders(<OfficerApplications />, '/officer/applications')
    await user.click(await screen.findByRole('button', { name: /Mira/ }))

    await user.click(screen.getByRole('button', { name: 'Accept' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('reviewNote: must be at most 2000 characters')
  })

  it('should show a not-authorized message when the API returns 403', async () => {
    server.use(http.get('/api/applications', () => HttpResponse.json({ error: 'officer only' }, { status: 403 })))

    renderWithProviders(<OfficerApplications />, '/officer/applications')

    expect(await screen.findByText('Only officers can review applications.')).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Log in with Discord' })).not.toBeInTheDocument()
  })

  it('should show a not-authorized message with a login link when the API returns 401', async () => {
    server.use(http.get('/api/applications', () => HttpResponse.json({ error: 'login required' }, { status: 401 })))

    renderWithProviders(<OfficerApplications />, '/officer/applications')

    expect(await screen.findByText('Only officers can review applications.')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Log in with Discord' })).toHaveAttribute(
      'href',
      'http://localhost:3000/api/auth/login',
    )
  })

  it('should show an empty message when no applications match the filter', async () => {
    server.use(http.get('/api/applications', () => HttpResponse.json([])))

    renderWithProviders(<OfficerApplications />, '/officer/applications')

    expect(await screen.findByText('No pending applications.')).toBeInTheDocument()
  })

  it('should show a loading message while the applications are loading', () => {
    renderWithProviders(<OfficerApplications />, '/officer/applications')

    expect(screen.getByText('Loading applications…')).toBeInTheDocument()
  })

  it('should show an error message when the applications cannot be loaded', async () => {
    server.use(http.get('/api/applications', () => HttpResponse.json({}, { status: 500 })))

    renderWithProviders(<OfficerApplications />, '/officer/applications')

    expect(await screen.findByText('Applications could not be loaded.', {}, { timeout: 3000 })).toBeInTheDocument()
  })
})
