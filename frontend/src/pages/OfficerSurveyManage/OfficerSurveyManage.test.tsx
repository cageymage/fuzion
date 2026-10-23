import { fireEvent, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { Route, Routes } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { surveyExportUrl } from '../../api/surveys'
import { toLocalInput } from '../../components/SurveyEditorForm/surveyDraft'
import { ToastProvider } from '../../components/Toast/Toast'
import { closedSurvey, openSurvey } from '../../mocks/handlers'
import { server } from '../../mocks/server'
import { renderWithProviders } from '../../testUtils'
import { OfficerSurveyManage } from './OfficerSurveyManage'

const officer = { id: 'user-1', username: 'Officer', avatarUrl: null, isOfficer: true }
const member = { id: 'user-2', username: 'Member', avatarUrl: null, isOfficer: false }

function signInAs(user: typeof officer) {
  server.use(http.get('/api/auth/me', () => HttpResponse.json(user)))
}

function renderPage() {
  return renderWithProviders(
    <ToastProvider>
      <Routes>
        <Route path="/officer/surveys/:id" element={<OfficerSurveyManage />} />
        <Route path="/officer/surveys" element={<p>Survey list</p>} />
        <Route path="/" element={<p>Home page</p>} />
      </Routes>
    </ToastProvider>,
    '/officer/surveys/survey-1',
  )
}

async function renderForOfficer() {
  signInAs(officer)
  renderPage()
  await screen.findByRole('heading', { name: 'Spring roster' })
}

function captureSurveyPatch() {
  const captured: { body: unknown } = { body: null }
  server.use(
    http.patch('/api/surveys/:id', async ({ request }) => {
      captured.body = await request.json()
      return HttpResponse.json(openSurvey)
    }),
  )
  return captured
}

function captureResponsePatch() {
  const captured: { body: unknown } = { body: null }
  server.use(
    http.patch('/api/surveys/:id/responses/:responseId', async ({ request }) => {
      captured.body = await request.json()
      return HttpResponse.json({})
    }),
  )
  return captured
}

describe('OfficerSurveyManage', () => {
  it('should show the survey settings prefilled and links to the public page and CSV when the officer opens it', async () => {
    await renderForOfficer()

    expect(screen.getByLabelText('Title')).toHaveValue('Spring roster')
    expect(screen.getByLabelText('Description')).toHaveValue('Help us plan the raid teams')
    expect(screen.getByLabelText('Closes at')).toHaveValue(toLocalInput(openSurvey.closesAt))
    expect(screen.getByRole('link', { name: 'View public page' })).toHaveAttribute('href', '/surveys/survey-1')
    const csv = screen.getByRole('link', { name: 'Download CSV' })
    expect(csv).toHaveAttribute('href', surveyExportUrl('survey-1'))
    expect(csv).toHaveAttribute('download')
  })

  it('should list every response with its Discord details and answers when responses exist', async () => {
    await renderForOfficer()

    const row = (await screen.findByRole('cell', { name: 'ana' })).closest('tr')
    expect(row).not.toBeNull()
    const cells = within(row as HTMLElement).getAllByRole('cell')
    expect(cells.map((cell) => cell.textContent)).toEqual([
      'ana',
      'Anaria',
      'Yes',
      cells[3].textContent,
      'Class: MageRaid nights: Tuesday, Thursday',
      'Counted',
      'Void',
    ])
  })

  it('should show who voided a response and why when a response is voided', async () => {
    await renderForOfficer()

    const row = (await screen.findByRole('cell', { name: 'ben' })).closest('tr') as HTMLElement
    expect(within(row).getByText('Voided by Officer: joke answers')).toBeInTheDocument()
    expect(within(row).getByText('Not given')).toBeInTheDocument()
    expect(within(row).getByRole('button', { name: 'Restore ben' })).toBeInTheDocument()
  })

  it('should show the response and voided counts when responses exist', async () => {
    await renderForOfficer()

    expect(await screen.findByText('2 responses, 1 voided')).toBeInTheDocument()
  })

  it('should hide people outside the Discord server when the officer ticks the server filter', async () => {
    const user = userEvent.setup()
    await renderForOfficer()
    await screen.findByRole('cell', { name: 'ben' })

    await user.click(screen.getByRole('checkbox', { name: 'Only people in our Discord server' }))

    expect(screen.getByRole('cell', { name: 'ana' })).toBeInTheDocument()
    expect(screen.queryByRole('cell', { name: 'ben' })).not.toBeInTheDocument()
  })

  it('should say there are no responses yet when nobody has responded', async () => {
    server.use(http.get('/api/surveys/:id/responses', () => HttpResponse.json([])))

    await renderForOfficer()

    expect(await screen.findByText('No responses yet.')).toBeInTheDocument()
  })

  it('should show an error when the responses fail to load', async () => {
    server.use(http.get('/api/surveys/:id/responses', () => new HttpResponse(null, { status: 500 })))

    await renderForOfficer()

    expect(await screen.findByRole('alert')).toHaveTextContent('The responses could not be loaded.')
  })

  it('should show the questions read-only with a note when responses exist', async () => {
    await renderForOfficer()

    expect(await screen.findByText('Questions cannot change once people have responded.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Add question' })).not.toBeInTheDocument()
    expect(
      screen.getByText((_, element) => element?.tagName === 'LI' && element.textContent === 'Class (single choice, required): Mage, Priest, Warrior'),
    ).toBeInTheDocument()
  })

  it('should let the officer edit the questions when nobody has responded', async () => {
    server.use(http.get('/api/surveys/:id/responses', () => HttpResponse.json([])))

    await renderForOfficer()

    expect(await screen.findByRole('button', { name: 'Add question' })).toBeInTheDocument()
    expect(screen.getByLabelText('Prompt for question 1')).toHaveValue('Class')
  })

  it('should save the title, description and deadline without the questions when responses exist', async () => {
    const user = userEvent.setup()
    const patch = captureSurveyPatch()
    await renderForOfficer()
    await screen.findByText('Questions cannot change once people have responded.')

    await user.clear(screen.getByLabelText('Title'))
    await user.type(screen.getByLabelText('Title'), 'Spring roster 2')
    await user.click(screen.getByRole('button', { name: 'Save changes' }))

    expect(await screen.findByText('Survey saved')).toBeInTheDocument()
    expect(patch.body).toEqual({
      title: 'Spring roster 2',
      description: 'Help us plan the raid teams',
      closesAt: new Date(toLocalInput(openSurvey.closesAt)).toISOString(),
    })
  })

  it('should save the questions too when nobody has responded', async () => {
    const user = userEvent.setup()
    server.use(http.get('/api/surveys/:id/responses', () => HttpResponse.json([])))
    const patch = captureSurveyPatch()
    await renderForOfficer()
    await screen.findByRole('button', { name: 'Add question' })

    await user.selectOptions(screen.getByLabelText('Options for question 2'), 'Roles')
    await user.click(screen.getByRole('button', { name: 'Save changes' }))

    await screen.findByText('Survey saved')
    expect(patch.body).toEqual({
      title: 'Spring roster',
      description: 'Help us plan the raid teams',
      closesAt: new Date(toLocalInput(openSurvey.closesAt)).toISOString(),
      questions: [
        { prompt: 'Class', type: 'single', required: true, options: ['Mage', 'Priest', 'Warrior'] },
        { prompt: 'Secondary class', type: 'single', required: false, preset: 'wow-roles' },
        { prompt: 'Raid nights', type: 'multi', required: true, options: ['Tuesday', 'Wednesday', 'Thursday'] },
      ],
    })
  })

  it('should ask for a closing time and send nothing when the officer clears the deadline', async () => {
    const user = userEvent.setup()
    const patch = captureSurveyPatch()
    await renderForOfficer()
    await screen.findByText('Questions cannot change once people have responded.')

    fireEvent.change(screen.getByLabelText('Closes at'), { target: { value: '' } })
    await user.click(screen.getByRole('button', { name: 'Save changes' }))

    expect(screen.getByRole('alert')).toHaveTextContent('Pick when the survey closes')
    expect(patch.body).toBeNull()
  })

  it("should show the server's message when saving is rejected", async () => {
    const user = userEvent.setup()
    server.use(
      http.patch('/api/surveys/:id', () => HttpResponse.json({ error: 'title: is required' }, { status: 400 })),
    )
    await renderForOfficer()
    await screen.findByText('Questions cannot change once people have responded.')

    await user.clear(screen.getByLabelText('Title'))
    await user.click(screen.getByRole('button', { name: 'Save changes' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('title: is required')
  })

  it('should close the survey at the current time when the officer confirms closing it now', async () => {
    const user = userEvent.setup()
    const patch = captureSurveyPatch()
    await renderForOfficer()

    await user.click(screen.getByRole('button', { name: 'Close now' }))
    const before = Date.now()
    await user.click(within(await screen.findByRole('dialog')).getByRole('button', { name: 'Close survey' }))

    expect(await screen.findByText('Survey closed')).toBeInTheDocument()
    const body = patch.body as { closesAt: string }
    expect(Object.keys(body)).toEqual(['closesAt'])
    expect(Math.abs(new Date(body.closesAt).getTime() - before)).toBeLessThan(5_000)
  })

  it('should show the new deadline in the settings form after the officer closes the survey now', async () => {
    const user = userEvent.setup()
    let surveyRequests = 0
    server.use(
      http.get('/api/surveys/:id', () => {
        surveyRequests += 1
        return HttpResponse.json(surveyRequests === 1 ? openSurvey : closedSurvey)
      }),
    )
    captureSurveyPatch()
    await renderForOfficer()
    expect(screen.getByLabelText('Closes at')).toHaveValue(toLocalInput(openSurvey.closesAt))

    await user.click(screen.getByRole('button', { name: 'Close now' }))
    await user.click(within(await screen.findByRole('dialog')).getByRole('button', { name: 'Close survey' }))

    await screen.findByText('Survey closed')
    expect(screen.getByLabelText('Closes at')).toHaveValue(toLocalInput(closedSurvey.closesAt))
  })

  it('should leave the survey open when the officer cancels closing it now', async () => {
    const user = userEvent.setup()
    const patch = captureSurveyPatch()
    await renderForOfficer()

    await user.click(screen.getByRole('button', { name: 'Close now' }))
    await user.click(within(await screen.findByRole('dialog')).getByRole('button', { name: 'Cancel' }))

    expect(patch.body).toBeNull()
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('should not offer to close the survey now when it is already closed', async () => {
    server.use(http.get('/api/surveys/:id', () => HttpResponse.json(closedSurvey)))

    await renderForOfficer()

    expect(screen.queryByRole('button', { name: 'Close now' })).not.toBeInTheDocument()
  })

  it('should void the response with the reason when the officer confirms the void dialog', async () => {
    const user = userEvent.setup()
    const patch = captureResponsePatch()
    await renderForOfficer()

    await user.click(await screen.findByRole('button', { name: 'Void ana' }))
    const dialog = await screen.findByRole('dialog')
    await user.type(within(dialog).getByLabelText('Reason (optional)'), '  joke answers  ')
    await user.click(within(dialog).getByRole('button', { name: 'Void response' }))

    await screen.findByText('2 responses, 1 voided')
    expect(patch.body).toEqual({ voided: true, reason: 'joke answers' })
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('should send an empty reason when the officer voids without giving one', async () => {
    const user = userEvent.setup()
    const patch = captureResponsePatch()
    await renderForOfficer()

    await user.click(await screen.findByRole('button', { name: 'Void ana' }))
    await user.click(within(await screen.findByRole('dialog')).getByRole('button', { name: 'Void response' }))

    await screen.findByText('2 responses, 1 voided')
    expect(patch.body).toEqual({ voided: true, reason: '' })
  })

  it('should not void anything when the officer cancels the void dialog', async () => {
    const user = userEvent.setup()
    const patch = captureResponsePatch()
    await renderForOfficer()

    await user.click(await screen.findByRole('button', { name: 'Void ana' }))
    await user.click(within(await screen.findByRole('dialog')).getByRole('button', { name: 'Cancel' }))

    expect(patch.body).toBeNull()
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it("should show the server's message in the dialog when voiding is rejected", async () => {
    const user = userEvent.setup()
    server.use(
      http.patch('/api/surveys/:id/responses/:responseId', () =>
        HttpResponse.json({ error: 'reason: must be at most 500 characters' }, { status: 400 }),
      ),
    )
    await renderForOfficer()

    await user.click(await screen.findByRole('button', { name: 'Void ana' }))
    await user.click(within(await screen.findByRole('dialog')).getByRole('button', { name: 'Void response' }))

    expect(await within(screen.getByRole('dialog')).findByRole('alert')).toHaveTextContent(
      'reason: must be at most 500 characters',
    )
  })

  it('should restore a voided response without a reason when the officer clicks restore', async () => {
    const user = userEvent.setup()
    const patch = captureResponsePatch()
    await renderForOfficer()

    await user.click(await screen.findByRole('button', { name: 'Restore ben' }))

    await screen.findByText('2 responses, 1 voided')
    expect(patch.body).toEqual({ voided: false })
  })

  it('should show an error when the survey fails to load', async () => {
    signInAs(officer)
    server.use(http.get('/api/surveys/:id', () => new HttpResponse(null, { status: 500 })))

    renderPage()

    expect(await screen.findByRole('alert', {}, { timeout: 3000 })).toHaveTextContent(
      'The survey could not be loaded.',
    )
  })

  it('should send the visitor to the home page when they are not an officer', async () => {
    signInAs(member)

    renderPage()

    expect(await screen.findByText('Home page')).toBeInTheDocument()
  })
})
