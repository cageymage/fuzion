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
})
