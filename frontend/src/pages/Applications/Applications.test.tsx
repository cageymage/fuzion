import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { delay, http, HttpResponse } from 'msw'
import { beforeEach, describe, expect, it } from 'vitest'
import { FakeTurnstile } from '../../mocks/fakeTurnstile'
import { server } from '../../mocks/server'
import { renderWithProviders } from '../../testUtils'
import { Applications } from './Applications'

const turnstile = new FakeTurnstile()

beforeEach(() => {
  turnstile.params = null
  turnstile.resets = 0
  turnstile.install()
})

async function fillEveryField(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText(/your name/i), 'Terry')
  await user.type(screen.getByLabelText(/character name/i), 'Ragnok')
  await user.selectOptions(screen.getByLabelText(/class/i), 'Warrior')
  await user.click(screen.getByRole('radio', { name: 'Tank' }))
  await user.type(screen.getByLabelText(/availability/i), 'Tue/Thu 8-11 EST')
  await user.type(screen.getByLabelText(/discord handle/i), 'terry#1234')
  await user.type(screen.getByRole('textbox', { name: /notes/i }), 'Looking for a raid home')
  await turnstile.solve('token-1')
}

describe('Applications', () => {
  it('should link to the officer review page when the visitor is an officer', async () => {
    server.use(
      http.get('/api/auth/me', () =>
        HttpResponse.json({ id: 'user-1', username: 'Officer', avatarUrl: null, isOfficer: true }),
      ),
    )

    renderWithProviders(<Applications />)

    expect(await screen.findByRole('link', { name: 'Manage applications' })).toHaveAttribute(
      'href',
      '/officer/applications',
    )
  })

  it('should not link to the officer review page when the visitor is not an officer', async () => {
    server.use(
      http.get('/api/auth/me', () =>
        HttpResponse.json({ id: 'user-2', username: 'Member', avatarUrl: null, isOfficer: false }),
      ),
    )

    renderWithProviders(<Applications />)

    await screen.findByRole('button', { name: /submit/i })
    expect(screen.queryByRole('link', { name: 'Manage applications' })).not.toBeInTheDocument()
  })

  it('should show a notice linking to the legal page when the form is shown', () => {
    renderWithProviders(<Applications />)

    expect(screen.getByRole('link', { name: 'Legal page' })).toHaveAttribute('href', '/legal')
    expect(screen.getByText(/by applying you agree/i)).toBeInTheDocument()
  })

  it('should show a thank-you message when the application is submitted successfully', async () => {
    const user = userEvent.setup()
    renderWithProviders(<Applications />)

    await fillEveryField(user)
    await user.click(screen.getByRole('button', { name: 'Submit application' }))

    expect(await screen.findByText(/thank/i)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Submit application' })).not.toBeInTheDocument()
  })

  it('should post every field and the Turnstile token to the API when the form is submitted', async () => {
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
      turnstileToken: 'token-1',
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
    await turnstile.solve('token-1')
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

  it('should render the Turnstile widget with the configured site key', async () => {
    renderWithProviders(<Applications />)

    await turnstile.waitForWidget()

    expect(turnstile.params?.sitekey).toBe('test-site-key')
  })

  it('should keep the submit button disabled when the Turnstile challenge has not been passed', async () => {
    renderWithProviders(<Applications />)

    await turnstile.waitForWidget()

    expect(screen.getByRole('button', { name: 'Submit application' })).toBeDisabled()
  })

  it('should enable the submit button when the Turnstile challenge is passed', async () => {
    renderWithProviders(<Applications />)

    await turnstile.solve('token-1')

    expect(screen.getByRole('button', { name: 'Submit application' })).toBeEnabled()
  })

  it('should disable the submit button when the Turnstile token expires', async () => {
    renderWithProviders(<Applications />)
    await turnstile.solve('token-1')

    await turnstile.expire()

    expect(screen.getByRole('button', { name: 'Submit application' })).toBeDisabled()
  })

  it('should reset the widget and disable submit when the API rejects the application', async () => {
    server.use(
      http.post('/api/applications', () =>
        HttpResponse.json({ error: 'turnstileToken: verification failed' }, { status: 400 }),
      ),
    )
    const user = userEvent.setup()
    renderWithProviders(<Applications />)
    await fillEveryField(user)

    await user.click(screen.getByRole('button', { name: 'Submit application' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('turnstileToken: verification failed')
    expect(turnstile.resets).toBe(1)
    expect(screen.getByRole('button', { name: 'Submit application' })).toBeDisabled()
  })

  it('should send the fresh token when the applicant resubmits after a failed submit', async () => {
    const received: unknown[] = []
    server.use(
      http.post('/api/applications', async ({ request }) => {
        received.push(await request.json())
        return received.length === 1
          ? HttpResponse.json({ error: 'application could not be submitted' }, { status: 500 })
          : HttpResponse.json({ id: 'app-1' }, { status: 201 })
      }),
    )
    const user = userEvent.setup()
    renderWithProviders(<Applications />)
    await fillEveryField(user)
    await user.click(screen.getByRole('button', { name: 'Submit application' }))
    await screen.findByRole('alert')
    await turnstile.solve('token-2')

    await user.click(screen.getByRole('button', { name: 'Submit application' }))

    await screen.findByText(/thank/i)
    expect(received.map((body) => (body as { turnstileToken: string }).turnstileToken)).toEqual([
      'token-1',
      'token-2',
    ])
  })
})
