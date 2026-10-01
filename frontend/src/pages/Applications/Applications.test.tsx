import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { delay, http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '../../mocks/server'
import { renderWithProviders } from '../../testUtils'
import { Applications } from './Applications'

async function fillEveryField(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText(/your name/i), 'Terry')
  await user.type(screen.getByLabelText(/character name/i), 'Ragnok')
  await user.selectOptions(screen.getByLabelText(/class/i), 'Warrior')
  await user.click(screen.getByRole('radio', { name: 'Tank' }))
  await user.type(screen.getByLabelText(/availability/i), 'Tue/Thu 8-11 EST')
  await user.type(screen.getByLabelText(/discord handle/i), 'terry#1234')
  await user.type(screen.getByRole('textbox', { name: /notes/i }), 'Looking for a raid home')
}

describe('Applications', () => {
  it('should show a thank-you message when the application is submitted successfully', async () => {
    const user = userEvent.setup()
    renderWithProviders(<Applications />)

    await fillEveryField(user)
    await user.click(screen.getByRole('button', { name: 'Submit application' }))

    expect(await screen.findByText(/thank/i)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Submit application' })).not.toBeInTheDocument()
  })

  it('should post every field to the API when the form is submitted', async () => {
    let received: unknown
    server.use(
      http.post('/api/applications', async ({ request }) => {
        received = await request.json()
        return HttpResponse.json({ id: 'app-1' }, { status: 201 })
      }),
    )
    const user = userEvent.setup()
    renderWithProviders(<Applications />)

    await fillEveryField(user)
    await user.click(screen.getByRole('button', { name: 'Submit application' }))
    await screen.findByText(/thank/i)

    expect(received).toEqual({
      applicantName: 'Terry',
      characterName: 'Ragnok',
      class: 'Warrior',
      role: 'tank',
      availability: 'Tue/Thu 8-11 EST',
      discordHandle: 'terry#1234',
      notes: 'Looking for a raid home',
    })
  })

  it('should show the server error message when the API rejects the application', async () => {
    server.use(
      http.post('/api/applications', () =>
        HttpResponse.json({ error: 'role: must be tank, healer or dps' }, { status: 400 }),
      ),
    )
    const user = userEvent.setup()
    renderWithProviders(<Applications />)

    await fillEveryField(user)
    await user.click(screen.getByRole('button', { name: 'Submit application' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('role: must be tank, healer or dps')
  })

  it('should show a generic error message when the server fails', async () => {
    server.use(
      http.post('/api/applications', () =>
        HttpResponse.json({ error: 'application could not be submitted' }, { status: 500 }),
      ),
    )
    const user = userEvent.setup()
    renderWithProviders(<Applications />)

    await fillEveryField(user)
    await user.click(screen.getByRole('button', { name: 'Submit application' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Something went wrong, try again or ping an officer on Discord',
    )
  })

  it('should show a generic error message when the network request fails', async () => {
    server.use(http.post('/api/applications', () => HttpResponse.error()))
    const user = userEvent.setup()
    renderWithProviders(<Applications />)

    await fillEveryField(user)
    await user.click(screen.getByRole('button', { name: 'Submit application' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Something went wrong, try again or ping an officer on Discord',
    )
  })

  it('should not submit when a required field is empty', async () => {
    let submitted = false
    server.use(
      http.post('/api/applications', () => {
        submitted = true
        return HttpResponse.json({ id: 'app-1' }, { status: 201 })
      }),
    )
    const user = userEvent.setup()
    renderWithProviders(<Applications />)

    await user.type(screen.getByLabelText(/your name/i), 'Terry')
    await user.click(screen.getByRole('button', { name: 'Submit application' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(/character name/i)
    expect(submitted).toBe(false)
    expect(screen.getByRole('button', { name: 'Submit application' })).toBeEnabled()
  })

  it('should disable the submit button while the request is in flight', async () => {
    server.use(
      http.post('/api/applications', async () => {
        await delay(100)
        return HttpResponse.json({ id: 'app-1' }, { status: 201 })
      }),
    )
    const user = userEvent.setup()
    renderWithProviders(<Applications />)

    await fillEveryField(user)
    await user.click(screen.getByRole('button', { name: 'Submit application' }))

    expect(screen.getByRole('button', { name: 'Sending…' })).toBeDisabled()
    await screen.findByText(/thank/i)
  })

  it('should update the notes counter as the applicant types', async () => {
    const user = userEvent.setup()
    renderWithProviders(<Applications />)

    await user.type(screen.getByRole('textbox', { name: /notes/i }), 'hello')

    expect(screen.getByText('5 / 2000')).toBeInTheDocument()
  })
})
