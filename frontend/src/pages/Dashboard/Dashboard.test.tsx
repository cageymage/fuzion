import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '../../mocks/server'
import { renderWithProviders } from '../../testUtils'
import type { AuthUser } from '../../types/auth'
import type { RankRequest } from '../../types/rankRequests'
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

const officer: AuthUser = { ...linkedMember, isOfficer: true }

function ownedCharacter(id: string, name: string, secondaryName: string, isMain: boolean): Character {
  return {
    id,
    name,
    secondaryName,
    realm: 'Emberreach',
    class: 'Warrior',
    spec: null,
    role: null,
    spec2: null,
    role2: null,
    isMain,
    rank: null,
    raidTeam: null,
    race: null,
    level: 60,
    faction: null,
    gender: null,
    createdAt: '2026-10-01T12:00:00Z',
    professions: [],
  }
}

function rankRequest(overrides: Partial<RankRequest>): RankRequest {
  return {
    id: 'request-1',
    characterId: 'char-a',
    characterName: 'Ragefist',
    characterSecondaryName: 'Ironjaw',
    characterClass: 'Warrior',
    direction: 'to_alt',
    status: 'open',
    note: '',
    resolutionNote: '',
    requestedBy: 'thundermane',
    createdAt: '2026-10-20T12:00:00Z',
    resolvedAt: null,
    resolvedBy: null,
    ...overrides,
  }
}

