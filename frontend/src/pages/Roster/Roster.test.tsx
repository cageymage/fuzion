import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '../../mocks/server'
import { renderWithProviders } from '../../testUtils'
import type { Character } from '../../types/roster'
import { Roster } from './Roster'

const ragnok: Character = {
  id: 'char-1',
  name: 'Ragnok',
  secondaryName: '',
  realm: 'Emberreach',
  class: 'Warrior',
  spec: 'Protection',
  role: 'tank',
  spec2: null,
  role2: null,
  isMain: true,
  raidTeam: 'Team Alpha',
  createdAt: '2026-01-01T00:00:00.000Z',
  professions: [],
}

const selene: Character = {
  ...ragnok,
  id: 'char-2',
  name: 'Selene',
  class: 'Priest',
  spec: 'Holy',
  role: 'healer',
}

const zaldrix: Character = {
  ...ragnok,
  id: 'char-3',
  name: 'Zaldrix',
  class: 'Warlock',
  spec: 'Affliction',
  role: 'dps',
  isMain: false,
  raidTeam: null,
}

const roster = [ragnok, selene, zaldrix]

describe('Roster', () => {
  it('should list only main characters by default', async () => {
    server.use(http.get('/api/roster', () => HttpResponse.json(roster)))

    renderWithProviders(<Roster />)

    expect(await screen.findByText('Ragnok')).toBeInTheDocument()
    expect(screen.getByText('Selene')).toBeInTheDocument()
    expect(screen.queryByText('Zaldrix')).not.toBeInTheDocument()
  })

  it('should include alts when the show-alts checkbox is checked', async () => {
    server.use(http.get('/api/roster', () => HttpResponse.json(roster)))
    const user = userEvent.setup()
    renderWithProviders(<Roster />)
    await screen.findByText('Ragnok')

    await user.click(screen.getByRole('checkbox', { name: 'Show alts' }))

    expect(screen.getByText('Zaldrix')).toBeInTheDocument()
  })

  it('should only show healers when the Healer role filter is selected', async () => {
    server.use(http.get('/api/roster', () => HttpResponse.json(roster)))
    const user = userEvent.setup()
    renderWithProviders(<Roster />)
    await screen.findByText('Ragnok')

    await user.click(screen.getByRole('button', { name: 'Healer' }))

    expect(screen.getByText('Selene')).toBeInTheDocument()
    expect(screen.queryByText('Ragnok')).not.toBeInTheDocument()
  })

  it('should show the secondary name next to the name when the character has one', async () => {
    const withSecondaryName: Character = { ...ragnok, secondaryName: 'Ironhide' }
    server.use(http.get('/api/roster', () => HttpResponse.json([withSecondaryName])))

    renderWithProviders(<Roster />)

    const table = within(await screen.findByRole('table'))
    expect(table.getByRole('columnheader', { name: 'Secondary name' })).toBeInTheDocument()
    expect(table.getByText('Ironhide')).toBeInTheDocument()
  })

  it('should link to the roster manager when the visitor is an officer', async () => {
    server.use(
      http.get('/api/roster', () => HttpResponse.json(roster)),
      http.get('/api/auth/me', () =>
        HttpResponse.json({ id: 'user-1', username: 'Officer', avatarUrl: null, isOfficer: true }),
      ),
    )

    renderWithProviders(<Roster />)

    expect(await screen.findByRole('link', { name: 'Manage roster' })).toHaveAttribute('href', '/officer/roster')
  })

  it('should not link to the roster manager when the visitor is anonymous', async () => {
    server.use(http.get('/api/roster', () => HttpResponse.json(roster)))

    renderWithProviders(<Roster />)
    await screen.findByText('Ragnok')

    expect(screen.queryByRole('link', { name: 'Manage roster' })).not.toBeInTheDocument()
  })

  it('should show both specs joined by a slash when the character has a second spec', async () => {
    const dualSpec: Character = { ...selene, spec: 'Holy', spec2: 'Shadow', role2: 'dps' }
    server.use(http.get('/api/roster', () => HttpResponse.json([dualSpec])))

    renderWithProviders(<Roster />)

    expect(await screen.findByText('Holy/Shadow')).toBeInTheDocument()
  })

  it('should show both roles joined by a slash when the second role differs from the first', async () => {
    const dualRole: Character = { ...selene, spec: 'Holy', spec2: 'Shadow', role2: 'dps' }
    server.use(http.get('/api/roster', () => HttpResponse.json([dualRole])))

    renderWithProviders(<Roster />)

    expect(await screen.findByText('Healer/DPS')).toBeInTheDocument()
  })

  it('should show a single role when the second role matches the first', async () => {
    const sameRole: Character = { ...ragnok, spec: 'Protection', spec2: 'Arms', role2: 'tank' }
    server.use(http.get('/api/roster', () => HttpResponse.json([sameRole])))

    renderWithProviders(<Roster />)

    const table = within(await screen.findByRole('table'))
    expect(table.getByText('Protection/Arms')).toBeInTheDocument()
    expect(table.getByText('Tank')).toBeInTheDocument()
    expect(table.queryByText('Tank/Tank')).not.toBeInTheDocument()
  })

  it('should include a character whose second role matches when the Healer role filter is selected', async () => {
    const tankHealer: Character = { ...ragnok, id: 'char-5', name: 'Aeliana', spec2: 'Holy', role2: 'healer' }
    server.use(http.get('/api/roster', () => HttpResponse.json([ragnok, selene, tankHealer])))
    const user = userEvent.setup()
    renderWithProviders(<Roster />)
    await screen.findByText('Ragnok')

    await user.click(screen.getByRole('button', { name: 'Healer' }))

    expect(screen.getByText('Selene')).toBeInTheDocument()
    expect(screen.getByText('Aeliana')).toBeInTheDocument()
    expect(screen.queryByText('Ragnok')).not.toBeInTheDocument()
  })

  it('should combine the role filter and the class filter', async () => {
    const priestDps: Character = { ...selene, id: 'char-4', name: 'Aeliana', role: 'dps' }
    server.use(http.get('/api/roster', () => HttpResponse.json([ragnok, selene, priestDps])))
    const user = userEvent.setup()
    renderWithProviders(<Roster />)
    await screen.findByText('Ragnok')

    await user.click(screen.getByRole('button', { name: 'DPS' }))
    await user.selectOptions(screen.getByRole('combobox', { name: 'Class' }), 'Priest')

    expect(screen.getByText('Aeliana')).toBeInTheDocument()
    expect(screen.queryByText('Selene')).not.toBeInTheDocument()
    expect(screen.queryByText('Ragnok')).not.toBeInTheDocument()
  })

  it('should show an empty message when no characters match the filters', async () => {
    server.use(http.get('/api/roster', () => HttpResponse.json(roster)))
    const user = userEvent.setup()
    renderWithProviders(<Roster />)
    await screen.findByText('Ragnok')

    await user.click(screen.getByRole('button', { name: 'Healer' }))
    await user.selectOptions(screen.getByRole('combobox', { name: 'Class' }), 'Warrior')

    expect(screen.getByText('No characters match.')).toBeInTheDocument()
  })

  it('should show an error message when the roster request fails', async () => {
    server.use(http.get('/api/roster', () => new HttpResponse(null, { status: 500 })))

    renderWithProviders(<Roster />)

    expect(await screen.findByText('The roster could not be loaded.')).toBeInTheDocument()
  })

  it('should show two profession icons for a character with two primary professions', async () => {
    const crafter: Character = {
      ...ragnok,
      professions: [
        { profession: 'Mining', skillLevel: 300 },
        { profession: 'Blacksmithing', skillLevel: 225 },
      ],
    }
    server.use(http.get('/api/roster', () => HttpResponse.json([crafter])))

    renderWithProviders(<Roster />)

    const row = (await screen.findByText('Ragnok')).closest('tr') as HTMLElement
    const icons = within(row).getAllByRole('img')
    expect(icons.map((icon) => icon.getAttribute('alt'))).toEqual(['Mining', 'Blacksmithing'])
    expect(icons[0]).toHaveAttribute(
      'src',
      'https://render.worldofwarcraft.com/us/icons/36/trade_mining.jpg',
    )
  })

  it('should show no profession icons when the character has only a secondary profession', async () => {
    const cook: Character = { ...ragnok, professions: [{ profession: 'Cooking', skillLevel: 300 }] }
    server.use(http.get('/api/roster', () => HttpResponse.json([cook])))

    renderWithProviders(<Roster />)

    const row = (await screen.findByText('Ragnok')).closest('tr') as HTMLElement
    expect(within(row).queryByRole('img')).not.toBeInTheDocument()
  })

  it('should show no profession icons when the character has no professions', async () => {
    server.use(http.get('/api/roster', () => HttpResponse.json([ragnok])))

    renderWithProviders(<Roster />)

    const row = (await screen.findByText('Ragnok')).closest('tr') as HTMLElement
    expect(within(row).queryByRole('img')).not.toBeInTheDocument()
  })
})
