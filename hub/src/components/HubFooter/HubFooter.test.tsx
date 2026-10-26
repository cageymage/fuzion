import { screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { renderWithProviders } from '../../testUtils'
import { HubFooter } from './HubFooter'

describe('HubFooter', () => {
  it('should link to the legal page', () => {
    renderWithProviders(<HubFooter />)

    expect(screen.getByRole('link', { name: 'Legal' })).toHaveAttribute('href', '/legal')
  })

  it('should link to the Fuzion GitHub repo when the footer renders', () => {
    renderWithProviders(<HubFooter />)

    expect(
      screen.getByRole('link', { name: 'Fuzion source code on GitHub' }),
    ).toHaveAttribute('href', 'https://github.com/cageymage/fuzion')
  })

  it('should open the GitHub link in a new tab when the footer renders', () => {
    renderWithProviders(<HubFooter />)

    const link = screen.getByRole('link', { name: 'Fuzion source code on GitHub' })
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveAttribute('rel', 'noopener noreferrer')
  })
})
