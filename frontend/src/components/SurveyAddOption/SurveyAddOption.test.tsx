import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import type { SurveyQuestion } from '../../types/surveys'
import { SurveyAddOption } from './SurveyAddOption'

const questions: SurveyQuestion[] = [
  { id: 'q-class', prompt: 'Class', type: 'single', required: true, options: ['Mage', 'Priest'] },
  { id: 'q-nights', prompt: 'Raid nights', type: 'multi', required: false, options: ['Tuesday', 'Thursday'] },
]

function renderForm(overrides: Partial<Parameters<typeof SurveyAddOption>[0]> = {}) {
  const onSubmit = vi.fn().mockResolvedValue(undefined)
  render(<SurveyAddOption questions={questions} pending={false} error={null} onSubmit={onSubmit} {...overrides} />)
  return onSubmit
}

describe('SurveyAddOption', () => {
  it('should hand the chosen question and the trimmed option to onSubmit when the officer adds an option', async () => {
    const user = userEvent.setup()
    const onSubmit = renderForm()

    await user.selectOptions(screen.getByLabelText('Question'), 'Raid nights')
    await user.type(screen.getByLabelText('New option'), '  Friday  ')
    await user.click(screen.getByRole('button', { name: 'Add option' }))

    expect(onSubmit).toHaveBeenCalledTimes(1)
    expect(onSubmit).toHaveBeenCalledWith('q-nights', 'Friday')
  })

  it('should clear the option field when the option was added', async () => {
    const user = userEvent.setup()
    renderForm()

    await user.type(screen.getByLabelText('New option'), 'Druid')
    await user.click(screen.getByRole('button', { name: 'Add option' }))

    expect(screen.getByLabelText('New option')).toHaveValue('')
  })

  it('should keep the typed option when adding it is rejected', async () => {
    const user = userEvent.setup()
    renderForm({ onSubmit: vi.fn().mockRejectedValue(new Error('rejected')), error: 'option: "Mage" is already an option' })

    await user.type(screen.getByLabelText('New option'), 'Mage')
    await user.click(screen.getByRole('button', { name: 'Add option' }))

    expect(screen.getByLabelText('New option')).toHaveValue('Mage')
    expect(screen.getByRole('alert')).toHaveTextContent('option: "Mage" is already an option')
  })

  it('should disable the add button when the option field is blank', async () => {
    const user = userEvent.setup()
    renderForm()

    expect(screen.getByRole('button', { name: 'Add option' })).toBeDisabled()
    await user.type(screen.getByLabelText('New option'), '   ')

    expect(screen.getByRole('button', { name: 'Add option' })).toBeDisabled()
  })

  it('should disable the add button while the option is being added', () => {
    renderForm({ pending: true })

    expect(screen.getByRole('button', { name: 'Adding…' })).toBeDisabled()
  })
})
