import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '../../mocks/server'
import { renderWithProviders } from '../../testUtils'
import type { AuthUser } from '../../types/auth'
import type { Character } from '../../types/roster'
import { Dashboard } from './Dashboard'

const unlinkedMember: AuthUser = {
  id: 'user-1',
  username: 'thundermane',
  avatarUrl: null,
  isOfficer: false,
  isGuildMember: true,
  battlenetLinked: false,
  battletag: null,
}

const linkedMember: AuthUser = {
  ...unlinkedMember,
  battlenetLinked: true,
  battletag: 'Thunder#1234',
}

function ownedCharacter(id: string, name: string, secondaryName: string, isMain: boolean): Character {
  return {
    id,
    name,
    secondaryName,
    realm: 'Emberreach',
    class: 'Warrior',
    spec: null,
    role: 'tank',
    spec2: null,
    role2: null,
    isMain,
    raidTeam: null,
    race: null,
    level: 60,
    faction: null,
    createdAt: '2026-10-01T12:00:00Z',
    professions: [],
  }
}

async function characterRows() {
  const section = await screen.findByRole('region', { name: 'Your characters' })
  const list = await within(section).findByRole('list')
  return within(list)
    .getAllByRole('listitem')
    .map((item) => item.textContent)
}

describe('Dashboard', () => {
  it('should prompt to log in when the visitor is logged out', async () => {
    renderWithProviders(<Dashboard />, '/dashboard')

    expect(await screen.findByRole('link', { name: 'Log in with Discord' })).toHaveAttribute(
      'href',
      'http://localhost:3000/api/auth/login?returnTo=%2Fdashboard',
    )
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

  it("should list the user's characters after linking", async () => {
    server.use(http.get('/api/auth/me', () => HttpResponse.json(linkedMember)))

    renderWithProviders(<Dashboard />, '/dashboard')

    expect(await characterRows()).toEqual([
      'Thundermane Stormcaller · WarriorMain',
      'Thundermend Lightward · PriestMake main',
    ])
  })

  it('should not show a characters section when Battle.net is not linked', async () => {
    server.use(http.get('/api/auth/me', () => HttpResponse.json(unlinkedMember)))

    renderWithProviders(<Dashboard />, '/dashboard')

    expect(await screen.findByRole('link', { name: 'Link Battle.net' })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Your characters' })).not.toBeInTheDocument()
  })

  it('should mark a character as main when its Make main button is clicked', async () => {
    let mainId = 'char-a'
    let patchBody: unknown
    server.use(
      http.get('/api/auth/me', () => HttpResponse.json(linkedMember)),
      http.get('/api/me/characters', () =>
        HttpResponse.json([
          ownedCharacter('char-a', 'Ragefist', 'Ironjaw', mainId === 'char-a'),
          ownedCharacter('char-b', 'Shieldwall', 'Oakheart', mainId === 'char-b'),
        ]),
      ),
      http.patch('/api/me/characters/:id', async ({ params, request }) => {
        patchBody = await request.json()
        mainId = params.id as string
        return HttpResponse.json(ownedCharacter('char-b', 'Shieldwall', 'Oakheart', true))
      }),
    )
    renderWithProviders(<Dashboard />, '/dashboard')

    await userEvent.click(await screen.findByRole('button', { name: 'Make Shieldwall Oakheart your main' }))

    expect(await screen.findByRole('button', { name: 'Make Ragefist Ironjaw your main' })).toBeInTheDocument()
    expect(await characterRows()).toEqual([
      'Ragefist Ironjaw · WarriorMake main',
      'Shieldwall Oakheart · WarriorMain',
    ])
    expect(patchBody).toEqual({ isMain: true })
  })

  it("should explain how to claim characters when none of the linked account's characters are on the roster", async () => {
    server.use(
      http.get('/api/auth/me', () => HttpResponse.json(linkedMember)),
      http.get('/api/me/characters', () => HttpResponse.json([])),
    )

    renderWithProviders(<Dashboard />, '/dashboard')

    const section = await screen.findByRole('region', { name: 'Your characters' })
    expect(
      await within(section).findByText(/None of the characters on your Battle.net account are on the guild roster yet/),
    ).toBeInTheDocument()
    expect(within(section).getByRole('link', { name: 'link Battle.net again' })).toHaveAttribute(
      'href',
      'http://localhost:3000/api/auth/battlenet/link',
    )
  })

  it('should show an error when the characters cannot be loaded', async () => {
    server.use(
      http.get('/api/auth/me', () => HttpResponse.json(linkedMember)),
      http.get('/api/me/characters', () => new HttpResponse(null, { status: 500 })),
    )

    renderWithProviders(<Dashboard />, '/dashboard')

    expect(await screen.findByRole('alert')).toHaveTextContent('Could not load your characters')
  })

  it('should show an error when making a character main fails', async () => {
    server.use(
      http.get('/api/auth/me', () => HttpResponse.json(linkedMember)),
      http.patch('/api/me/characters/:id', () => new HttpResponse(null, { status: 500 })),
    )
    renderWithProviders(<Dashboard />, '/dashboard')

    await userEvent.click(await screen.findByRole('button', { name: 'Make Thundermend Lightward your main' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Could not change your main')
  })
})
