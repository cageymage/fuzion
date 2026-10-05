import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { AppFooter } from './AppFooter'

describe('AppFooter', () => {
  it('should name the guild, realm and game', () => {
    render(<AppFooter />)

    expect(
      screen.getByText('Fuzion · Emberreach · World of Warcraft: Forever'),
    ).toBeInTheDocument()
  })

  it('should credit Blizzard Entertainment when profession icons are shown on the site', () => {
    render(<AppFooter />)

    expect(screen.getByText('Profession icons © Blizzard Entertainment')).toBeInTheDocument()
  })
})
