import { screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { renderWithProviders } from '../../testUtils'
import { AppFooter } from './AppFooter'

describe('AppFooter', () => {
  it('should name the guild, realm and game', () => {
    renderWithProviders(<AppFooter />)

    expect(
      screen.getByText('Fuzion · Emberreach · World of Warcraft: Forever'),
    ).toBeInTheDocument()
  })

  it('should link to the legal page', () => {
    renderWithProviders(<AppFooter />)

    expect(screen.getByRole('link', { name: 'Legal' })).toHaveAttribute('href', '/legal')
  })
})
