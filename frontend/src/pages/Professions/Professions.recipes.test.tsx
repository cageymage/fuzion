import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '../../mocks/server'
import { renderWithProviders } from '../../testUtils'
import type { ProfessionEntry, Recipe } from '../../types/professions'
import { Professions } from './Professions'

const aeliana = { id: 'char-1', name: 'Aeliana', secondaryName: 'Dawnsong', class: 'Priest' }
const zephyrion = { id: 'char-2', name: 'Zephyrion', secondaryName: 'Ironhide', class: 'Warrior' }

const professions: ProfessionEntry[] = [
  { id: 'p1', profession: 'Alchemy', skillLevel: 300, character: aeliana },
  { id: 'p2', profession: 'Alchemy', skillLevel: 225, character: zephyrion },
  { id: 'p3', profession: 'Blacksmithing', skillLevel: 300, character: zephyrion },
]

const elixir: Recipe = { id: 2329, name: "Elixir of Lion's Strength", profession: 'Alchemy', characters: [aeliana] }
const brewOfWorms: Recipe = { id: 9999, name: 'Brew of Worms', profession: 'Cooking', characters: [zephyrion] }

function respondWithProfessions() {
  server.use(http.get('/api/professions', () => HttpResponse.json(professions)))
}

function respondWithRecipes(handler: (params: URLSearchParams) => Recipe[]) {
  server.use(http.get('/api/recipes', ({ request }) => HttpResponse.json(handler(new URL(request.url).searchParams))))
}

describe('Professions recipe search', () => {
  it('should show who can craft a recipe when the search text matches its name', async () => {
    respondWithProfessions()
    respondWithRecipes((params) => (params.get('q') === 'lion' ? [elixir] : []))
    const user = userEvent.setup()
    renderWithProviders(<Professions />)

    await user.type(screen.getByRole('searchbox', { name: 'Search' }), 'lion')

    const block = await screen.findByRole('region', { name: 'Who can craft it' })
    const item = within(block).getByRole('listitem')
    expect(item).toHaveTextContent("Elixir of Lion's Strength")
    expect(item).toHaveTextContent('Alchemy')
    expect(item).toHaveTextContent('Aeliana Dawnsong')
  })

  it('should link each matched recipe to a Wowhead Forever search for its name', async () => {
    respondWithProfessions()
    respondWithRecipes(() => [elixir])
    const user = userEvent.setup()
    renderWithProviders(<Professions />)

    await user.type(screen.getByRole('searchbox', { name: 'Search' }), 'lion')

    const link = await screen.findByRole('link', { name: "Elixir of Lion's Strength" })
    expect(link).toHaveAttribute('href', "https://www.wowhead.com/forever/search?q=Elixir%20of%20Lion's%20Strength")
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveAttribute('rel', 'noreferrer')
  })

  it('should not ask for recipes or show the crafters block when the search box is empty', async () => {
    respondWithProfessions()
    let requests = 0
    respondWithRecipes(() => {
      requests++
      return [elixir]
    })

    renderWithProviders(<Professions />)

    await screen.findByRole('region', { name: 'Alchemy' })
    expect(requests).toBe(0)
    expect(screen.queryByRole('region', { name: 'Who can craft it' })).not.toBeInTheDocument()
  })

  it('should not ask for recipes when the search text is a single character', async () => {
    respondWithProfessions()
    let requests = 0
    respondWithRecipes(() => {
      requests++
      return [elixir]
    })
    const user = userEvent.setup()
    renderWithProviders(<Professions />)
    await screen.findByRole('region', { name: 'Alchemy' })

    await user.type(screen.getByRole('searchbox', { name: 'Search' }), 'e')

    expect(requests).toBe(0)
  })

  it('should narrow the directory to the characters who craft the matched recipe when the text matches no character', async () => {
    respondWithProfessions()
    respondWithRecipes(() => [elixir])
    const user = userEvent.setup()
    renderWithProviders(<Professions />)
    await screen.findByRole('region', { name: 'Alchemy' })

    await user.type(screen.getByRole('searchbox', { name: 'Search' }), 'lion')

    await screen.findByRole('region', { name: 'Who can craft it' })
    const alchemy = screen.getByRole('region', { name: 'Alchemy' })
    expect(within(alchemy).getAllByRole('listitem')).toHaveLength(1)
    expect(within(alchemy).getByText('Aeliana')).toBeInTheDocument()
    expect(screen.queryByRole('region', { name: 'Blacksmithing' })).not.toBeInTheDocument()
  })

  it('should show a no-match message when neither the directory nor the recipes match the search', async () => {
    respondWithProfessions()
    respondWithRecipes(() => [])
    const user = userEvent.setup()
    renderWithProviders(<Professions />)
    await screen.findByRole('region', { name: 'Alchemy' })

    await user.type(screen.getByRole('searchbox', { name: 'Search' }), 'dragonscale')

    expect(await screen.findByText('No matches for "dragonscale".')).toBeInTheDocument()
  })

  it('should show an error message in the crafters block when the recipes request fails', async () => {
    respondWithProfessions()
    server.use(http.get('/api/recipes', () => new HttpResponse(null, { status: 500 })))
    const user = userEvent.setup()
    renderWithProviders(<Professions />)

    await user.type(screen.getByRole('searchbox', { name: 'Search' }), 'lion')

    expect(await screen.findByText('Recipes could not be loaded.')).toBeInTheDocument()
  })
})

