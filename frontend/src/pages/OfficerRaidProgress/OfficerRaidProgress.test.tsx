import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '../../mocks/server'
import { renderWithProviders } from '../../testUtils'
import { OfficerRaidProgress } from './OfficerRaidProgress'

const officer = { id: 'user-1', username: 'Officer', avatarUrl: null, isOfficer: true }
const member = { id: 'user-2', username: 'Member', avatarUrl: null, isOfficer: false }

function signInAs(user: typeof officer) {
  server.use(http.get('/api/auth/me', () => HttpResponse.json(user)))
}

describe('OfficerRaidProgress', () => {
  it('should show an officers-only message when the visitor is not an officer', async () => {
    signInAs(member)

    renderWithProviders(<OfficerRaidProgress />, '/officer/raid-progress')

    expect(await screen.findByText('Only officers can manage raid progress.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Create raid' })).not.toBeInTheDocument()
  })

  it('should show a login link when the visitor is anonymous', async () => {
    renderWithProviders(<OfficerRaidProgress />, '/officer/raid-progress')

    expect(await screen.findByText('Only officers can manage raid progress.')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Log in with Discord' })).toHaveAttribute(
      'href',
      'http://localhost:3000/api/auth/login',
    )
    expect(screen.queryByRole('button', { name: 'Create raid' })).not.toBeInTheDocument()
  })

  it('should list every tier with its current flag and boss kills when the visitor is an officer', async () => {
    signInAs(officer)

    renderWithProviders(<OfficerRaidProgress />, '/officer/raid-progress')

    expect(await screen.findByRole('heading', { name: 'Molten Depths' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Shattered Spire' })).toBeInTheDocument()
    expect(screen.getByRole('checkbox', { name: 'Molten Depths is current' })).toBeChecked()
    expect(screen.getByRole('checkbox', { name: 'Shattered Spire is current' })).not.toBeChecked()
    expect(screen.getByRole('checkbox', { name: 'Grimjaw killed' })).toBeChecked()
    expect(screen.getByRole('checkbox', { name: 'Ashveil killed' })).not.toBeChecked()
  })

  it('should show an error message when the tiers cannot be loaded', async () => {
    signInAs(officer)
    server.use(http.get('/api/raid-tiers', () => HttpResponse.json({}, { status: 500 })))

    renderWithProviders(<OfficerRaidProgress />, '/officer/raid-progress')

    expect(await screen.findByText('Raids could not be loaded.')).toBeInTheDocument()
  })

  it('should create a tier with the boss names in order and the next sort order when an officer submits the form', async () => {
    signInAs(officer)
    let requestBody: unknown
    server.use(
      http.post('/api/raid-tiers', async ({ request }) => {
        requestBody = await request.json()
        return HttpResponse.json(
          { id: 'tier-3', name: 'Nightfall Citadel', isCurrent: false, sortOrder: 3, bosses: [] },
          { status: 201 },
        )
      }),
    )
    renderWithProviders(<OfficerRaidProgress />, '/officer/raid-progress')

    // given an officer who has filled in a tier name and two boss names
    await userEvent.type(await screen.findByLabelText('Raid name'), 'Nightfall Citadel')
    await userEvent.type(screen.getByLabelText('Boss 1 name'), 'Vexmar')
    await userEvent.click(screen.getByRole('button', { name: 'Add boss' }))
    await userEvent.type(screen.getByLabelText('Boss 2 name'), 'Sereth')

    // when I create the tier
    await userEvent.click(screen.getByRole('button', { name: 'Create raid' }))

    // then the API receives the tier after the highest existing sort order
    await waitFor(() =>
      expect(requestBody).toEqual({
        name: 'Nightfall Citadel',
        sortOrder: 3,
        bosses: ['Vexmar', 'Sereth'],
      }),
    )
  })

  it('should send the boss names in the moved order when an officer moves a boss up before submitting', async () => {
    signInAs(officer)
    let requestBody: unknown
    server.use(
      http.post('/api/raid-tiers', async ({ request }) => {
        requestBody = await request.json()
        return HttpResponse.json(
          { id: 'tier-3', name: 'Nightfall Citadel', isCurrent: false, sortOrder: 3, bosses: [] },
          { status: 201 },
        )
      }),
    )
    renderWithProviders(<OfficerRaidProgress />, '/officer/raid-progress')
    await userEvent.type(await screen.findByLabelText('Raid name'), 'Nightfall Citadel')
    await userEvent.type(screen.getByLabelText('Boss 1 name'), 'Vexmar')
    await userEvent.click(screen.getByRole('button', { name: 'Add boss' }))
    await userEvent.type(screen.getByLabelText('Boss 2 name'), 'Sereth')

    // when I move the second boss up
    await userEvent.click(screen.getByRole('button', { name: 'Move boss 2 up' }))
    await userEvent.click(screen.getByRole('button', { name: 'Create raid' }))

    // then the API receives the bosses in the new order
    await waitFor(() =>
      expect(requestBody).toEqual({
        name: 'Nightfall Citadel',
        sortOrder: 3,
        bosses: ['Sereth', 'Vexmar'],
      }),
    )
  })

  it('should leave out blank boss rows when an officer submits the form', async () => {
    signInAs(officer)
    let requestBody: unknown
    server.use(
      http.post('/api/raid-tiers', async ({ request }) => {
        requestBody = await request.json()
        return HttpResponse.json(
          { id: 'tier-3', name: 'Nightfall Citadel', isCurrent: false, sortOrder: 3, bosses: [] },
          { status: 201 },
        )
      }),
    )
    renderWithProviders(<OfficerRaidProgress />, '/officer/raid-progress')
    await userEvent.type(await screen.findByLabelText('Raid name'), 'Nightfall Citadel')
    await userEvent.type(screen.getByLabelText('Boss 1 name'), 'Vexmar')
    await userEvent.click(screen.getByRole('button', { name: 'Add boss' }))

    await userEvent.click(screen.getByRole('button', { name: 'Create raid' }))

    await waitFor(() =>
      expect(requestBody).toEqual({ name: 'Nightfall Citadel', sortOrder: 3, bosses: ['Vexmar'] }),
    )
  })

  it('should clear the form when a tier is created', async () => {
    signInAs(officer)
    renderWithProviders(<OfficerRaidProgress />, '/officer/raid-progress')
    await userEvent.type(await screen.findByLabelText('Raid name'), 'Nightfall Citadel')
    await userEvent.type(screen.getByLabelText('Boss 1 name'), 'Vexmar')

    await userEvent.click(screen.getByRole('button', { name: 'Create raid' }))

    await waitFor(() => expect(screen.getByLabelText('Raid name')).toHaveValue(''))
    expect(screen.getByLabelText('Boss 1 name')).toHaveValue('')
  })

  it('should show the API error message when creating a tier is rejected', async () => {
    signInAs(officer)
    server.use(
      http.post('/api/raid-tiers', () =>
        HttpResponse.json({ error: 'name: must not be empty' }, { status: 400 }),
      ),
    )
    renderWithProviders(<OfficerRaidProgress />, '/officer/raid-progress')

    await userEvent.click(await screen.findByRole('button', { name: 'Create raid' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('name: must not be empty')
  })

  it('should mark a tier current when an officer toggles its current flag', async () => {
    signInAs(officer)
    let requestBody: unknown
    let requestedPath = ''
    server.use(
      http.patch('/api/raid-tiers/:id', async ({ request, params }) => {
        requestBody = await request.json()
        requestedPath = String(params.id)
        return HttpResponse.json({
          id: 'tier-2',
          name: 'Shattered Spire',
          isCurrent: true,
          sortOrder: 1,
          bosses: [],
        })
      }),
    )
    renderWithProviders(<OfficerRaidProgress />, '/officer/raid-progress')

    await userEvent.click(await screen.findByRole('checkbox', { name: 'Shattered Spire is current' }))

    await waitFor(() => expect(requestBody).toEqual({ isCurrent: true }))
    expect(requestedPath).toBe('tier-2')
  })

  it('should show a boss as killed after an officer marks it killed and the tiers refetch', async () => {
    signInAs(officer)
    let ashveilKilledAt: string | null = null
    server.use(
      http.get('/api/raid-tiers', () =>
        HttpResponse.json([
          {
            id: 'tier-1',
            name: 'Molten Depths',
            isCurrent: true,
            sortOrder: 2,
            bosses: [{ id: 'boss-2', name: 'Ashveil', killedAt: ashveilKilledAt }],
          },
        ]),
      ),
      http.patch('/api/raid-bosses/:id', async ({ request }) => {
        const body = (await request.json()) as { killed: boolean }
        ashveilKilledAt = body.killed ? '2026-03-01T20:00:00Z' : null
        return HttpResponse.json({ id: 'boss-2', name: 'Ashveil', killedAt: ashveilKilledAt })
      }),
    )
    renderWithProviders(<OfficerRaidProgress />, '/officer/raid-progress')

    // when I mark the boss killed
    await userEvent.click(await screen.findByRole('checkbox', { name: 'Ashveil killed' }))

    // then the refetched tier list shows it killed
    await waitFor(() => expect(screen.getByRole('checkbox', { name: 'Ashveil killed' })).toBeChecked())
  })

  it('should send the killed flag to the boss endpoint when an officer toggles a killed boss off', async () => {
    signInAs(officer)
    let requestBody: unknown
    let requestedPath = ''
    server.use(
      http.patch('/api/raid-bosses/:id', async ({ request, params }) => {
        requestBody = await request.json()
        requestedPath = String(params.id)
        return HttpResponse.json({ id: 'boss-1', name: 'Grimjaw', killedAt: null })
      }),
    )
    renderWithProviders(<OfficerRaidProgress />, '/officer/raid-progress')

    await userEvent.click(await screen.findByRole('checkbox', { name: 'Grimjaw killed' }))

    await waitFor(() => expect(requestBody).toEqual({ killed: false }))
    expect(requestedPath).toBe('boss-1')
  })

  it('should show an error message when a toggle is rejected', async () => {
    signInAs(officer)
    server.use(
      http.patch('/api/raid-bosses/:id', () =>
        HttpResponse.json({ error: 'raid boss not found' }, { status: 404 }),
      ),
    )
    renderWithProviders(<OfficerRaidProgress />, '/officer/raid-progress')

    await userEvent.click(await screen.findByRole('checkbox', { name: 'Ashveil killed' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('raid boss not found')
  })
})
