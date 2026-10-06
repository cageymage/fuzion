import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '../../mocks/server'
import { renderWithProviders } from '../../testUtils'
import type { ProfessionEntry } from '../../types/professions'
import { Professions } from './Professions'

const aeliana = { id: 'char-1', name: 'Aeliana', secondaryName: 'Dawnsong', class: 'Priest' }
const zephyrion = { id: 'char-2', name: 'Zephyrion', secondaryName: 'Ironhide', class: 'Warrior' }

const professions: ProfessionEntry[] = [
  { id: 'p1', profession: 'Alchemy', skillLevel: 300, character: aeliana },
  { id: 'p2', profession: 'Alchemy', skillLevel: 225, character: zephyrion },
  { id: 'p3', profession: 'Blacksmithing', skillLevel: 300, character: zephyrion },
]

function respondWith(entries: ProfessionEntry[]) {
  server.use(http.get('/api/professions', () => HttpResponse.json(entries)))
}

describe('Professions', () => {
  it('should group characters under their profession', async () => {
    respondWith(professions)

    renderWithProviders(<Professions />)

    const alchemy = await screen.findByRole('region', { name: 'Alchemy' })
    expect(within(alchemy).getAllByRole('listitem')).toHaveLength(2)
    expect(within(alchemy).getByText('Aeliana')).toBeInTheDocument()
    expect(within(alchemy).getByText('Zephyrion')).toBeInTheDocument()
    const blacksmithing = screen.getByRole('region', { name: 'Blacksmithing' })
    expect(within(blacksmithing).getAllByRole('listitem')).toHaveLength(1)
    expect(within(blacksmithing).getByText('Zephyrion')).toBeInTheDocument()
  })

  it('should name the group after the whole profession when the profession has several words', async () => {
    respondWith([{ id: 'p9', profession: 'First Aid', skillLevel: 150, character: aeliana }])

    renderWithProviders(<Professions />)

    const firstAid = await screen.findByRole('region', { name: 'First Aid' })
    expect(within(firstAid).getByText('Aeliana')).toBeInTheDocument()
  })

  it("should show each character's skill level in the order the api returned them", async () => {
    respondWith(professions)

    renderWithProviders(<Professions />)

    const alchemy = await screen.findByRole('region', { name: 'Alchemy' })
    const [first, second] = within(alchemy).getAllByRole('listitem')
    expect(first).toHaveTextContent('Aeliana')
    expect(first).toHaveTextContent('300')
    expect(second).toHaveTextContent('Zephyrion')
    expect(second).toHaveTextContent('225')
  })

  it('should only show groups matching the search text when searching by profession', async () => {
    respondWith(professions)
    const user = userEvent.setup()
    renderWithProviders(<Professions />)
    await screen.findByRole('region', { name: 'Alchemy' })

    await user.type(screen.getByRole('searchbox', { name: 'Search' }), 'black')

    expect(screen.getByRole('region', { name: 'Blacksmithing' })).toBeInTheDocument()
    expect(screen.queryByRole('region', { name: 'Alchemy' })).not.toBeInTheDocument()
  })

  it("should show a character's professions when searching by character name", async () => {
    respondWith(professions)
    const user = userEvent.setup()
    renderWithProviders(<Professions />)
    await screen.findByRole('region', { name: 'Alchemy' })

    await user.type(screen.getByRole('searchbox', { name: 'Search' }), 'zephyr')

    const alchemy = screen.getByRole('region', { name: 'Alchemy' })
    expect(within(alchemy).getAllByRole('listitem')).toHaveLength(1)
    expect(within(alchemy).getByText('Zephyrion')).toBeInTheDocument()
    expect(within(alchemy).queryByText('Aeliana')).not.toBeInTheDocument()
    expect(screen.getByRole('region', { name: 'Blacksmithing' })).toBeInTheDocument()
  })

  it('should show a no-matches message when the search matches nothing', async () => {
    respondWith(professions)
    const user = userEvent.setup()
    renderWithProviders(<Professions />)
    await screen.findByRole('region', { name: 'Alchemy' })

    await user.type(screen.getByRole('searchbox', { name: 'Search' }), 'tailoring')

    expect(screen.getByText('No matches for "tailoring".')).toBeInTheDocument()
    expect(screen.queryByRole('region', { name: 'Alchemy' })).not.toBeInTheDocument()
  })

  it('should show an empty message when nobody has recorded a profession', async () => {
    respondWith([])

    renderWithProviders(<Professions />)

    expect(await screen.findByText('Nobody has recorded a profession yet.')).toBeInTheDocument()
  })

  it('should show a loading message when the professions request is still pending', () => {
    server.use(http.get('/api/professions', () => new Promise<never>(() => {})))

    renderWithProviders(<Professions />)

    expect(screen.getByText('Loading professions…')).toBeInTheDocument()
  })

  it('should show an error message when the professions request fails', async () => {
    server.use(http.get('/api/professions', () => new HttpResponse(null, { status: 500 })))

    renderWithProviders(<Professions />)

    expect(await screen.findByText('Professions could not be loaded.')).toBeInTheDocument()
  })
})