describe('Professions character recipes', () => {
  it("should not ask for a character's recipes when the character row is collapsed", async () => {
    respondWithProfessions()
    let requests = 0
    respondWithRecipes(() => {
      requests++
      return []
    })

    renderWithProviders(<Professions />)

    await screen.findByRole('region', { name: 'Alchemy' })
    expect(requests).toBe(0)
  })

  it("should list the character's recipes for that profession when the row is expanded", async () => {
    respondWithProfessions()
    const characterIds: (string | null)[] = []
    respondWithRecipes((params) => {
      characterIds.push(params.get('characterId'))
      return [elixir, brewOfWorms]
    })
    const user = userEvent.setup()
    renderWithProviders(<Professions />)
    const alchemy = await screen.findByRole('region', { name: 'Alchemy' })

    await user.click(within(alchemy).getByRole('button', { name: /Aeliana/ }))

    expect(await within(alchemy).findByText("Elixir of Lion's Strength")).toBeInTheDocument()
    expect(within(alchemy).queryByText('Brew of Worms')).not.toBeInTheDocument()
    expect(characterIds).toEqual(['char-1'])
  })

  it('should hide the recipes again when an expanded row is clicked a second time', async () => {
    respondWithProfessions()
    respondWithRecipes(() => [elixir])
    const user = userEvent.setup()
    renderWithProviders(<Professions />)
    const alchemy = await screen.findByRole('region', { name: 'Alchemy' })
    const row = within(alchemy).getByRole('button', { name: /Aeliana/ })
    await user.click(row)
    await within(alchemy).findByText("Elixir of Lion's Strength")

    await user.click(row)

    expect(within(alchemy).queryByText("Elixir of Lion's Strength")).not.toBeInTheDocument()
    expect(row).toHaveAttribute('aria-expanded', 'false')
  })

  it('should say no recipes are recorded when the character has none in that profession', async () => {
    respondWithProfessions()
    respondWithRecipes(() => [brewOfWorms])
    const user = userEvent.setup()
    renderWithProviders(<Professions />)
    const alchemy = await screen.findByRole('region', { name: 'Alchemy' })

    await user.click(within(alchemy).getByRole('button', { name: /Aeliana/ }))

    expect(await within(alchemy).findByText('No recipes recorded yet.')).toBeInTheDocument()
  })

  it('should show only the first eight recipes until show all is clicked when the character knows more', async () => {
    respondWithProfessions()
    const many: Recipe[] = Array.from({ length: 10 }, (_, index) => ({
      id: 100 + index,
      name: `Elixir ${index + 1}`,
      profession: 'Alchemy',
      characters: [aeliana],
    }))
    respondWithRecipes(() => many)
    const user = userEvent.setup()
    renderWithProviders(<Professions />)
    const alchemy = await screen.findByRole('region', { name: 'Alchemy' })
    await user.click(within(alchemy).getByRole('button', { name: /Aeliana/ }))
    await within(alchemy).findByText('Elixir 1')
    expect(within(alchemy).queryByText('Elixir 9')).not.toBeInTheDocument()

    await user.click(within(alchemy).getByRole('button', { name: 'Show all 10' }))

    expect(within(alchemy).getByText('Elixir 10')).toBeInTheDocument()
  })

  it("should show an error message in the row when the character's recipes fail to load", async () => {
    respondWithProfessions()
    server.use(http.get('/api/recipes', () => new HttpResponse(null, { status: 500 })))
    const user = userEvent.setup()
    renderWithProviders(<Professions />)
    const alchemy = await screen.findByRole('region', { name: 'Alchemy' })

    await user.click(within(alchemy).getByRole('button', { name: /Aeliana/ }))

    expect(await within(alchemy).findByText('Recipes could not be loaded.')).toBeInTheDocument()
  })
})
