import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '../../mocks/server'
import { renderWithProviders } from '../../testUtils'
import type { Character } from '../../types/roster'
import { OfficerRoster } from './OfficerRoster'

const officer = { id: 'user-1', username: 'Officer', avatarUrl: null, isOfficer: true }
const member = { id: 'user-2', username: 'Member', avatarUrl: null, isOfficer: false }

const ragnok: Character = {
  id: 'char-1',
  name: 'Ragnok',
  secondaryName: 'Ironhide',
  realm: 'Emberreach',
  class: 'Warrior',
  spec: 'Protection',
  role: 'tank',
  spec2: null,
  role2: null,
  isMain: true,
  raidTeam: 'Team Alpha',
  createdAt: '2026-01-01T00:00:00.000Z',
}

const selene: Character = {
  ...ragnok,
  id: 'char-2',
  name: 'Selene',
  secondaryName: 'Dawnsong',
  class: 'Priest',
  spec: 'Holy',
  role: 'healer',
  spec2: 'Shadow',
  role2: 'dps',
}

const zaldrix: Character = {
  ...ragnok,
  id: 'char-3',
  name: 'Zaldrix',
  secondaryName: 'Nightfall',
  class: 'Warlock',
  spec: 'Affliction',
  role: 'dps',
  isMain: false,
  raidTeam: null,
}

function signInAs(user: typeof officer) {
  server.use(http.get('/api/auth/me', () => HttpResponse.json(user)))
}

interface RosterCalls {
  created: unknown[]
  updated: { id: string; body: unknown }[]
  deleted: string[]
}

function serveRoster(initial: Character[]) {
  let characters = initial
  const calls: RosterCalls = { created: [], updated: [], deleted: [] }
  server.use(
    http.get('/api/roster', () => HttpResponse.json(characters)),
    http.post('/api/roster', async ({ request }) => {
      calls.created.push(await request.json())
      const created: Character = { ...zaldrix, id: 'char-new', name: 'Aeliana', secondaryName: 'Stormwind' }
      characters = [...characters, created]
      return HttpResponse.json(created, { status: 201 })
    }),
    http.patch('/api/roster/:id', async ({ request, params }) => {
      calls.updated.push({ id: String(params.id), body: await request.json() })
      return HttpResponse.json(characters.find((character) => character.id === params.id))
    }),
    http.delete('/api/roster/:id', ({ params }) => {
      calls.deleted.push(String(params.id))
      characters = characters.filter((character) => character.id !== params.id)
      return new HttpResponse(null, { status: 204 })
    }),
  )
  return calls
}

async function openAddDialog(user: ReturnType<typeof userEvent.setup>) {
  await user.click(await screen.findByRole('button', { name: 'Add character' }))
  return within(screen.getByRole('dialog', { name: 'Add character' }))
}

async function openEditDialog(user: ReturnType<typeof userEvent.setup>, name: string) {
  await user.click(await screen.findByRole('button', { name: `Edit ${name}` }))
  return within(screen.getByRole('dialog', { name: `Edit ${name}` }))
}

