import { screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { renderWithProviders } from '../../testUtils'
import { AppFooter } from './AppFooter'

describe('AppFooter', () => {
  it('should name the guild, realm and game', () => {
    renderWithProviders(<AppFooter />)

    expect(
      screen.getByText('Fuzion · PvE · World of Warcraft: Forever'),
    ).toBeInTheDocument()
  })

  it('should link to the about page', () => {
    renderWithProviders(<AppFooter />)

    expect(screen.getByRole('link', { name: 'About' })).toHaveAttribute('href', '/about')
  })

  it('should link to the charter page', () => {
    renderWithProviders(<AppFooter />)

    expect(screen.getByRole('link', { name: 'Charter' })).toHaveAttribute('href', '/charter')
  })

  it('should link to the legal page', () => {
    renderWithProviders(<AppFooter />)

    expect(screen.getByRole('link', { name: 'Legal' })).toHaveAttribute('href', '/legal')
  })

  it('should link to the Fuzion GitHub repo when the footer renders', () => {
    renderWithProviders(<AppFooter />)

    expect(
      screen.getByRole('link', { name: 'Fuzion source code on GitHub' }),
    ).toHaveAttribute('href', 'https://github.com/cageymage/fuzion')
  })

  it('should open the GitHub link in a new tab when the footer renders', () => {
    renderWithProviders(<AppFooter />)

    const link = screen.getByRole('link', { name: 'Fuzion source code on GitHub' })
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveAttribute('rel', 'noopener noreferrer')
  })
})
