import { screen } from '@testing-library/react'
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
  isMain: true,
  raidTeam: 'Team Alpha',
  createdAt: '2026-01-01T00:00:00.000Z',
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
})