// Each character's own line (name, class, guild rank), without its specs or requests.
async function characterRows() {
  const list = await screen.findByRole('list', { name: 'Characters' })
  return within(list)
    .getAllByTestId('character-header')
    .map((header) => header.textContent)
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

  it('should link to the submission page when the user is a guild member', async () => {
    server.use(http.get('/api/auth/me', () => HttpResponse.json(unlinkedMember)))

    renderWithProviders(<Dashboard />, '/dashboard')

    expect(await screen.findByRole('link', { name: 'Submit art or screenshots' })).toHaveAttribute(
      'href',
      '/submissions',
    )
  })

  it('should not link to the submission page when the user is not a guild member', async () => {
    server.use(http.get('/api/auth/me', () => HttpResponse.json({ ...unlinkedMember, isGuildMember: false })))

    renderWithProviders(<Dashboard />, '/dashboard')

    await screen.findByRole('link', { name: 'Link Battle.net' })
    expect(screen.queryByRole('link', { name: 'Submit art or screenshots' })).not.toBeInTheDocument()
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

  it('should say the characters were refreshed when the callback redirects with battlenet=linked', async () => {
    server.use(http.get('/api/auth/me', () => HttpResponse.json(linkedMember)))

    renderWithProviders(<Dashboard />, '/dashboard?battlenet=linked')

    expect(await screen.findByRole('status')).toHaveTextContent(
      'Battle.net is linked and your characters are refreshed.',
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
      'Thundermane Stormcaller · WarriorOfficer',
      'Thundermend Lightward · PriestAlt',
    ])
  })

  it("should not offer a main or alt toggle on the user's characters", async () => {
    server.use(http.get('/api/auth/me', () => HttpResponse.json(linkedMember)))

    renderWithProviders(<Dashboard />, '/dashboard')

    const section = await screen.findByRole('region', { name: 'Your characters' })
    await within(section).findByText('Thundermane Stormcaller')
    expect(within(section).queryByRole('button', { name: /your main|an alt/ })).not.toBeInTheDocument()
  })

  it('should explain that main or alt follows the in-game guild rank', async () => {
    server.use(http.get('/api/auth/me', () => HttpResponse.json(linkedMember)))

    renderWithProviders(<Dashboard />, '/dashboard')

    const section = await screen.findByRole('region', { name: 'Your characters' })
    expect(await within(section).findByText(/Main or alt follows your guild rank in game/)).toBeInTheDocument()
  })

  it("should color each character's name and class with its class color", async () => {
    server.use(http.get('/api/auth/me', () => HttpResponse.json(linkedMember)))

    renderWithProviders(<Dashboard />, '/dashboard')

    const list = await screen.findByRole('region', { name: 'Your characters' })
    expect(await within(list).findByText('Thundermane Stormcaller')).toHaveStyle({ color: '#C79C6E' })
    expect(within(list).getByText('Warrior')).toHaveStyle({ color: '#C79C6E' })
    expect(within(list).getByText('Thundermend Lightward')).toHaveStyle({ color: '#FFFFFF' })
  })

  it('should say no characters were claimed when the callback redirects with battlenet=characters-unavailable', async () => {
    server.use(http.get('/api/auth/me', () => HttpResponse.json(linkedMember)))

    renderWithProviders(<Dashboard />, '/dashboard?battlenet=characters-unavailable')

    expect(await screen.findByRole('status')).toHaveTextContent(
      'Battle.net is linked, but its World of Warcraft characters could not be read, so none were claimed',
    )
  })

  it('should not show a characters section when Battle.net is not linked', async () => {
    server.use(http.get('/api/auth/me', () => HttpResponse.json(unlinkedMember)))

    renderWithProviders(<Dashboard />, '/dashboard')

    expect(await screen.findByRole('link', { name: 'Link Battle.net' })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Your characters' })).not.toBeInTheDocument()
  })

  it('should offer a Refresh characters button linking to Battle.net when characters are listed', async () => {
    server.use(http.get('/api/auth/me', () => HttpResponse.json(linkedMember)))

    renderWithProviders(<Dashboard />, '/dashboard')

    const section = await screen.findByRole('region', { name: 'Your characters' })
    expect(within(section).getByRole('link', { name: 'Refresh characters' })).toHaveAttribute(
      'href',
      'http://localhost:3000/api/auth/battlenet/link',
    )
    expect(
      within(section).getByText(
        'Refreshing picks up characters added to the guild roster since you linked Battle.net.',
      ),
    ).toBeInTheDocument()
  })

  it("should offer the Refresh characters button when none of the linked account's characters are on the roster", async () => {
    server.use(
      http.get('/api/auth/me', () => HttpResponse.json(linkedMember)),
      http.get('/api/me/characters', () => HttpResponse.json([])),
    )

    renderWithProviders(<Dashboard />, '/dashboard')

    const section = await screen.findByRole('region', { name: 'Your characters' })
    expect(
      await within(section).findByText(
        'None of the characters on your Battle.net account are on the guild roster yet. Once they are, refresh your characters to claim them.',
      ),
    ).toBeInTheDocument()
    expect(within(section).getByRole('link', { name: 'Refresh characters' })).toHaveAttribute(
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

  it("should show each character's professions with their skill levels without edit controls", async () => {
    server.use(http.get('/api/auth/me', () => HttpResponse.json(linkedMember)))

    renderWithProviders(<Dashboard />, '/dashboard')

    const section = await screen.findByRole('region', { name: 'Your characters' })
    expect(await within(section).findByText('Mining 300')).toBeInTheDocument()
    expect(within(section).queryByRole('spinbutton')).not.toBeInTheDocument()
    expect(within(section).queryByRole('button', { name: /^(Add|Remove)/ })).not.toBeInTheDocument()
  })

  it('should show the specs as not set when the owner has not picked any', async () => {
    server.use(
      http.get('/api/auth/me', () => HttpResponse.json(linkedMember)),
      http.get('/api/me/characters', () =>
        HttpResponse.json([ownedCharacter('char-a', 'Ragefist', 'Ironjaw', true)]),
      ),
    )

    renderWithProviders(<Dashboard />, '/dashboard')

    const section = await screen.findByRole('region', { name: 'Your characters' })
    await within(section).findByText('Ragefist Ironjaw')
    const specs = within(section)
      .getAllByRole('definition')
      .map((item) => item.textContent)
    expect(specs).toEqual(['Not set'])
  })

  it('should show the primary and secondary specs with their roles when both are set', async () => {
    server.use(
      http.get('/api/auth/me', () => HttpResponse.json(linkedMember)),
      http.get('/api/me/characters', () =>
        HttpResponse.json([
          {
            ...ownedCharacter('char-a', 'Ragefist', 'Ironjaw', true),
            spec: 'Protection',
            role: 'tank',
            spec2: 'Fury',
            role2: 'dps',
          },
        ]),
      ),
    )

    renderWithProviders(<Dashboard />, '/dashboard')

    const section = await screen.findByRole('region', { name: 'Your characters' })
    await within(section).findByText('Ragefist Ironjaw')
    const terms = within(section)
      .getAllByRole('term')
      .map((item) => item.textContent)
    const specs = within(section)
      .getAllByRole('definition')
      .map((item) => item.textContent)
    expect(terms).toEqual(['Primary spec', 'Secondary spec'])
    expect(specs).toEqual(['Protection (Tank)', 'Fury (DPS)'])
  })

  it('should save the primary spec and role when the specs form is submitted', async () => {
    let character = ownedCharacter('char-a', 'Ragefist', 'Ironjaw', true)
    let patchBody: unknown
    server.use(
      http.get('/api/auth/me', () => HttpResponse.json(linkedMember)),
      http.get('/api/me/characters', () => HttpResponse.json([character])),
      http.patch('/api/me/characters/char-a', async ({ request }) => {
        patchBody = await request.json()
        character = { ...character, spec: 'Protection', role: 'tank' }
        return HttpResponse.json(character)
      }),
    )
    renderWithProviders(<Dashboard />, '/dashboard')

    await userEvent.click(await screen.findByRole('button', { name: 'Edit specs for Ragefist Ironjaw' }))
    const form = screen.getByRole('form', { name: 'Specs for Ragefist Ironjaw' })
    await userEvent.selectOptions(within(form).getByRole('combobox', { name: 'Primary spec' }), 'Protection')
    await userEvent.selectOptions(within(form).getByRole('combobox', { name: 'Primary role' }), 'Tank')
    await userEvent.click(within(form).getByRole('button', { name: 'Save' }))

    expect(await screen.findByText('Protection (Tank)')).toBeInTheDocument()
    expect(patchBody).toEqual({ spec: 'Protection', role: 'tank' })
  })

  it('should send the secondary spec with a role when one is chosen', async () => {
    let patchBody: unknown
    server.use(
      http.get('/api/auth/me', () => HttpResponse.json(linkedMember)),
      http.get('/api/me/characters', () =>
        HttpResponse.json([
          { ...ownedCharacter('char-a', 'Ragefist', 'Ironjaw', true), spec: 'Protection', role: 'tank' },
        ]),
      ),
      http.patch('/api/me/characters/char-a', async ({ request }) => {
        patchBody = await request.json()
        return HttpResponse.json(ownedCharacter('char-a', 'Ragefist', 'Ironjaw', true))
      }),
    )
    renderWithProviders(<Dashboard />, '/dashboard')

    await userEvent.click(await screen.findByRole('button', { name: 'Edit specs for Ragefist Ironjaw' }))
    const form = screen.getByRole('form', { name: 'Specs for Ragefist Ironjaw' })
    await userEvent.selectOptions(within(form).getByRole('combobox', { name: 'Secondary spec' }), 'Fury')
    await userEvent.selectOptions(within(form).getByRole('combobox', { name: 'Secondary role' }), 'DPS')
    await userEvent.click(within(form).getByRole('button', { name: 'Save' }))

    await screen.findByRole('button', { name: 'Edit specs for Ragefist Ironjaw' })
    expect(patchBody).toEqual({ spec2: 'Fury', role2: 'dps' })
  })

  it('should send an empty role when the owner sets the primary role back to Not set', async () => {
    let character: Character = { ...ownedCharacter('char-a', 'Ragefist', 'Ironjaw', true), role: 'tank' }
    let patchBody: unknown
    server.use(
      http.get('/api/auth/me', () => HttpResponse.json(linkedMember)),
      http.get('/api/me/characters', () => HttpResponse.json([character])),
      http.patch('/api/me/characters/char-a', async ({ request }) => {
        patchBody = await request.json()
        character = { ...character, role: null }
        return HttpResponse.json(character)
      }),
    )
    renderWithProviders(<Dashboard />, '/dashboard')

    await userEvent.click(await screen.findByRole('button', { name: 'Edit specs for Ragefist Ironjaw' }))
    const form = screen.getByRole('form', { name: 'Specs for Ragefist Ironjaw' })
    await userEvent.selectOptions(within(form).getByRole('combobox', { name: 'Primary role' }), 'Not set')
    await userEvent.click(within(form).getByRole('button', { name: 'Save' }))

    expect(await screen.findByText('Not set')).toBeInTheDocument()
    expect(patchBody).toEqual({ role: '' })
  })

  it("should show the server's message when saving the specs fails", async () => {
    server.use(
      http.get('/api/auth/me', () => HttpResponse.json(linkedMember)),
      http.get('/api/me/characters', () =>
        HttpResponse.json([ownedCharacter('char-a', 'Ragefist', 'Ironjaw', true)]),
      ),
      http.patch('/api/me/characters/char-a', () =>
        HttpResponse.json({ error: 'spec: must be one of Arms, Fury, Protection' }, { status: 400 }),
      ),
    )
    renderWithProviders(<Dashboard />, '/dashboard')

    await userEvent.click(await screen.findByRole('button', { name: 'Edit specs for Ragefist Ironjaw' }))
    const form = screen.getByRole('form', { name: 'Specs for Ragefist Ironjaw' })
    await userEvent.selectOptions(within(form).getByRole('combobox', { name: 'Primary spec' }), 'Arms')
    await userEvent.click(within(form).getByRole('button', { name: 'Save' }))

    expect(await within(form).findByRole('alert')).toHaveTextContent(
      'spec: must be one of Arms, Fury, Protection',
    )
  })

  it('should offer an alt rank request when the character is a main', async () => {
    server.use(
      http.get('/api/auth/me', () => HttpResponse.json(linkedMember)),
      http.get('/api/me/characters', () =>
        HttpResponse.json([ownedCharacter('char-a', 'Ragefist', 'Ironjaw', true)]),
      ),
    )

    renderWithProviders(<Dashboard />, '/dashboard')

    expect(await screen.findByRole('button', { name: 'Request alt rank for Ragefist Ironjaw' })).toBeInTheDocument()
  })

  it('should offer a main rank request when the character is an alt', async () => {
    server.use(
      http.get('/api/auth/me', () => HttpResponse.json(linkedMember)),
      http.get('/api/me/characters', () =>
        HttpResponse.json([ownedCharacter('char-b', 'Shieldwall', 'Ironjaw', false)]),
      ),
    )

    renderWithProviders(<Dashboard />, '/dashboard')

    expect(
      await screen.findByRole('button', { name: 'Request main rank for Shieldwall Ironjaw' }),
    ).toBeInTheDocument()
  })

  it('should file a rank request with the note when Send request is clicked', async () => {
    let requests: RankRequest[] = []
    let postBody: unknown
    server.use(
      http.get('/api/auth/me', () => HttpResponse.json(linkedMember)),
      http.get('/api/me/characters', () =>
        HttpResponse.json([ownedCharacter('char-a', 'Ragefist', 'Ironjaw', true)]),
      ),
      http.get('/api/me/rank-requests', () => HttpResponse.json(requests)),
      http.post('/api/me/characters/char-a/rank-requests', async ({ request }) => {
        postBody = await request.json()
        requests = [rankRequest({ note: 'Levelling a new main' })]
        return HttpResponse.json(requests[0], { status: 201 })
      }),
    )
    renderWithProviders(<Dashboard />, '/dashboard')

    await userEvent.click(await screen.findByRole('button', { name: 'Request alt rank for Ragefist Ironjaw' }))
    const form = screen.getByRole('form', { name: 'Rank request for Ragefist Ironjaw' })
    await userEvent.type(
      within(form).getByRole('textbox', { name: 'Note for officers (optional)' }),
      'Levelling a new main',
    )
    await userEvent.click(within(form).getByRole('button', { name: 'Send request' }))

    expect(
      await screen.findByText('You asked for a move to an alt rank. An officer will change it in game.'),
    ).toBeInTheDocument()
    expect(postBody).toEqual({ note: 'Levelling a new main' })
  })

  it("should show the server's message when filing a rank request fails", async () => {
    server.use(
      http.get('/api/auth/me', () => HttpResponse.json(linkedMember)),
      http.get('/api/me/characters', () =>
        HttpResponse.json([ownedCharacter('char-a', 'Ragefist', 'Ironjaw', true)]),
      ),
      http.post('/api/me/characters/char-a/rank-requests', () =>
        HttpResponse.json({ error: 'this character already has an open rank request' }, { status: 409 }),
      ),
    )
    renderWithProviders(<Dashboard />, '/dashboard')

    await userEvent.click(await screen.findByRole('button', { name: 'Request alt rank for Ragefist Ironjaw' }))
    const form = screen.getByRole('form', { name: 'Rank request for Ragefist Ironjaw' })
    await userEvent.click(within(form).getByRole('button', { name: 'Send request' }))

    expect(await within(form).findByRole('alert')).toHaveTextContent(
      'this character already has an open rank request',
    )
  })

  it('should cancel an open rank request when Cancel request is clicked', async () => {
    let requests = [rankRequest({})]
    server.use(
      http.get('/api/auth/me', () => HttpResponse.json(linkedMember)),
      http.get('/api/me/characters', () =>
        HttpResponse.json([ownedCharacter('char-a', 'Ragefist', 'Ironjaw', true)]),
      ),
      http.get('/api/me/rank-requests', () => HttpResponse.json(requests)),
      http.delete('/api/me/rank-requests/request-1', () => {
        requests = []
        return new HttpResponse(null, { status: 204 })
      }),
    )
    renderWithProviders(<Dashboard />, '/dashboard')

    await userEvent.click(await screen.findByRole('button', { name: 'Cancel rank request for Ragefist Ironjaw' }))

    expect(await screen.findByRole('button', { name: 'Request alt rank for Ragefist Ironjaw' })).toBeInTheDocument()
    expect(screen.queryByText(/You asked for a move/)).not.toBeInTheDocument()
  })

  it("should show the officer's note when a rank request was declined", async () => {
    server.use(
      http.get('/api/auth/me', () => HttpResponse.json(linkedMember)),
      http.get('/api/me/characters', () =>
        HttpResponse.json([ownedCharacter('char-a', 'Ragefist', 'Ironjaw', true)]),
      ),
      http.get('/api/me/rank-requests', () =>
        HttpResponse.json([
          rankRequest({ status: 'declined', resolutionNote: 'Raid needs your main', resolvedBy: 'officerbob' }),
        ]),
      ),
    )
    renderWithProviders(<Dashboard />, '/dashboard')

    expect(
      await screen.findByText('Your request for a move to an alt rank was declined: Raid needs your main'),
    ).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Request alt rank for Ragefist Ironjaw' })).toBeInTheDocument()
  })

  it('should show officers a rank requests link with the pending count', async () => {
    server.use(
      http.get('/api/auth/me', () => HttpResponse.json(officer)),
      http.get('/api/rank-requests/count', () => HttpResponse.json({ open: 3 })),
    )

    renderWithProviders(<Dashboard />, '/dashboard')

    const section = await screen.findByRole('region', { name: 'Officer' })
    expect(within(section).getByRole('link', { name: 'Rank requests' })).toHaveAttribute(
      'href',
      '/officer/rank-requests',
    )
    expect(await within(section).findByText('3 pending')).toBeInTheDocument()
  })

  it("should update the officer's pending count when they file a rank request for their own character", async () => {
    let open = 0
    server.use(
      http.get('/api/auth/me', () => HttpResponse.json(officer)),
      http.get('/api/me/characters', () =>
        HttpResponse.json([ownedCharacter('char-a', 'Ragefist', 'Ironjaw', true)]),
      ),
      http.get('/api/rank-requests/count', () => HttpResponse.json({ open })),
      http.post('/api/me/characters/char-a/rank-requests', () => {
        open = 1
        return HttpResponse.json(rankRequest({}), { status: 201 })
      }),
    )
    renderWithProviders(<Dashboard />, '/dashboard')

    const section = await screen.findByRole('region', { name: 'Officer' })
    expect(await within(section).findByText('none pending')).toBeInTheDocument()
    await userEvent.click(await screen.findByRole('button', { name: 'Request alt rank for Ragefist Ironjaw' }))
    const form = screen.getByRole('form', { name: 'Rank request for Ragefist Ironjaw' })
    await userEvent.click(within(form).getByRole('button', { name: 'Send request' }))

    expect(await within(section).findByText('1 pending')).toBeInTheDocument()
  })

  it('should not show the rank requests link when the user is not an officer', async () => {
    let countRequested = false
    server.use(
      http.get('/api/auth/me', () => HttpResponse.json(linkedMember)),
      http.get('/api/rank-requests/count', () => {
        countRequested = true
        return HttpResponse.json({ open: 3 })
      }),
    )

    renderWithProviders(<Dashboard />, '/dashboard')

    await screen.findByRole('region', { name: 'Your characters' })
    expect(screen.queryByRole('link', { name: 'Rank requests' })).not.toBeInTheDocument()
    expect(countRequested).toBe(false)
  })
})
