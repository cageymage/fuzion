import { fireEvent, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { Route, Routes } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { server } from '../../mocks/server'
import { renderWithProviders } from '../../testUtils'
import { OfficerSurveyNew } from './OfficerSurveyNew'

const officer = { id: 'user-1', username: 'Officer', avatarUrl: null, isOfficer: true }
const member = { id: 'user-2', username: 'Member', avatarUrl: null, isOfficer: false }

function signInAs(user: typeof officer) {
  server.use(http.get('/api/auth/me', () => HttpResponse.json(user)))
}

function renderPage() {
  return renderWithProviders(
    <Routes>
      <Route path="/officer/surveys/new" element={<OfficerSurveyNew />} />
      <Route path="/officer/surveys/:id" element={<p>Manage page</p>} />
      <Route path="/officer/surveys" element={<p>Survey list</p>} />
      <Route path="/" element={<p>Home page</p>} />
    </Routes>,
    '/officer/surveys/new',
  )
}

const closesAtLocal = '2026-03-20T20:00'

function fillBasics() {
  fireEvent.change(screen.getByLabelText('Title'), { target: { value: 'Spring roster' } })
  fireEvent.change(screen.getByLabelText('Description'), { target: { value: 'Help us plan' } })
  fireEvent.change(screen.getByLabelText('Closes at'), { target: { value: closesAtLocal } })
}

describe('OfficerSurveyNew', () => {
  it('should send the survey with presets and custom options and open its manage page when the officer creates it', async () => {
    const user = userEvent.setup()
    let created: unknown = null
    server.use(
      http.post('/api/surveys', async ({ request }) => {
        created = await request.json()
        return HttpResponse.json({ id: 'survey-9' }, { status: 201 })
      }),
    )
    signInAs(officer)
    renderPage()
    await screen.findByRole('heading', { name: 'New Survey' })

    fillBasics()
    await user.type(screen.getByLabelText('Prompt for question 1'), 'Class')
    await user.selectOptions(screen.getByLabelText('Options for question 1'), 'Classic classes')
    await user.click(screen.getByRole('button', { name: 'Add question' }))
    await user.type(screen.getByLabelText('Prompt for question 2'), 'Raid nights')
    await user.selectOptions(screen.getByLabelText('Type of question 2'), 'Multiple choice')
    await user.type(screen.getByLabelText('Custom options for question 2, one per line'), 'Tuesday{enter}Thursday')
    await user.click(screen.getByRole('checkbox', { name: 'Question 2 is required' }))
    await user.click(screen.getByRole('button', { name: 'Create survey' }))

    expect(await screen.findByText('Manage page')).toBeInTheDocument()
    expect(created).toEqual({
      title: 'Spring roster',
      description: 'Help us plan',
      closesAt: new Date(closesAtLocal).toISOString(),
      questions: [
        { prompt: 'Class', type: 'single', required: true, preset: 'wow-classes' },
        { prompt: 'Raid nights', type: 'multi', required: false, options: ['Tuesday', 'Thursday'] },
      ],
    })
  })

  it('should ask for a closing time and send nothing when the deadline is empty', async () => {
    const user = userEvent.setup()
    let requests = 0
    server.use(
      http.post('/api/surveys', () => {
        requests += 1
        return HttpResponse.json({ id: 'survey-9' }, { status: 201 })
      }),
    )
    signInAs(officer)
    renderPage()
    await screen.findByRole('heading', { name: 'New Survey' })

    await user.type(screen.getByLabelText('Title'), 'Spring roster')
    await user.click(screen.getByRole('button', { name: 'Create survey' }))

    expect(screen.getByRole('alert')).toHaveTextContent('Pick when the survey closes')
    expect(requests).toBe(0)
  })

  it("should show the server's message when the server rejects the survey", async () => {
    const user = userEvent.setup()
    server.use(
      http.post('/api/surveys', () =>
        HttpResponse.json({ error: 'questions[0].options: at least two options are required' }, { status: 400 }),
      ),
    )
    signInAs(officer)
    renderPage()
    await screen.findByRole('heading', { name: 'New Survey' })

    fillBasics()
    await user.type(screen.getByLabelText('Prompt for question 1'), 'Class')
    await user.click(screen.getByRole('button', { name: 'Create survey' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'questions[0].options: at least two options are required',
    )
  })

  it('should keep the officer on the form when the server rejects the survey', async () => {
    const user = userEvent.setup()
    server.use(
      http.post('/api/surveys', () => HttpResponse.json({ error: 'title: is required' }, { status: 400 })),
    )
    signInAs(officer)
    renderPage()
    await screen.findByRole('heading', { name: 'New Survey' })

    fillBasics()
    await user.click(screen.getByRole('button', { name: 'Create survey' }))

    await screen.findByRole('alert')
    expect(screen.queryByText('Manage page')).not.toBeInTheDocument()
    expect(screen.getByLabelText('Title')).toHaveValue('Spring roster')
  })

  it('should remove a question when the officer removes one of two', async () => {
    const user = userEvent.setup()
    signInAs(officer)
    renderPage()
    await screen.findByRole('heading', { name: 'New Survey' })
    await user.click(screen.getByRole('button', { name: 'Add question' }))

    await user.click(screen.getByRole('button', { name: 'Remove question 2' }))

    expect(screen.queryByLabelText('Prompt for question 2')).not.toBeInTheDocument()
  })

  it('should not allow removing the question when it is the only one', async () => {
    signInAs(officer)
    renderPage()

    expect(await screen.findByRole('button', { name: 'Remove question 1' })).toBeDisabled()
  })

  it('should swap the order of two questions when the officer moves one up', async () => {
    const user = userEvent.setup()
    signInAs(officer)
    renderPage()
    await screen.findByRole('heading', { name: 'New Survey' })
    await user.type(screen.getByLabelText('Prompt for question 1'), 'First')
    await user.click(screen.getByRole('button', { name: 'Add question' }))
    await user.type(screen.getByLabelText('Prompt for question 2'), 'Second')

    await user.click(screen.getByRole('button', { name: 'Move question 2 up' }))

    expect(screen.getByLabelText('Prompt for question 1')).toHaveValue('Second')
    expect(screen.getByLabelText('Prompt for question 2')).toHaveValue('First')
  })

  it('should hide the custom options box when the officer picks a preset', async () => {
    const user = userEvent.setup()
    signInAs(officer)
    renderPage()
    await screen.findByRole('heading', { name: 'New Survey' })
    expect(screen.getByLabelText('Custom options for question 1, one per line')).toBeInTheDocument()

    await user.selectOptions(screen.getByLabelText('Options for question 1'), 'Weekdays')

    expect(screen.queryByLabelText('Custom options for question 1, one per line')).not.toBeInTheDocument()
    const sources = within(screen.getByLabelText('Options for question 1')).getAllByRole('option')
    expect(sources.map((option) => option.textContent)).toEqual([
      'Classic classes',
      'Roles',
      'Weekdays',
      'Custom options',
    ])
  })

  it('should send the visitor to the home page when they are not an officer', async () => {
    signInAs(member)

    renderPage()

    expect(await screen.findByText('Home page')).toBeInTheDocument()
  })
})
