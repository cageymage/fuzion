import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { officerRankRequests } from '../../mocks/handlers'
import { server } from '../../mocks/server'
import { renderWithProviders } from '../../testUtils'
import { OfficerRankRequests } from './OfficerRankRequests'

function shortDate(isoDate: string): string {
  return new Date(isoDate).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
}

function definitions(container: HTMLElement): (string | null)[] {
  return within(container)
    .getAllByRole('definition')
    .map((item) => item.textContent)
}

describe('OfficerRankRequests', () => {
  it('should show each open request with its character, direction, member, date and note when an officer opens the page', async () => {
    renderWithProviders(<OfficerRankRequests />, '/officer/rank-requests')

    const open = await screen.findByRole('list', { name: 'Open rank requests' })
    const request = within(open).getByRole('listitem', { name: 'Rank request for Thundermane Stormcaller' })
    expect(definitions(request)).toEqual([
      'Thundermane Stormcaller (Warrior)',
      'Main to alt rank',
      'Thundermane',
      shortDate('2026-10-20T12:00:00Z'),
      'Levelling a new main',
    ])
  })

  it('should show who resolved each past request and their note under Recently resolved', async () => {
    renderWithProviders(<OfficerRankRequests />, '/officer/rank-requests')

    const resolved = await screen.findByRole('list', { name: 'Resolved rank requests' })
    expect(definitions(resolved)).toEqual([
      'Sylvara Moonwhisper (Priest)',
      'Alt to main rank',
      'Sylvara',
      shortDate('2026-10-01T12:00:00Z'),
      'None',
    ])
    expect(
      within(resolved).getByText(`Declined by OfficerBob on ${shortDate('2026-10-02T12:00:00Z')}: Raid needs your main`),
    ).toBeInTheDocument()
  })

  it('should name the roster sync when it closed a request', async () => {
    server.use(
      http.get('/api/rank-requests', () =>
        HttpResponse.json([
          {
            ...officerRankRequests[0],
            status: 'done',
            resolvedAt: '2026-10-21T12:00:00Z',
            resolvedBy: null,
          },
        ]),
      ),
    )
    renderWithProviders(<OfficerRankRequests />, '/officer/rank-requests')

    expect(
      await screen.findByText(`Done by the roster sync on ${shortDate('2026-10-21T12:00:00Z')}`),
    ).toBeInTheDocument()
    expect(screen.getByText('No open rank requests.')).toBeInTheDocument()
  })

  it('should send the decision with the note when the officer declines a request', async () => {
    const user = userEvent.setup()
    let requests = officerRankRequests
    let patchBody: unknown
    server.use(
      http.get('/api/rank-requests', () => HttpResponse.json(requests)),
      http.patch('/api/rank-requests/rank-request-1', async ({ request }) => {
        patchBody = await request.json()
        requests = [
          { ...officerRankRequests[0], status: 'declined', resolutionNote: 'We need your tank', resolvedAt: '2026-10-22T12:00:00Z', resolvedBy: 'OfficerBob' },
          officerRankRequests[1],
        ]
        return HttpResponse.json(requests[0])
      }),
    )
    renderWithProviders(<OfficerRankRequests />, '/officer/rank-requests')

    const request = await screen.findByRole('listitem', { name: 'Rank request for Thundermane Stormcaller' })
    await user.type(within(request).getByRole('textbox', { name: 'Note to the member (optional)' }), ' We need your tank ')
    await user.click(within(request).getByRole('button', { name: 'Decline' }))

    expect(await screen.findByRole('status')).toHaveTextContent('The request for Thundermane Stormcaller was declined.')
    expect(patchBody).toEqual({ status: 'declined', note: 'We need your tank' })
    expect(await screen.findByText('No open rank requests.')).toBeInTheDocument()
  })

  it('should send done without a note when the officer marks a request done', async () => {
    const user = userEvent.setup()
    let patchBody: unknown
    server.use(
      http.patch('/api/rank-requests/rank-request-1', async ({ request }) => {
        patchBody = await request.json()
        return HttpResponse.json({ ...officerRankRequests[0], status: 'done' })
      }),
    )
    renderWithProviders(<OfficerRankRequests />, '/officer/rank-requests')

    const request = await screen.findByRole('listitem', { name: 'Rank request for Thundermane Stormcaller' })
    await user.click(within(request).getByRole('button', { name: 'Done' }))

    expect(await screen.findByRole('status')).toHaveTextContent('The request for Thundermane Stormcaller was marked done.')
    expect(patchBody).toEqual({ status: 'done', note: '' })
  })

  it("should show the server's message when resolving a request fails", async () => {
    const user = userEvent.setup()
    server.use(
      http.patch('/api/rank-requests/rank-request-1', () =>
        HttpResponse.json({ error: 'rank request is no longer open' }, { status: 409 }),
      ),
    )
    renderWithProviders(<OfficerRankRequests />, '/officer/rank-requests')

    const request = await screen.findByRole('listitem', { name: 'Rank request for Thundermane Stormcaller' })
    await user.click(within(request).getByRole('button', { name: 'Done' }))

    expect(await within(request).findByRole('alert')).toHaveTextContent('rank request is no longer open')
  })

  it('should say there are no requests when nothing has been asked yet', async () => {
    server.use(http.get('/api/rank-requests', () => HttpResponse.json([])))
    renderWithProviders(<OfficerRankRequests />, '/officer/rank-requests')

    expect(await screen.findByText('No open rank requests.')).toBeInTheDocument()
    expect(screen.getByText('Nothing resolved yet.')).toBeInTheDocument()
  })

  it('should say only officers can handle rank requests when a member opens the page', async () => {
    server.use(http.get('/api/rank-requests', () => new HttpResponse(null, { status: 403 })))
    renderWithProviders(<OfficerRankRequests />, '/officer/rank-requests')

    expect(await screen.findByText('Only officers can handle rank requests.')).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Log in with Discord' })).not.toBeInTheDocument()
  })

  it('should offer a Discord login when a visitor who is not logged in opens the page', async () => {
    server.use(http.get('/api/rank-requests', () => new HttpResponse(null, { status: 401 })))
    renderWithProviders(<OfficerRankRequests />, '/officer/rank-requests')

    expect(await screen.findByText('Only officers can handle rank requests.')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Log in with Discord' })).toBeInTheDocument()
  })

  it('should say the requests could not be loaded when the server fails', async () => {
    server.use(http.get('/api/rank-requests', () => new HttpResponse(null, { status: 500 })))
    renderWithProviders(<OfficerRankRequests />, '/officer/rank-requests')

    // The page retries once before giving up, so this waits past the retry delay.
    expect(await screen.findByText('Rank requests could not be loaded.', {}, { timeout: 3000 })).toBeInTheDocument()
  })
})
