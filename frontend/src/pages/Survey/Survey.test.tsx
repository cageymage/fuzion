import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { Route, Routes } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { loginUrl } from '../../api/auth'
import { ToastProvider } from '../../components/Toast/Toast'
import { closedSurvey, openSurvey } from '../../mocks/handlers'
import { server } from '../../mocks/server'
import { renderWithProviders } from '../../testUtils'
import type { Survey as SurveyData } from '../../types/surveys'
import { Survey } from './Survey'

const member = { id: 'user-2', username: 'Member', avatarUrl: null, isOfficer: false, isGuildMember: true }

function signInAsMember() {
  server.use(http.get('/api/auth/me', () => HttpResponse.json(member)))
}

const classQuestion = () => screen.getByRole('group', { name: 'Class *' })

function serveSurvey(survey: SurveyData) {
  server.use(http.get('/api/surveys/:id', () => HttpResponse.json(survey)))
}

function renderSurveyPage() {
  return renderWithProviders(
    <ToastProvider>
      <Routes>
        <Route path="/surveys/:id" element={<Survey />} />
        <Route path="/surveys" element={<p>Surveys list</p>} />
      </Routes>
    </ToastProvider>,
    '/surveys/survey-1',
  )
}

describe('Survey', () => {
  it('should show the title, description and deadline when the survey is open', async () => {
    signInAsMember()

    renderSurveyPage()

    expect(await screen.findByRole('heading', { name: 'Spring roster' })).toBeInTheDocument()
    expect(screen.getByText('Help us plan the raid teams')).toBeInTheDocument()
    expect(screen.getByText(/^Closes /)).toBeInTheDocument()
  })

  it('should offer a Discord login link that returns to the survey instead of the form when the visitor is not logged in', async () => {
    renderSurveyPage()

    expect(await screen.findByRole('link', { name: 'Log in with Discord' })).toHaveAttribute(
      'href',
      `${loginUrl}?returnTo=%2Fsurveys%2Fsurvey-1`,
    )
    expect(screen.queryByRole('button', { name: 'Submit response' })).not.toBeInTheDocument()
  })

  it('should show a members-only notice instead of the form when the user is not a guild member', async () => {
    server.use(http.get('/api/auth/me', () => HttpResponse.json({ ...member, isGuildMember: false })))

    renderSurveyPage()

    expect(
      await screen.findByText(
        'Only members of the guild Discord server can respond. If you have just joined, log out and back in to refresh your membership.',
      ),
    ).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Submit response' })).not.toBeInTheDocument()
  })

  it('should render radio groups for single choice questions and checkboxes for multi choice when logged in', async () => {
    signInAsMember()

    renderSurveyPage()

    const classGroup = await screen.findByRole('group', { name: 'Class *' })
    expect(within(classGroup).getAllByRole('radio').map((radio) => radio.getAttribute('value'))).toEqual([
      'Mage',
      'Priest',
      'Warrior',
    ])
    const nightsGroup = screen.getByRole('group', { name: 'Raid nights *' })
    expect(within(nightsGroup).getAllByRole('checkbox').map((box) => box.getAttribute('value'))).toEqual([
      'Tuesday',
      'Wednesday',
      'Thursday',
    ])
    expect(screen.getByRole('group', { name: 'Secondary class' })).toBeInTheDocument()
  })

  it('should tell the member that officers can see their answers and link to the legal page when the form is shown', async () => {
    signInAsMember()

    renderSurveyPage()

    expect(await screen.findByText(/officers can see your answers and your discord name/i)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Legal page' })).toHaveAttribute('href', '/legal')
  })

  it('should list the unanswered required questions and send nothing when the member submits an empty form', async () => {
    const user = userEvent.setup()
    let requests = 0
    server.use(
      http.put('/api/surveys/:id/response', () => {
        requests += 1
        return HttpResponse.json({})
      }),
    )
    signInAsMember()
    renderSurveyPage()

    await user.click(await screen.findByRole('button', { name: 'Submit response' }))

    const alert = screen.getByRole('alert')
    expect(within(alert).getAllByRole('listitem').map((item) => item.textContent)).toEqual([
      'Class',
      'Raid nights',
    ])
    expect(requests).toBe(0)
  })

  it('should send the chosen answers in option order and the trimmed guild name when the member submits', async () => {
    const user = userEvent.setup()
    let submitted: unknown = null
    server.use(
      http.put('/api/surveys/:id/response', async ({ request }) => {
        submitted = await request.json()
        return HttpResponse.json({})
      }),
    )
    signInAsMember()
    renderSurveyPage()
    await screen.findByRole('button', { name: 'Submit response' })

    await user.type(screen.getByLabelText(/name you use in the guild/i), '  Thunder  ')
    await user.click(within(classQuestion()).getByRole('radio', { name: 'Mage' }))
    await user.click(screen.getByRole('checkbox', { name: 'Thursday' }))
    await user.click(screen.getByRole('checkbox', { name: 'Tuesday' }))
    await user.click(screen.getByRole('button', { name: 'Submit response' }))

    await screen.findByText('Response saved')
    expect(submitted).toEqual({
      guildName: 'Thunder',
      answers: [
        { questionId: 'q-class', values: ['Mage'] },
        { questionId: 'q-nights', values: ['Tuesday', 'Thursday'] },
      ],
    })
  })

  it('should leave the optional question out of the request when the member clears their secondary choice', async () => {
    const user = userEvent.setup()
    let submitted: unknown = null
    server.use(
      http.put('/api/surveys/:id/response', async ({ request }) => {
        submitted = await request.json()
        return HttpResponse.json({})
      }),
    )
    signInAsMember()
    renderSurveyPage()
    await screen.findByRole('button', { name: 'Submit response' })

    await user.click(within(screen.getByRole('group', { name: 'Class *' })).getByRole('radio', { name: 'Mage' }))
    await user.click(
      within(screen.getByRole('group', { name: 'Secondary class' })).getByRole('radio', { name: 'Priest' }),
    )
    await user.click(screen.getByRole('button', { name: 'Clear Secondary class' }))
    await user.click(screen.getByRole('checkbox', { name: 'Tuesday' }))
    await user.click(screen.getByRole('button', { name: 'Submit response' }))

    await screen.findByText('Response saved')
    expect(submitted).toEqual({
      guildName: '',
      answers: [
        { questionId: 'q-class', values: ['Mage'] },
        { questionId: 'q-nights', values: ['Tuesday'] },
      ],
    })
  })

  it('should prefill the form and offer to update when the member has already responded', async () => {
    serveSurvey({
      ...openSurvey,
      myResponse: {
        guildName: 'Thunder',
        answers: [
          { questionId: 'q-class', values: ['Priest'] },
          { questionId: 'q-nights', values: ['Wednesday', 'Thursday'] },
        ],
      },
    })
    signInAsMember()

    renderSurveyPage()

    expect(await screen.findByRole('button', { name: 'Update response' })).toBeInTheDocument()
    expect(screen.getByLabelText(/name you use in the guild/i)).toHaveValue('Thunder')
    expect(within(classQuestion()).getByRole('radio', { name: 'Priest' })).toBeChecked()
    expect(screen.getByRole('checkbox', { name: 'Wednesday' })).toBeChecked()
    expect(screen.getByRole('checkbox', { name: 'Thursday' })).toBeChecked()
    expect(screen.getByRole('checkbox', { name: 'Tuesday' })).not.toBeChecked()
  })

  it("should show the server's message when the server rejects the answers", async () => {
    const user = userEvent.setup()
    server.use(
      http.put('/api/surveys/:id/response', () =>
        HttpResponse.json({ error: 'answers: "Shaman" is not an option for Class' }, { status: 400 }),
      ),
    )
    signInAsMember()
    renderSurveyPage()
    await screen.findByRole('button', { name: 'Submit response' })

    await user.click(within(classQuestion()).getByRole('radio', { name: 'Mage' }))
    await user.click(screen.getByRole('checkbox', { name: 'Tuesday' }))
    await user.click(screen.getByRole('button', { name: 'Submit response' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('answers: "Shaman" is not an option for Class')
  })

  it('should show a generic error when saving the response fails on the server', async () => {
    const user = userEvent.setup()
    server.use(http.put('/api/surveys/:id/response', () => new HttpResponse(null, { status: 500 })))
    signInAsMember()
    renderSurveyPage()
    await screen.findByRole('button', { name: 'Submit response' })

    await user.click(within(classQuestion()).getByRole('radio', { name: 'Mage' }))
    await user.click(screen.getByRole('checkbox', { name: 'Tuesday' }))
    await user.click(screen.getByRole('button', { name: 'Submit response' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Something went wrong, try again or ping an officer on Discord',
    )
  })

  it('should switch to the results and say the survey closed when the deadline passes while the form is open', async () => {
    const user = userEvent.setup()
    let surveyRequests = 0
    server.use(
      http.get('/api/surveys/:id', () => {
        surveyRequests += 1
        return HttpResponse.json(surveyRequests === 1 ? openSurvey : closedSurvey)
      }),
      http.put('/api/surveys/:id/response', () =>
        HttpResponse.json({ error: 'survey is closed' }, { status: 409 }),
      ),
    )
    signInAsMember()
    renderSurveyPage()
    await screen.findByRole('button', { name: 'Submit response' })

    await user.click(within(classQuestion()).getByRole('radio', { name: 'Mage' }))
    await user.click(screen.getByRole('checkbox', { name: 'Tuesday' }))
    await user.click(screen.getByRole('button', { name: 'Submit response' }))

    expect(await screen.findByText('This survey has closed, here are the results')).toBeInTheDocument()
    expect(await screen.findByText('3 responses')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Submit response' })).not.toBeInTheDocument()
  })

  it('should show each option with its count and percentage of turnout when the survey is closed', async () => {
    serveSurvey(closedSurvey)

    renderSurveyPage()

    expect(await screen.findByText('3 responses')).toBeInTheDocument()
    expect(screen.getByText(/^Closed /)).toBeInTheDocument()
    const classResults = screen.getByRole('region', { name: 'Class' })
    expect(within(classResults).getByText('Mage').closest('li')).toHaveTextContent('2 (67%)')
    expect(within(classResults).getByText('Priest').closest('li')).toHaveTextContent('0 (0%)')
    expect(within(classResults).getByText('Warrior').closest('li')).toHaveTextContent('1 (33%)')
    const nightResults = screen.getByRole('region', { name: 'Raid nights' })
    expect(within(nightResults).getByText('Tuesday').closest('li')).toHaveTextContent('3 (100%)')
  })

  it('should show results without a login prompt when the survey is closed and the visitor is anonymous', async () => {
    serveSurvey(closedSurvey)

    renderSurveyPage()

    await screen.findByText('3 responses')
    expect(screen.queryByRole('link', { name: 'Log in with Discord' })).not.toBeInTheDocument()
  })

  it('should show a singular response count when only one response was counted', async () => {
    serveSurvey(closedSurvey)
    server.use(
      http.get('/api/surveys/:id/results', () =>
        HttpResponse.json({ surveyId: 'survey-1', title: 'Spring roster', turnout: 1, questions: [] }),
      ),
    )

    renderSurveyPage()

    expect(await screen.findByText('1 response')).toBeInTheDocument()
  })

  it('should show an error when the results fail to load', async () => {
    serveSurvey(closedSurvey)
    server.use(http.get('/api/surveys/:id/results', () => new HttpResponse(null, { status: 500 })))

    renderSurveyPage()

    expect(await screen.findByRole('alert')).toHaveTextContent('The results could not be loaded.')
  })

  it('should say the survey was not found and link back to the list when the id is unknown', async () => {
    server.use(
      http.get('/api/surveys/:id', () => HttpResponse.json({ error: 'survey not found' }, { status: 404 })),
    )

    renderSurveyPage()

    expect(await screen.findByRole('heading', { name: 'Survey not found' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'All surveys' })).toHaveAttribute('href', '/surveys')
  })

  it('should show an error when the survey fails to load', async () => {
    server.use(http.get('/api/surveys/:id', () => new HttpResponse(null, { status: 500 })))

    renderSurveyPage()

    // The hook retries a server error once, after react-query's one second backoff.
    expect(await screen.findByRole('alert', {}, { timeout: 3000 })).toHaveTextContent(
      'The survey could not be loaded.',
    )
  })
})
