import { screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { renderWithProviders } from '../../testUtils'
import { Charter } from './Charter'

describe('Charter', () => {
  it.each(['Guild rules', 'How we treat people', 'Raids', 'Loot'])(
    'should show the %s section heading when the page renders',
    (heading) => {
      renderWithProviders(<Charter />)

      expect(screen.getByRole('heading', { level: 2, name: heading })).toBeInTheDocument()
    },
  )

  it('should say the charter is a draft when the page renders', () => {
    renderWithProviders(<Charter />)

    expect(screen.getByText(/This is a draft for the guild leaders to review/)).toBeInTheDocument()
  })

  it('should ban GDKP raids and gold buying when the page renders', () => {
    renderWithProviders(<Charter />)

    const rules = screen.getByRole('region', { name: 'Guild rules' })
    expect(rules).toHaveTextContent('Running or joining GDKP raids')
    expect(rules).toHaveTextContent('Buying gold')
  })

  it('should say banter and adult humor are expected when the page renders', () => {
    renderWithProviders(<Charter />)

    expect(screen.getByRole('region', { name: 'How we treat people' })).toHaveTextContent(
      'adult conversation and humor are to be expected',
    )
  })

  it('should say the raid schedule is not set yet when the page renders', () => {
    renderWithProviders(<Charter />)

    expect(screen.getByRole('region', { name: 'Raids' })).toHaveTextContent(
      'The raid schedule is not set yet',
    )
  })

  it('should name the in-game roll system as the expected loot system when the page renders', () => {
    renderWithProviders(<Charter />)

    const loot = screen.getByRole('region', { name: 'Loot' })
    expect(loot).toHaveTextContent('in-game roll system')
    expect(loot).toHaveTextContent('This is not final')
  })
})
