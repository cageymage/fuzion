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
  race: null,
  level: null,
  faction: null,
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

describe('Roster race and level', () => {
  it('should show the race as an icon with the race name as its accessible name when the character has a race', async () => {
    server.use(http.get('/api/roster', () => HttpResponse.json([{ ...ragnok, race: 'Orc', level: 60, faction: 'Horde' }])))

    renderWithProviders(<Roster />)

    const row = (await screen.findByText('Ragnok')).closest('tr') as HTMLElement
    const icon = within(row).getByRole('img', { name: 'Orc' })
    expect(icon).toHaveAttribute('title', 'Orc')
    expect(icon).toHaveAttribute(
      'src',
      'https://render.worldofwarcraft.com/us/icons/36/achievement_character_orc_male.jpg',
    )
    expect(within(row).queryByText('Orc')).not.toBeInTheDocument()
  })

  it('should show a generic badge with the race name as its accessible name when the race has no icon', async () => {
    server.use(
      http.get('/api/roster', () =>
        HttpResponse.json([{ ...ragnok, race: 'Skyborne (Windshaper)', level: 12, faction: 'Horde' }]),
      ),
    )

    renderWithProviders(<Roster />)

    const row = (await screen.findByText('Ragnok')).closest('tr') as HTMLElement
    expect(within(row).getByRole('img', { name: 'Skyborne (Windshaper)' })).toBeInTheDocument()
  })

  it('should show the level when the character has one', async () => {
    server.use(http.get('/api/roster', () => HttpResponse.json([{ ...ragnok, level: 58 }])))

    renderWithProviders(<Roster />)

    const row = (await screen.findByText('Ragnok')).closest('tr') as HTMLElement
    expect(within(row).getByRole('cell', { name: '58' })).toBeInTheDocument()
  })

  it('should leave the race and level cells empty when the character has neither', async () => {
    server.use(http.get('/api/roster', () => HttpResponse.json([ragnok])))

    renderWithProviders(<Roster />)

    const row = (await screen.findByText('Ragnok')).closest('tr') as HTMLElement
    const cells = within(row).getAllByRole('cell')
    expect(cells[0]).toBeEmptyDOMElement()
    expect(cells[3]).toBeEmptyDOMElement()
    expect(within(row).queryByRole('img')).not.toBeInTheDocument()
  })

  it('should order the columns identity first, then build, then organization when the roster loads', async () => {
    server.use(http.get('/api/roster', () => HttpResponse.json([ragnok])))

    renderWithProviders(<Roster />)

    await screen.findByText('Ragnok')
    expect(screen.getAllByRole('columnheader').map((header) => header.textContent)).toEqual([
      'Race',
      'Name',
      'Secondary name',
      'Level',
      'Class',
      'Spec',
      'Role',
      'Raid Team',
      'Main/Alt',
      'Professions',
    ])
  })
})

