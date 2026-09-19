import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '../../mocks/server'
import { renderWithProviders } from '../../testUtils'
import type { AuthUser } from '../../types/auth'
import { AppHeader } from './AppHeader'

const thundermane: AuthUser = {
  id: '11111111-1111-1111-1111-111111111111',
  username: 'thundermane',
  avatarUrl: 'https://cdn.discordapp.com/avatars/80351110224678912/abc.png',
}

function loggedInAs(user: AuthUser) {
  server.use(http.get('/api/auth/me', () => HttpResponse.json(user)))
}

describe('AppHeader', () => {
  it('should render a link for every navigation item', () => {
    renderWithProviders(<AppHeader />)

    expect(screen.getByRole('link', { name: 'Home' })).toHaveAttribute('href', '/')
    expect(screen.getByRole('link', { name: 'Roster' })).toHaveAttribute('href', '/roster')
    expect(screen.getByRole('link', { name: 'Calendar & Events' })).toHaveAttribute(
      'href',
      '/calendar',
    )
  })

  it('should mark the roster link as current when the roster route is active', () => {
    renderWithProviders(<AppHeader />, '/roster')

    expect(screen.getByRole('link', { name: 'Roster' })).toHaveAttribute('aria-current', 'page')
    expect(screen.getByRole('link', { name: 'Home' })).not.toHaveAttribute('aria-current')
  })

  it('should show a Discord login link when nobody is logged in', async () => {
    renderWithProviders(<AppHeader />)

    // vitest.config pins VITE_API_BASE_URL to http://localhost:3000/api; in the browser this is /api/auth/login
    expect(await screen.findByRole('link', { name: 'Log in with Discord' })).toHaveAttribute(
      'href',
      'http://localhost:3000/api/auth/login',
    )
    expect(screen.queryByRole('button', { name: 'Log out' })).not.toBeInTheDocument()
  })

  it('should show the username when a member is logged in', async () => {
    loggedInAs(thundermane)

    renderWithProviders(<AppHeader />)

    expect(await screen.findByText('thundermane')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Log out' })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Log in with Discord' })).not.toBeInTheDocument()
  })

  it('should show the avatar as decorative when the logged-in member has one', async () => {
    loggedInAs(thundermane)

    renderWithProviders(<AppHeader />)

    await screen.findByText('thundermane')
    expect(screen.getByRole('presentation')).toHaveAttribute('src', thundermane.avatarUrl)
  })

  it('should call the logout endpoint when Log out is clicked', async () => {
    loggedInAs(thundermane)
    let logoutCalls = 0
    server.use(
      http.post('/api/auth/logout', () => {
        logoutCalls += 1
        server.use(
          http.get('/api/auth/me', () =>
            HttpResponse.json({ error: 'login required' }, { status: 401 }),
          ),
        )
        return new HttpResponse(null, { status: 204 })
      }),
    )
    renderWithProviders(<AppHeader />)
    const logoutButton = await screen.findByRole('button', { name: 'Log out' })

    await userEvent.click(logoutButton)

    await waitFor(() => expect(logoutCalls).toBe(1))
    expect(await screen.findByRole('link', { name: 'Log in with Discord' })).toBeInTheDocument()
    expect(screen.queryByText('thundermane')).not.toBeInTheDocument()
  })
})