describe('OfficerRoster', () => {
  it('should show an officers-only message when the visitor is not an officer', async () => {
    signInAs(member)

    renderWithProviders(<OfficerRoster />, '/officer/roster')

    expect(await screen.findByText('Only officers can manage the roster.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Add character' })).not.toBeInTheDocument()
  })

  it('should show a login link when the visitor is anonymous', async () => {
    renderWithProviders(<OfficerRoster />, '/officer/roster')

    expect(await screen.findByText('Only officers can manage the roster.')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Log in with Discord' })).toHaveAttribute(
      'href',
      'http://localhost:3000/api/auth/login',
    )
  })

  it('should list every character including alts when the visitor is an officer', async () => {
    signInAs(officer)
    serveRoster([ragnok, selene, zaldrix])

    renderWithProviders(<OfficerRoster />, '/officer/roster')

    expect(await screen.findByText('Ragnok')).toBeInTheDocument()
    expect(screen.getByText('Selene')).toBeInTheDocument()
    expect(screen.getByText('Zaldrix')).toBeInTheDocument()
    expect(screen.getByText('Holy/Shadow')).toBeInTheDocument()
  })

  it('should show an error message when the roster cannot be loaded', async () => {
    signInAs(officer)
    server.use(http.get('/api/roster', () => new HttpResponse(null, { status: 500 })))

    renderWithProviders(<OfficerRoster />, '/officer/roster')

    expect(await screen.findByText('The roster could not be loaded.')).toBeInTheDocument()
  })

  it('should show an empty message when the roster has no characters', async () => {
    signInAs(officer)
    serveRoster([])

    renderWithProviders(<OfficerRoster />, '/officer/roster')

    expect(await screen.findByText('No characters yet.')).toBeInTheDocument()
  })

  it('should offer only the specs of the chosen class when an officer picks a class', async () => {
    signInAs(officer)
    serveRoster([])
    const user = userEvent.setup()
    renderWithProviders(<OfficerRoster />, '/officer/roster')
    const dialog = await openAddDialog(user)

    await user.selectOptions(dialog.getByRole('combobox', { name: 'Class' }), 'Priest')

    const specOptions = within(dialog.getByRole('combobox', { name: 'Spec' }))
      .getAllByRole('option')
      .map((option) => option.textContent)
    expect(specOptions).toEqual(['Select spec', 'Discipline', 'Holy', 'Shadow'])
  })

  it('should reset the spec choice when an officer changes the class', async () => {
    signInAs(officer)
    serveRoster([ragnok])
    const user = userEvent.setup()
    renderWithProviders(<OfficerRoster />, '/officer/roster')
    const dialog = await openEditDialog(user, 'Ragnok')
    expect(dialog.getByRole('combobox', { name: 'Spec' })).toHaveValue('Protection')

    await user.selectOptions(dialog.getByRole('combobox', { name: 'Class' }), 'Mage')

    expect(dialog.getByRole('combobox', { name: 'Spec' })).toHaveValue('')
  })

  it('should ask for a second role when an officer picks a second spec', async () => {
    signInAs(officer)
    serveRoster([])
    const user = userEvent.setup()
    renderWithProviders(<OfficerRoster />, '/officer/roster')
    const dialog = await openAddDialog(user)
    await user.selectOptions(dialog.getByRole('combobox', { name: 'Class' }), 'Rogue')
    expect(dialog.queryByRole('combobox', { name: 'Second role' })).not.toBeInTheDocument()

    await user.selectOptions(dialog.getByRole('combobox', { name: 'Second spec' }), 'Combat')

    expect(dialog.getByRole('combobox', { name: 'Second role' })).toBeInTheDocument()
  })

  it('should send the new character and show it in the list when an officer submits the add form', async () => {
    signInAs(officer)
    const calls = serveRoster([ragnok])
    const user = userEvent.setup()
    renderWithProviders(<OfficerRoster />, '/officer/roster')
    const dialog = await openAddDialog(user)

    await user.type(dialog.getByLabelText('Name'), 'Aeliana')
    await user.type(dialog.getByLabelText('Secondary name'), 'Stormwind')
    await user.selectOptions(dialog.getByRole('combobox', { name: 'Class' }), 'Priest')
    await user.selectOptions(dialog.getByRole('combobox', { name: 'Spec' }), 'Holy')
    await user.selectOptions(dialog.getByRole('combobox', { name: 'Role' }), 'healer')
    await user.click(dialog.getByRole('button', { name: 'Save character' }))

    expect(await screen.findByText('Aeliana')).toBeInTheDocument()
    expect(calls.created).toEqual([
      {
        name: 'Aeliana',
        secondaryName: 'Stormwind',
        class: 'Priest',
        spec: 'Holy',
        role: 'healer',
        isMain: true,
      },
    ])
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('should show the server message and keep the dialog open when the API rejects a new character with 400', async () => {
    signInAs(officer)
    serveRoster([])
    server.use(
      http.post('/api/roster', () =>
        HttpResponse.json({ error: 'name: must be between 2 and 12 characters' }, { status: 400 }),
      ),
    )
    const user = userEvent.setup()
    renderWithProviders(<OfficerRoster />, '/officer/roster')
    const dialog = await openAddDialog(user)
    await user.type(dialog.getByLabelText('Name'), 'A')
    await user.type(dialog.getByLabelText('Secondary name'), 'Stormwind')
    await user.selectOptions(dialog.getByRole('combobox', { name: 'Class' }), 'Priest')
    await user.selectOptions(dialog.getByRole('combobox', { name: 'Spec' }), 'Holy')

    await user.click(dialog.getByRole('button', { name: 'Save character' }))

    expect(await dialog.findByRole('alert')).toHaveTextContent('name: must be between 2 and 12 characters')
    expect(screen.getByRole('dialog', { name: 'Add character' })).toBeInTheDocument()
  })

  it('should show a duplicate message when the API returns 409', async () => {
    signInAs(officer)
    serveRoster([ragnok])
    server.use(
      http.post('/api/roster', () =>
        HttpResponse.json(
          { error: 'character with this name and secondary name already exists' },
          { status: 409 },
        ),
      ),
    )
    const user = userEvent.setup()
    renderWithProviders(<OfficerRoster />, '/officer/roster')
    const dialog = await openAddDialog(user)
    await user.type(dialog.getByLabelText('Name'), 'Ragnok')
    await user.type(dialog.getByLabelText('Secondary name'), 'Ironhide')
    await user.selectOptions(dialog.getByRole('combobox', { name: 'Class' }), 'Warrior')
    await user.selectOptions(dialog.getByRole('combobox', { name: 'Spec' }), 'Protection')

    await user.click(dialog.getByRole('button', { name: 'Save character' }))

    expect(await dialog.findByRole('alert')).toHaveTextContent(
      'character with this name and secondary name already exists',
    )
  })

  it('should send only the changed fields when an officer edits a character', async () => {
    signInAs(officer)
    const calls = serveRoster([ragnok, selene])
    const user = userEvent.setup()
    renderWithProviders(<OfficerRoster />, '/officer/roster')
    const dialog = await openEditDialog(user, 'Ragnok')

    await user.clear(dialog.getByLabelText('Raid team'))
    await user.type(dialog.getByLabelText('Raid team'), 'Team Beta')
    await user.click(dialog.getByRole('button', { name: 'Save character' }))

    await waitForDialogToClose()
    expect(calls.updated).toEqual([{ id: 'char-1', body: { raidTeam: 'Team Beta' } }])
  })

  it('should send the second spec and role when an officer adds a second spec', async () => {
    signInAs(officer)
    const calls = serveRoster([ragnok])
    const user = userEvent.setup()
    renderWithProviders(<OfficerRoster />, '/officer/roster')
    const dialog = await openEditDialog(user, 'Ragnok')

    await user.selectOptions(dialog.getByRole('combobox', { name: 'Second spec' }), 'Arms')
    await user.selectOptions(dialog.getByRole('combobox', { name: 'Second role' }), 'dps')
    await user.click(dialog.getByRole('button', { name: 'Save character' }))

    await waitForDialogToClose()
    expect(calls.updated).toEqual([{ id: 'char-1', body: { spec2: 'Arms', role2: 'dps' } }])
  })

  it('should send empty strings for the second spec and role when an officer removes the second spec', async () => {
    signInAs(officer)
    const calls = serveRoster([selene])
    const user = userEvent.setup()
    renderWithProviders(<OfficerRoster />, '/officer/roster')
    const dialog = await openEditDialog(user, 'Selene')

    await user.selectOptions(dialog.getByRole('combobox', { name: 'Second spec' }), 'None')
    await user.click(dialog.getByRole('button', { name: 'Save character' }))

    await waitForDialogToClose()
    expect(calls.updated).toEqual([{ id: 'char-2', body: { spec2: '', role2: '' } }])
  })

  it('should remove the character from the list when an officer confirms deletion', async () => {
    signInAs(officer)
    const calls = serveRoster([ragnok, selene])
    const user = userEvent.setup()
    renderWithProviders(<OfficerRoster />, '/officer/roster')
    await user.click(await screen.findByRole('button', { name: 'Delete Ragnok' }))

    await user.click(
      within(screen.getByRole('dialog', { name: 'Delete Ragnok' })).getByRole('button', { name: 'Delete character' }),
    )

    await waitForDialogToClose()
    expect(screen.queryByText('Ragnok')).not.toBeInTheDocument()
    expect(screen.getByText('Selene')).toBeInTheDocument()
    expect(calls.deleted).toEqual(['char-1'])
  })

  it('should keep the character when an officer cancels deletion', async () => {
    signInAs(officer)
    const calls = serveRoster([ragnok])
    const user = userEvent.setup()
    renderWithProviders(<OfficerRoster />, '/officer/roster')
    await user.click(await screen.findByRole('button', { name: 'Delete Ragnok' }))

    await user.click(
      within(screen.getByRole('dialog', { name: 'Delete Ragnok' })).getByRole('button', { name: 'Cancel' }),
    )

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(screen.getByText('Ragnok')).toBeInTheDocument()
    expect(calls.deleted).toEqual([])
  })
})

async function waitForDialogToClose() {
  await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
}
