import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '../../mocks/server'
import { renderWithProviders } from '../../testUtils'
import type { AuthUser } from '../../types/auth'
import type { MyCharacter } from '../../types/roster'
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

function ownedCharacter(id: string, name: string, secondaryName: string, isMain: boolean): MyCharacter {
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

// Each character's own line (name, class, main/alt), without its profession editor.
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

  it("should show each character's professions with their skill levels", async () => {
    server.use(http.get('/api/auth/me', () => HttpResponse.json(linkedMember)))

    renderWithProviders(<Dashboard />, '/dashboard')

    const professions = await screen.findByRole('list', { name: 'Professions of Thundermane Stormcaller' })
    expect(within(professions).getByRole('spinbutton', { name: 'Mining skill for Thundermane Stormcaller' })).toHaveValue(300)
    expect(screen.getByText('No professions yet.')).toBeInTheDocument()
  })

  it('should not offer a profession the character already knows when adding one', async () => {
    server.use(http.get('/api/auth/me', () => HttpResponse.json(linkedMember)))

    renderWithProviders(<Dashboard />, '/dashboard')

    const select = await screen.findByRole('combobox', { name: 'Profession to add to Thundermane Stormcaller' })
    const options = within(select)
      .getAllByRole('option')
      .map((option) => option.textContent)
    expect(options).toEqual([
      'Add a profession…',
      'Alchemy',
      'Blacksmithing',
      'Enchanting',
      'Engineering',
      'Herbalism',
      'Leatherworking',
      'Skinning',
      'Tailoring',
      'Cooking',
      'Fishing',
      'First Aid',
    ])
  })

  it('should add a profession to a character when the Add form is submitted', async () => {
    const professions: MyCharacter['professions'] = []
    let postBody: unknown
    server.use(
      http.get('/api/auth/me', () => HttpResponse.json(linkedMember)),
      http.get('/api/me/characters', () =>
        HttpResponse.json([{ ...ownedCharacter('char-a', 'Ragefist', 'Ironjaw', true), professions }]),
      ),
      http.post('/api/me/characters/char-a/professions', async ({ request }) => {
        postBody = await request.json()
        professions.push({ id: 'prof-1', profession: 'Tailoring', skillLevel: 150 })
        return HttpResponse.json({ id: 'prof-1', profession: 'Tailoring', skillLevel: 150 }, { status: 201 })
      }),
    )
    renderWithProviders(<Dashboard />, '/dashboard')

    await userEvent.selectOptions(
      await screen.findByRole('combobox', { name: 'Profession to add to Ragefist Ironjaw' }),
      'Tailoring',
    )
    const skill = screen.getByRole('spinbutton', { name: 'Skill for the new profession on Ragefist Ironjaw' })
    await userEvent.clear(skill)
    await userEvent.type(skill, '150')
    await userEvent.click(screen.getByRole('button', { name: 'Add' }))

    expect(
      await screen.findByRole('spinbutton', { name: 'Tailoring skill for Ragefist Ironjaw' }),
    ).toHaveValue(150)
    expect(postBody).toEqual({ profession: 'Tailoring', skillLevel: 150 })
  })

  it('should save a new skill level when it is changed and Save is clicked', async () => {
    let skillLevel = 300
    let patchBody: unknown
    server.use(
      http.get('/api/auth/me', () => HttpResponse.json(linkedMember)),
      http.get('/api/me/characters', () =>
        HttpResponse.json([
          {
            ...ownedCharacter('char-a', 'Ragefist', 'Ironjaw', true),
            professions: [{ id: 'prof-1', profession: 'Mining', skillLevel }],
          },
        ]),
      ),
      http.patch('/api/me/professions/prof-1', async ({ request }) => {
        patchBody = await request.json()
        skillLevel = 310
        return HttpResponse.json({ id: 'prof-1', profession: 'Mining', skillLevel })
      }),
    )
    renderWithProviders(<Dashboard />, '/dashboard')

    const skill = await screen.findByRole('spinbutton', { name: 'Mining skill for Ragefist Ironjaw' })
    expect(screen.getByRole('button', { name: 'Save' })).toBeDisabled()
    await userEvent.clear(skill)
    await userEvent.type(skill, '310')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    expect(patchBody).toEqual({ skillLevel: 310 })
    expect(await screen.findByRole('spinbutton', { name: 'Mining skill for Ragefist Ironjaw' })).toHaveValue(310)
    expect(screen.getByRole('button', { name: 'Save' })).toBeDisabled()
  })

  it('should remove a profession when its Remove button is clicked', async () => {
    let professions = [{ id: 'prof-1', profession: 'Mining', skillLevel: 300 }]
    server.use(
      http.get('/api/auth/me', () => HttpResponse.json(linkedMember)),
      http.get('/api/me/characters', () =>
        HttpResponse.json([{ ...ownedCharacter('char-a', 'Ragefist', 'Ironjaw', true), professions }]),
      ),
      http.delete('/api/me/professions/prof-1', () => {
        professions = []
        return new HttpResponse(null, { status: 204 })
      }),
    )
    renderWithProviders(<Dashboard />, '/dashboard')

    await userEvent.click(await screen.findByRole('button', { name: 'Remove Mining from Ragefist Ironjaw' }))

    expect(await screen.findByText('No professions yet.')).toBeInTheDocument()
    expect(screen.queryByRole('spinbutton', { name: 'Mining skill for Ragefist Ironjaw' })).not.toBeInTheDocument()
  })

  it("should show the server's message when adding a profession the character already has", async () => {
    server.use(
      http.get('/api/auth/me', () => HttpResponse.json(linkedMember)),
      http.post('/api/me/characters/:id/professions', () =>
        HttpResponse.json({ error: 'character already has this profession' }, { status: 409 }),
      ),
    )
    renderWithProviders(<Dashboard />, '/dashboard')

    await userEvent.selectOptions(
      await screen.findByRole('combobox', { name: 'Profession to add to Thundermend Lightward' }),
      'Alchemy',
    )
    await userEvent.click(within(screen.getByRole('form', { name: 'Add a profession to Thundermend Lightward' })).getByRole('button', { name: 'Add' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('character already has this profession')
  })
})
