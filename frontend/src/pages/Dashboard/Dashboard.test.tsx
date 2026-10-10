import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '../../mocks/server'
import { renderWithProviders } from '../../testUtils'
import type { AuthUser } from '../../types/auth'
import { Dashboard } from './Dashboard'

const unlinkedMember: AuthUser = {
  id: 'user-1',
  username: 'thundermane',
  avatarUrl: null,
  isOfficer: false,
  battlenetLinked: false,
  battletag: null,
}

const linkedMember: AuthUser = {
  ...unlinkedMember,
  battlenetLinked: true,
  battletag: 'Thunder#1234',
}

describe('Dashboard', () => {
  it('should prompt to log in when the visitor is logged out', async () => {
    renderWithProviders(<Dashboard />, '/dashboard')

    expect(await screen.findByRole('link', { name: 'Log in with Discord' })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Link Battle.net' })).not.toBeInTheDocument()
  })

  it('should show a Link Battle.net button when the user has not linked yet', async () => {
    server.use(http.get('/api/auth/me', () => HttpResponse.json(unlinkedMember)))

    renderWithProviders(<Dashboard />, '/dashboard')

    expect(await screen.findByRole('link', { name: 'Link Battle.net' })).toHaveAttribute(
      'href',
      'http://localhost:3000/api/auth/battlenet/link',
    )
    expect(screen.queryByRole('button', { name: 'Unlink Battle.net' })).not.toBeInTheDocument()
  })

  it('should show the battletag and an Unlink button when Battle.net is linked', async () => {
    server.use(http.get('/api/auth/me', () => HttpResponse.json(linkedMember)))

    renderWithProviders(<Dashboard />, '/dashboard')

    expect(await screen.findByText('Thunder#1234')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Unlink Battle.net' })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Link Battle.net' })).not.toBeInTheDocument()
  })

  it('should show an already-linked message when the callback redirects with battlenet=already-linked', async () => {
    server.use(http.get('/api/auth/me', () => HttpResponse.json(unlinkedMember)))

    renderWithProviders(<Dashboard />, '/dashboard?battlenet=already-linked')

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'already linked to another member',
    )
  })

  it('should return to the Link button when Unlink Battle.net is clicked', async () => {
    let linked = true
    server.use(
      http.get('/api/auth/me', () => HttpResponse.json(linked ? linkedMember : unlinkedMember)),
      http.delete('/api/auth/battlenet', () => {
        linked = false
        return new HttpResponse(null, { status: 204 })
      }),
    )
    renderWithProviders(<Dashboard />, '/dashboard')

    await userEvent.click(await screen.findByRole('button', { name: 'Unlink Battle.net' }))

    expect(await screen.findByRole('link', { name: 'Link Battle.net' })).toBeInTheDocument()
  })

  it('should show an error when unlinking fails', async () => {
    server.use(
      http.get('/api/auth/me', () => HttpResponse.json(linkedMember)),
      http.delete('/api/auth/battlenet', () => new HttpResponse(null, { status: 500 })),
    )
    renderWithProviders(<Dashboard />, '/dashboard')

    await userEvent.click(await screen.findByRole('button', { name: 'Unlink Battle.net' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Could not unlink Battle.net')
  })
})
