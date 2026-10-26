import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { SurveyEditorForm } from './SurveyEditorForm'
import type { SurveyDraft } from './surveyDraft'

const draft: SurveyDraft = {
  title: 'Spring roster',
  description: 'Help us plan',
  closesAt: '2026-03-20T20:00',
  questions: [
    {
      key: 'q-1',
      prompt: 'Class',
      type: 'single',
      required: true,
      source: 'custom',
      customOptions: 'Mage\nPriest',
    },
    {
      key: 'q-2',
      prompt: 'Raid nights',
      type: 'multi',
      required: false,
      source: 'custom',
      customOptions: 'Tuesday\nThursday',
    },
  ],
}

function renderForm(overrides: Partial<Parameters<typeof SurveyEditorForm>[0]> = {}) {
  const onSubmit = vi.fn()
  render(
    <SurveyEditorForm
      initial={draft}
      questionsLocked={false}
      submitLabel="Save changes"
      pending={false}
      error={null}
      onSubmit={onSubmit}
      {...overrides}
    />,
  )
  return onSubmit
}

describe('SurveyEditorForm', () => {
  it('should hand the edited draft to onSubmit when the officer changes the title and saves', async () => {
    const user = userEvent.setup()
    const onSubmit = renderForm()

    await user.clear(screen.getByLabelText('Title'))
    await user.type(screen.getByLabelText('Title'), 'Spring roster 2')
    await user.click(screen.getByRole('button', { name: 'Save changes' }))

    expect(onSubmit).toHaveBeenCalledTimes(1)
    expect(onSubmit).toHaveBeenCalledWith({ ...draft, title: 'Spring roster 2' })
  })

  it('should show the question builder with one card per question when the questions are not locked', () => {
    renderForm()

    expect(screen.getByLabelText('Prompt for question 1')).toHaveValue('Class')
    expect(screen.getByLabelText('Prompt for question 2')).toHaveValue('Raid nights')
    expect(screen.getByRole('button', { name: 'Add question' })).toBeInTheDocument()
  })

  it('should show the questions read-only with a note and no builder when the questions are locked', () => {
    renderForm({ questionsLocked: true })

    expect(screen.getByText('Questions cannot change once people have responded.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Add question' })).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Prompt for question 1')).not.toBeInTheDocument()
    expect(
      screen.getByText((_, element) => element?.tagName === 'LI' && element.textContent === 'Class (single choice, required): Mage, Priest'),
    ).toBeInTheDocument()
    expect(
      screen.getByText(
        (_, element) =>
          element?.tagName === 'LI' && element.textContent === 'Raid nights (multiple choice): Tuesday, Thursday',
      ),
    ).toBeInTheDocument()
  })

  it('should disable the submit button and say it is saving when a save is pending', () => {
    renderForm({ pending: true })

    expect(screen.getByRole('button', { name: 'Saving…' })).toBeDisabled()
  })

  it('should show the error message when the parent passes one', () => {
    renderForm({ error: 'title: is required' })

    expect(screen.getByRole('alert')).toHaveTextContent('title: is required')
  })
})
