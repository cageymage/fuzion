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

  it('should disable deleting a tier when it is the current raid', async () => {
    signInAs(officer)

    renderWithProviders(<OfficerRaidProgress />, '/officer/raid-progress')

    expect(await screen.findByRole('button', { name: 'Delete Molten Depths' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Delete Shattered Spire' })).toBeEnabled()
  })

  it('should not send a delete request when an officer clicks delete on a tier without confirming', async () => {
    signInAs(officer)
    let deleteRequested = false
    server.use(
      http.delete('/api/raid-tiers/:id', () => {
        deleteRequested = true
        return new HttpResponse(null, { status: 204 })
      }),
    )
    renderWithProviders(<OfficerRaidProgress />, '/officer/raid-progress')

    // given an officer who clicked delete on a retired tier
    await userEvent.click(await screen.findByRole('button', { name: 'Delete Shattered Spire' }))

    // when I cancel
    await userEvent.click(screen.getByRole('button', { name: 'Cancel deleting Shattered Spire' }))

    // then no delete is sent and the delete button is back
    expect(screen.getByRole('button', { name: 'Delete Shattered Spire' })).toBeInTheDocument()
    expect(deleteRequested).toBe(false)
  })

  it('should remove a tier from the list when an officer confirms deleting it', async () => {
    signInAs(officer)
    let tiers = [
      { id: 'tier-1', name: 'Molten Depths', isCurrent: true, sortOrder: 2, bosses: [] },
      { id: 'tier-2', name: 'Shattered Spire', isCurrent: false, sortOrder: 1, bosses: [] },
    ]
    let requestedPath = ''
    server.use(
      http.get('/api/raid-tiers', () => HttpResponse.json(tiers)),
      http.delete('/api/raid-tiers/:id', ({ params }) => {
        requestedPath = String(params.id)
        tiers = tiers.filter((tier) => tier.id !== params.id)
        return new HttpResponse(null, { status: 204 })
      }),
    )
    renderWithProviders(<OfficerRaidProgress />, '/officer/raid-progress')

    // given an officer who clicked delete on a retired tier
    await userEvent.click(await screen.findByRole('button', { name: 'Delete Shattered Spire' }))

    // when I confirm the delete
    await userEvent.click(screen.getByRole('button', { name: 'Confirm delete Shattered Spire' }))

    // then the tier is deleted and gone from the refetched list
    await waitFor(() =>
      expect(screen.queryByRole('heading', { name: 'Shattered Spire' })).not.toBeInTheDocument(),
    )
    expect(requestedPath).toBe('tier-2')
  })

  it('should show the API error message when deleting a tier is rejected', async () => {
    signInAs(officer)
    server.use(
      http.delete('/api/raid-tiers/:id', () =>
        HttpResponse.json(
          { error: 'raid tier is current; clear its current flag before deleting it' },
          { status: 409 },
        ),
      ),
    )
    renderWithProviders(<OfficerRaidProgress />, '/officer/raid-progress')
    await userEvent.click(await screen.findByRole('button', { name: 'Delete Shattered Spire' }))

    await userEvent.click(screen.getByRole('button', { name: 'Confirm delete Shattered Spire' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'raid tier is current; clear its current flag before deleting it',
    )
  })

  it('should remove a boss from its tier when an officer confirms deleting it', async () => {
    signInAs(officer)
    let bosses = [
      { id: 'boss-1', name: 'Grimjaw', killedAt: null },
      { id: 'boss-2', name: 'Ashveil', killedAt: null },
    ]
    let requestedPath = ''
    server.use(
      http.get('/api/raid-tiers', () =>
        HttpResponse.json([{ id: 'tier-1', name: 'Molten Depths', isCurrent: true, sortOrder: 2, bosses }]),
      ),
      http.delete('/api/raid-bosses/:id', ({ params }) => {
        requestedPath = String(params.id)
        bosses = bosses.filter((boss) => boss.id !== params.id)
        return new HttpResponse(null, { status: 204 })
      }),
    )
    renderWithProviders(<OfficerRaidProgress />, '/officer/raid-progress')

    // given an officer who clicked delete on a boss
    await userEvent.click(await screen.findByRole('button', { name: 'Delete Grimjaw' }))

    // when I confirm the delete
    await userEvent.click(screen.getByRole('button', { name: 'Confirm delete Grimjaw' }))

    // then the boss is deleted and gone from the refetched tier
    await waitFor(() =>
      expect(screen.queryByRole('checkbox', { name: 'Grimjaw killed' })).not.toBeInTheDocument(),
    )
    expect(screen.getByRole('checkbox', { name: 'Ashveil killed' })).toBeInTheDocument()
    expect(requestedPath).toBe('boss-1')
  })

  it('should show the API error message when deleting a boss is rejected', async () => {
    signInAs(officer)
    server.use(
      http.delete('/api/raid-bosses/:id', () =>
        HttpResponse.json({ error: 'raid boss not found' }, { status: 404 }),
      ),
    )
    renderWithProviders(<OfficerRaidProgress />, '/officer/raid-progress')
    await userEvent.click(await screen.findByRole('button', { name: 'Delete Grimjaw' }))

    await userEvent.click(screen.getByRole('button', { name: 'Confirm delete Grimjaw' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('raid boss not found')
  })

  it('should show the new tier name after an officer renames a tier and the tiers refetch', async () => {
    signInAs(officer)
    let tierName = 'Molten Depthz'
    let requestBody: unknown
    let requestedPath = ''
    server.use(
      http.get('/api/raid-tiers', () =>
        HttpResponse.json([{ id: 'tier-1', name: tierName, isCurrent: true, sortOrder: 2, bosses: [] }]),
      ),
      http.patch('/api/raid-tiers/:id', async ({ request, params }) => {
        requestBody = await request.json()
        requestedPath = String(params.id)
        tierName = 'Molten Depths'
        return HttpResponse.json({ id: 'tier-1', name: tierName, isCurrent: true, sortOrder: 2, bosses: [] })
      }),
    )
    renderWithProviders(<OfficerRaidProgress />, '/officer/raid-progress')

    // given an officer who opened the rename field and typed a corrected name
    await userEvent.click(await screen.findByRole('button', { name: 'Rename Molten Depthz' }))
    const nameInput = screen.getByRole('textbox', { name: 'New name for Molten Depthz' })
    await userEvent.clear(nameInput)
    await userEvent.type(nameInput, 'Molten Depths')

    // when I save the name
    await userEvent.click(screen.getByRole('button', { name: 'Save name for Molten Depthz' }))

    // then the API receives the new name and the refetched heading shows it
    expect(await screen.findByRole('heading', { name: 'Molten Depths' })).toBeInTheDocument()
    expect(requestBody).toEqual({ name: 'Molten Depths' })
    expect(requestedPath).toBe('tier-1')
    expect(screen.queryByRole('textbox', { name: 'New name for Molten Depthz' })).not.toBeInTheDocument()
  })

  it('should not send a rename request when an officer cancels renaming a tier', async () => {
    signInAs(officer)
    let renameRequested = false
    server.use(
      http.patch('/api/raid-tiers/:id', () => {
        renameRequested = true
        return HttpResponse.json({})
      }),
    )
    renderWithProviders(<OfficerRaidProgress />, '/officer/raid-progress')
    await userEvent.click(await screen.findByRole('button', { name: 'Rename Shattered Spire' }))
    await userEvent.type(screen.getByRole('textbox', { name: 'New name for Shattered Spire' }), ' Two')

    await userEvent.click(screen.getByRole('button', { name: 'Cancel renaming Shattered Spire' }))

    expect(screen.getByRole('button', { name: 'Rename Shattered Spire' })).toBeInTheDocument()
    expect(renameRequested).toBe(false)
  })

  it('should keep the rename field open with the API error message when renaming a tier is rejected', async () => {
    signInAs(officer)
    server.use(
      http.patch('/api/raid-tiers/:id', () =>
        HttpResponse.json({ error: 'name: must not be empty' }, { status: 400 }),
      ),
    )
    renderWithProviders(<OfficerRaidProgress />, '/officer/raid-progress')
    await userEvent.click(await screen.findByRole('button', { name: 'Rename Shattered Spire' }))
    await userEvent.clear(screen.getByRole('textbox', { name: 'New name for Shattered Spire' }))

    await userEvent.click(screen.getByRole('button', { name: 'Save name for Shattered Spire' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('name: must not be empty')
    expect(screen.getByRole('textbox', { name: 'New name for Shattered Spire' })).toBeInTheDocument()
  })

  it('should send every tier id in the new order when an officer moves a tier down', async () => {
    signInAs(officer)
    let requestBody: unknown
    server.use(
      http.put('/api/raid-tiers/order', async ({ request }) => {
        requestBody = await request.json()
        return new HttpResponse(null, { status: 204 })
      }),
    )
    renderWithProviders(<OfficerRaidProgress />, '/officer/raid-progress')

    await userEvent.click(await screen.findByRole('button', { name: 'Move Molten Depths down' }))

    await waitFor(() => expect(requestBody).toEqual({ ids: ['tier-2', 'tier-1'] }))
  })

  it('should disable moving the first tier up and the last tier down', async () => {
    signInAs(officer)

    renderWithProviders(<OfficerRaidProgress />, '/officer/raid-progress')

    expect(await screen.findByRole('button', { name: 'Move Molten Depths up' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Move Molten Depths down' })).toBeEnabled()
    expect(screen.getByRole('button', { name: 'Move Shattered Spire up' })).toBeEnabled()
    expect(screen.getByRole('button', { name: 'Move Shattered Spire down' })).toBeDisabled()
  })

  it('should show the API error message when reordering tiers is rejected', async () => {
    signInAs(officer)
    server.use(
      http.put('/api/raid-tiers/order', () =>
        HttpResponse.json({ error: 'ids: must list every raid exactly once' }, { status: 400 }),
      ),
    )
    renderWithProviders(<OfficerRaidProgress />, '/officer/raid-progress')

    await userEvent.click(await screen.findByRole('button', { name: 'Move Shattered Spire up' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('ids: must list every raid exactly once')
  })

  it('should show a no-bosses message when a tier has no bosses', async () => {
    signInAs(officer)
    server.use(
      http.get('/api/raid-tiers', () =>
        HttpResponse.json([{ id: 'tier-1', name: 'Molten Depths', isCurrent: false, sortOrder: 1, bosses: [] }]),
      ),
    )

    renderWithProviders(<OfficerRaidProgress />, '/officer/raid-progress')

    expect(await screen.findByText('No bosses yet.')).toBeInTheDocument()
  })

  it('should show the added boss after an officer adds a boss to a tier and the tiers refetch', async () => {
    signInAs(officer)
    const bosses = [{ id: 'boss-1', name: 'Grimjaw', killedAt: null }]
    let requestBody: unknown
    let requestedPath = ''
    server.use(
      http.get('/api/raid-tiers', () =>
        HttpResponse.json([{ id: 'tier-1', name: 'Molten Depths', isCurrent: true, sortOrder: 2, bosses }]),
      ),
      http.post('/api/raid-tiers/:id/bosses', async ({ request, params }) => {
        requestBody = await request.json()
        requestedPath = String(params.id)
        const added = { id: 'boss-2', name: 'Pyrelord', killedAt: null }
        bosses.push(added)
        return HttpResponse.json(added, { status: 201 })
      }),
    )
    renderWithProviders(<OfficerRaidProgress />, '/officer/raid-progress')

    // given an officer who typed a new boss name for the tier
    await userEvent.type(await screen.findByRole('textbox', { name: 'New boss for Molten Depths' }), 'Pyrelord')

    // when I add the boss
    await userEvent.click(screen.getByRole('button', { name: 'Add boss to Molten Depths' }))

    // then the API receives the name, the refetched tier lists the boss, and the field clears
    expect(await screen.findByRole('checkbox', { name: 'Pyrelord killed' })).toBeInTheDocument()
    expect(requestBody).toEqual({ name: 'Pyrelord' })
    expect(requestedPath).toBe('tier-1')
    expect(screen.getByRole('textbox', { name: 'New boss for Molten Depths' })).toHaveValue('')
  })

  it('should show the API error message when adding a boss is rejected', async () => {
    signInAs(officer)
    server.use(
      http.post('/api/raid-tiers/:id/bosses', () =>
        HttpResponse.json({ error: 'name: must not be empty' }, { status: 400 }),
      ),
    )
    renderWithProviders(<OfficerRaidProgress />, '/officer/raid-progress')

    await userEvent.click(await screen.findByRole('button', { name: 'Add boss to Molten Depths' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('name: must not be empty')
  })

  it('should send the new name to the boss endpoint when an officer renames a boss', async () => {
    signInAs(officer)
    let requestBody: unknown
    let requestedPath = ''
    server.use(
      http.patch('/api/raid-bosses/:id', async ({ request, params }) => {
        requestBody = await request.json()
        requestedPath = String(params.id)
        return HttpResponse.json({ id: 'boss-2', name: 'Ashveil the Burning', killedAt: null })
      }),
    )
    renderWithProviders(<OfficerRaidProgress />, '/officer/raid-progress')
    await userEvent.click(await screen.findByRole('button', { name: 'Rename Ashveil' }))
    await userEvent.type(screen.getByRole('textbox', { name: 'New name for Ashveil' }), ' the Burning')

    await userEvent.click(screen.getByRole('button', { name: 'Save name for Ashveil' }))

    await waitFor(() => expect(requestBody).toEqual({ name: 'Ashveil the Burning' }))
    expect(requestedPath).toBe('boss-2')
  })

  it('should send every boss id of the tier in the new order when an officer moves a boss up', async () => {
    signInAs(officer)
    let requestBody: unknown
    let requestedPath = ''
    server.use(
      http.put('/api/raid-tiers/:id/bosses/order', async ({ request, params }) => {
        requestBody = await request.json()
        requestedPath = String(params.id)
        return new HttpResponse(null, { status: 204 })
      }),
    )
    renderWithProviders(<OfficerRaidProgress />, '/officer/raid-progress')

    await userEvent.click(await screen.findByRole('button', { name: 'Move Pyrelord up' }))

    await waitFor(() => expect(requestBody).toEqual({ ids: ['boss-1', 'boss-3', 'boss-2'] }))
    expect(requestedPath).toBe('tier-1')
  })

  it('should show the API error message when reordering bosses is rejected', async () => {
    signInAs(officer)
    server.use(
      http.put('/api/raid-tiers/:id/bosses/order', () =>
        HttpResponse.json({ error: 'ids: must list every boss in the raid exactly once' }, { status: 400 }),
      ),
    )
    renderWithProviders(<OfficerRaidProgress />, '/officer/raid-progress')

    await userEvent.click(await screen.findByRole('button', { name: 'Move Grimjaw down' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'ids: must list every boss in the raid exactly once',
    )
  })
})
