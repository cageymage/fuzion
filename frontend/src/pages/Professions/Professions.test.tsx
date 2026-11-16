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

const brannoch = { id: 'char-3', name: 'Brannoch', secondaryName: 'Stonefist', class: 'Paladin' }
const smithingBrannoch: ProfessionEntry = { id: 'p6', profession: 'Blacksmithing', skillLevel: 275, character: brannoch }
const cookingAeliana: ProfessionEntry ={ id: 'p4', profession: 'Cooking', skillLevel: 150, character: aeliana }

function respondWith(entries: ProfessionEntry[]) {
  server.use(http.get('/api/professions', () => HttpResponse.json(entries)))
}

describe('Professions', () => {
  it('should select the first primary tab with characters when the page loads', async () => {
    respondWith([...professions, cookingAeliana])

    renderWithProviders(<Professions />)

    expect(await screen.findByRole('tab', { name: 'Alchemy (2)' })).toHaveAttribute('aria-selected', 'true')
    expect(screen.getByRole('tab', { name: 'Blacksmithing (1)' })).toHaveAttribute('aria-selected', 'false')
    expect(screen.getByRole('tab', { name: 'Cooking (1)' })).toHaveAttribute('aria-selected', 'false')
  })

  it('should select the first secondary tab with characters when no primary profession has characters', async () => {
    respondWith([cookingAeliana])

    renderWithProviders(<Professions />)

    expect(await screen.findByRole('tab', { name: 'Cooking (1)' })).toHaveAttribute('aria-selected', 'true')
  })

  it('should list primary and secondary professions in separate tab rows', async () => {
    respondWith([...professions, cookingAeliana])

    renderWithProviders(<Professions />)

    const primary = await screen.findByRole('tablist', { name: 'Primary professions' })
    const secondary = screen.getByRole('tablist', { name: 'Secondary professions' })
    expect(within(primary).getAllByRole('tab').map((tab) => tab.textContent)).toEqual([
      'Alchemy (2)',
      'Blacksmithing (1)',
      'Enchanting (0)',
      'Engineering (0)',
      'Herbalism (0)',
      'Leatherworking (0)',
      'Mining (0)',
      'Skinning (0)',
      'Tailoring (0)',
    ])
    expect(within(secondary).getAllByRole('tab').map((tab) => tab.textContent)).toEqual([
      'Cooking (1)',
      'Fishing (0)',
      'First Aid (0)',
    ])
  })

  it('should show only the selected profession panel when several professions have characters', async () => {
    respondWith(professions)

    renderWithProviders(<Professions />)

    const panel = await screen.findByRole('tabpanel', { name: 'Alchemy (2)' })
    expect(within(panel).getAllByRole('listitem')).toHaveLength(2)
    expect(screen.getAllByRole('tabpanel')).toHaveLength(1)
  })

  it('should show the characters of another profession when its tab is clicked', async () => {
    respondWith(professions)
    const user = userEvent.setup()
    renderWithProviders(<Professions />)

    await user.click(await screen.findByRole('tab', { name: 'Blacksmithing (1)' }))

    const panel = screen.getByRole('tabpanel', { name: 'Blacksmithing (1)' })
    expect(within(panel).getAllByRole('listitem')).toHaveLength(1)
    expect(within(panel).getByText('Zephyrion')).toBeInTheDocument()
  })

  it("should show each character's skill level in the order the api returned them", async () => {
    respondWith(professions)

    renderWithProviders(<Professions />)

    const panel = await screen.findByRole('tabpanel')
    const [first, second] = within(panel).getAllByRole('listitem')
    expect(first).toHaveTextContent('Aeliana')
    expect(first).toHaveTextContent('300')
    expect(second).toHaveTextContent('Zephyrion')
    expect(second).toHaveTextContent('225')
  })

  it('should name the first-aid tab after the whole profession when the profession has several words', async () => {
    respondWith([{ id: 'p9', profession: 'First Aid', skillLevel: 150, character: aeliana }])

    renderWithProviders(<Professions />)

    const panel = await screen.findByRole('tabpanel', { name: 'First Aid (1)' })
    expect(within(panel).getByText('Aeliana')).toBeInTheDocument()
  })

  it('should update the tab counts to the matching characters when searching by character name', async () => {
    respondWith(professions)
    const user = userEvent.setup()
    renderWithProviders(<Professions />)
    await screen.findByRole('tab', { name: 'Alchemy (2)' })

    await user.type(screen.getByRole('searchbox', { name: 'Search' }), 'zephyr')

    expect(screen.getByRole('tab', { name: 'Alchemy (1)' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Blacksmithing (1)' })).toBeInTheDocument()
    const panel = screen.getByRole('tabpanel')
    expect(within(panel).getByText('Zephyrion')).toBeInTheDocument()
    expect(within(panel).queryByText('Aeliana')).not.toBeInTheDocument()
  })

  it('should show a no-matches message when the search text is only a profession name', async () => {
    respondWith(professions)
    const user = userEvent.setup()
    renderWithProviders(<Professions />)
    await screen.findByRole('tab', { name: 'Alchemy (2)' })

    await user.type(screen.getByRole('searchbox', { name: 'Search' }), 'alch')

    expect(screen.getByText('No matches for "alch".')).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Alchemy (0)' })).toHaveAttribute('aria-disabled', 'true')
  })

  it('should disable a tab and keep it unselected when it has no characters', async () => {
    respondWith(professions)
    const user = userEvent.setup()
    renderWithProviders(<Professions />)
    const tailoring = await screen.findByRole('tab', { name: 'Tailoring (0)' })

    await user.click(tailoring)

    expect(tailoring).toHaveAttribute('aria-disabled', 'true')
    expect(tailoring).toHaveAttribute('aria-selected', 'false')
    expect(screen.getByRole('tab', { name: 'Alchemy (2)' })).toHaveAttribute('aria-selected', 'true')
  })

  it('should select the first tab that still has matches when a search empties the selected tab', async () => {
    respondWith([...professions, smithingBrannoch])
    const user = userEvent.setup()
    renderWithProviders(<Professions />)
    await screen.findByRole('tab', { name: 'Alchemy (2)' })

    await user.type(screen.getByRole('searchbox', { name: 'Search' }), 'brann')

    expect(screen.getByRole('tab', { name: 'Alchemy (0)' })).toHaveAttribute('aria-disabled', 'true')
    expect(screen.getByRole('tab', { name: 'Blacksmithing (1)' })).toHaveAttribute('aria-selected', 'true')
  })

  it('should keep the jumped-to tab selected when the search is cleared', async () => {
    respondWith([...professions, smithingBrannoch])
    const user = userEvent.setup()
    renderWithProviders(<Professions />)
    await screen.findByRole('tab', { name: 'Alchemy (2)' })
    const search = screen.getByRole('searchbox', { name: 'Search' })
    await user.type(search, 'brann')

    await user.clear(search)

    expect(screen.getByRole('tab', { name: 'Blacksmithing (2)' })).toHaveAttribute('aria-selected', 'true')
  })

  it('should move focus and selection to the next enabled tab when the right arrow is pressed', async () => {
    respondWith(professions)
    const user = userEvent.setup()
    renderWithProviders(<Professions />)
    const alchemy = await screen.findByRole('tab', { name: 'Alchemy (2)' })
    alchemy.focus()

    await user.keyboard('{ArrowRight}')

    const blacksmithing = screen.getByRole('tab', { name: 'Blacksmithing (1)' })
    expect(blacksmithing).toHaveFocus()
    expect(blacksmithing).toHaveAttribute('aria-selected', 'true')
  })

  it('should skip disabled tabs when the right arrow is pressed', async () => {
    respondWith([...professions, { id: 'p5', profession: 'Tailoring', skillLevel: 100, character: aeliana }])
    const user = userEvent.setup()
    renderWithProviders(<Professions />)
    const blacksmithing = await screen.findByRole('tab', { name: 'Blacksmithing (1)' })
    await user.click(blacksmithing)

    await user.keyboard('{ArrowRight}')

    const tailoring = screen.getByRole('tab', { name: 'Tailoring (1)' })
    expect(tailoring).toHaveFocus()
    expect(tailoring).toHaveAttribute('aria-selected', 'true')
  })

  it('should wrap to the last enabled tab in the row when the left arrow is pressed on the first tab', async () => {
    respondWith(professions)
    const user = userEvent.setup()
    renderWithProviders(<Professions />)
    const alchemy = await screen.findByRole('tab', { name: 'Alchemy (2)' })
    alchemy.focus()

    await user.keyboard('{ArrowLeft}')

    expect(screen.getByRole('tab', { name: 'Blacksmithing (1)' })).toHaveFocus()
  })

  it('should make the first enabled secondary tab a tab stop when the selection is in the primary row', async () => {
    respondWith([...professions, cookingAeliana])

    renderWithProviders(<Professions />)

    expect(await screen.findByRole('tab', { name: 'Cooking (1)' })).toHaveAttribute('tabindex', '0')
    expect(screen.getByRole('tab', { name: 'Fishing (0)' })).toHaveAttribute('tabindex', '-1')
    expect(screen.getByRole('tab', { name: 'Alchemy (2)' })).toHaveAttribute('tabindex', '0')
    expect(screen.getByRole('tab', { name: 'Blacksmithing (1)' })).toHaveAttribute('tabindex', '-1')
  })

  it('should keep arrow navigation inside the primary row when the right arrow is pressed on its last enabled tab', async () => {
    respondWith([...professions, cookingAeliana])
    const user = userEvent.setup()
    renderWithProviders(<Professions />)
    await user.click(await screen.findByRole('tab', { name: 'Blacksmithing (1)' }))

    await user.keyboard('{ArrowRight}')

    expect(screen.getByRole('tab', { name: 'Alchemy (2)' })).toHaveFocus()
  })

  it('should show a no-matches message and select no tab when the search matches nothing', async () => {
    respondWith(professions)
    const user = userEvent.setup()
    renderWithProviders(<Professions />)
    await screen.findByRole('tab', { name: 'Alchemy (2)' })

    await user.type(screen.getByRole('searchbox', { name: 'Search' }), 'dragonscale')

    expect(screen.getByText('No matches for "dragonscale".')).toBeInTheDocument()
    expect(screen.queryByRole('tabpanel')).not.toBeInTheDocument()
    for (const tab of screen.getAllByRole('tab')) {
      expect(tab).toHaveAttribute('aria-selected', 'false')
      expect(tab).toHaveAttribute('aria-disabled', 'true')
    }
  })

  it('should color the secondary name with the class color when an entry is rendered', async () => {
    // given a Warrior whose secondary name is Ironhide
    respondWith([{ id: 'p2', profession: 'Alchemy', skillLevel: 225, character: zephyrion }])

    // when the page renders the entry
    renderWithProviders(<Professions />)

    // then both halves of the name use the Warrior class color
    const panel = await screen.findByRole('tabpanel')
    expect(within(panel).getByText('Zephyrion')).toHaveStyle({ color: '#C79C6E' })
    expect(within(panel).getByText('Ironhide')).toHaveStyle({ color: '#C79C6E' })
  })

  it('should show a dash instead of a skill level when none is recorded', async () => {
    respondWith([{ id: 'p8', profession: 'Cooking', skillLevel: 0, character: aeliana }])

    renderWithProviders(<Professions />)

    const panel = await screen.findByRole('tabpanel')
    expect(within(panel).getByRole('listitem')).toHaveTextContent('—')
    expect(within(panel).getByRole('listitem')).not.toHaveTextContent('0')
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