describe('Roster sorting and search', () => {
  const low: Character = { ...ragnok, id: 'low', name: 'Lowbie', level: 12 }
  const high: Character = { ...ragnok, id: 'high', name: 'Highbie', level: 60 }
  const mid: Character = { ...ragnok, id: 'mid', name: 'Midbie', level: 40 }

  function rowNames() {
    return screen
      .getAllByRole('row')
      .slice(1)
      .map((row) => within(row).getAllByRole('cell')[1].textContent)
  }

  it('should sort the table by level when the Level header is clicked', async () => {
    server.use(http.get('/api/roster', () => HttpResponse.json([high, low, mid])))
    const user = userEvent.setup()
    renderWithProviders(<Roster />)
    await screen.findByText('Highbie')

    await user.click(screen.getByRole('button', { name: 'Level' }))

    expect(rowNames()).toEqual(['Lowbie', 'Midbie', 'Highbie'])
    expect(screen.getByRole('columnheader', { name: /Level/ })).toHaveAttribute('aria-sort', 'ascending')
  })

  it('should sort descending when the Level header is clicked twice', async () => {
    server.use(http.get('/api/roster', () => HttpResponse.json([high, low, mid])))
    const user = userEvent.setup()
    renderWithProviders(<Roster />)
    await screen.findByText('Highbie')

    await user.click(screen.getByRole('button', { name: 'Level' }))
    await user.click(screen.getByRole('button', { name: /Level/ }))

    expect(rowNames()).toEqual(['Highbie', 'Midbie', 'Lowbie'])
  })

  it('should restore the default order when the Level header is clicked three times', async () => {
    server.use(http.get('/api/roster', () => HttpResponse.json([high, low, mid])))
    const user = userEvent.setup()
    renderWithProviders(<Roster />)
    await screen.findByText('Highbie')

    for (let click = 0; click < 3; click++) {
      await user.click(screen.getByRole('button', { name: /Level/ }))
    }

    expect(rowNames()).toEqual(['Highbie', 'Lowbie', 'Midbie'])
    expect(screen.getByRole('columnheader', { name: 'Level' })).toHaveAttribute('aria-sort', 'none')
  })

  it('should sort the table by race name when the Race header is clicked', async () => {
    const orc: Character = { ...ragnok, id: 'orc', name: 'Orcish', race: 'Orc' }
    const dwarf: Character = { ...ragnok, id: 'dwarf', name: 'Dwarfish', race: 'Dwarf' }
    server.use(http.get('/api/roster', () => HttpResponse.json([orc, dwarf])))
    const user = userEvent.setup()
    renderWithProviders(<Roster />)
    await screen.findByText('Orcish')

    await user.click(screen.getByRole('button', { name: 'Race' }))

    expect(rowNames()).toEqual(['Dwarfish', 'Orcish'])
  })

  it('should only list matching characters when a name is typed in the search box', async () => {
    server.use(http.get('/api/roster', () => HttpResponse.json([high, low, mid])))
    const user = userEvent.setup()
    renderWithProviders(<Roster />)
    await screen.findByText('Highbie')

    await user.type(screen.getByRole('searchbox', { name: 'Name' }), 'low')

    expect(rowNames()).toEqual(['Lowbie'])
  })

  it('should match the secondary name when it is typed in the search box', async () => {
    const twinked: Character = { ...ragnok, id: 'twink', name: 'Aeliana', secondaryName: 'Dawnsong' }
    server.use(http.get('/api/roster', () => HttpResponse.json([ragnok, twinked])))
    const user = userEvent.setup()
    renderWithProviders(<Roster />)
    await screen.findByText('Aeliana')

    await user.type(screen.getByRole('searchbox', { name: 'Name' }), 'dawn')

    expect(rowNames()).toEqual(['Aeliana'])
  })

  it('should narrow the search to the selected role when a role filter is also active', async () => {
    server.use(http.get('/api/roster', () => HttpResponse.json([ragnok, selene])))
    const user = userEvent.setup()
    renderWithProviders(<Roster />)
    await screen.findByText('Ragnok')

    await user.click(screen.getByRole('button', { name: 'Healer' }))
    await user.type(screen.getByRole('searchbox', { name: 'Name' }), 'r')

    expect(screen.getByText('No characters match.')).toBeInTheDocument()
  })
})

describe('Roster composition summary', () => {
  it('should show role, main and alt counts and the average level for the visible characters', async () => {
    const dps: Character = { ...selene, id: 'dps', name: 'Dpsguy', role: 'dps', level: 58 }
    server.use(
      http.get('/api/roster', () =>
        HttpResponse.json([{ ...ragnok, level: 60 }, { ...selene, level: 60 }, dps, { ...zaldrix, level: 50 }]),
      ),
    )
    const user = userEvent.setup()
    renderWithProviders(<Roster />)
    await screen.findByText('Ragnok')

    const summary = screen.getByRole('region', { name: 'Roster summary' })
    expect(within(summary).getByText('3 characters')).toBeInTheDocument()
    expect(within(summary).getByText(/1 Tank \| 1 Healer \| 1 DPS/)).toBeInTheDocument()
    expect(within(summary).getByText(/Main 3 \| 0 Alts/)).toBeInTheDocument()
    expect(within(summary).getByText(/Avg level 59/)).toBeInTheDocument()

    await user.click(screen.getByRole('checkbox', { name: 'Show alts' }))

    expect(within(summary).getByText('4 characters')).toBeInTheDocument()
    expect(within(summary).getByText(/Main 3 \| 1 Alt/)).toBeInTheDocument()
  })

  it('should follow the role filter when a role is selected', async () => {
    server.use(http.get('/api/roster', () => HttpResponse.json([ragnok, selene])))
    const user = userEvent.setup()
    renderWithProviders(<Roster />)
    await screen.findByText('Ragnok')

    await user.click(screen.getByRole('button', { name: 'Healer' }))

    const summary = screen.getByRole('region', { name: 'Roster summary' })
    expect(within(summary).getByText('1 character')).toBeInTheDocument()
    expect(within(summary).getByText(/0 Tanks \| 1 Healer/)).toBeInTheDocument()
  })

  it('should leave out the average level when no visible character has a level', async () => {
    server.use(http.get('/api/roster', () => HttpResponse.json([ragnok])))
    renderWithProviders(<Roster />)

    await screen.findByText('Ragnok')

    expect(screen.getByRole('region', { name: 'Roster summary' })).not.toHaveTextContent('Avg level')
  })
})
