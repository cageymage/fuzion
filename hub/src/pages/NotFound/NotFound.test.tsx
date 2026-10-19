import { screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { renderWithProviders } from '../../testUtils'
import { NotFound } from './NotFound'

describe('NotFound', () => {
  it('should say the page was not found', () => {
    renderWithProviders(<NotFound />)

    expect(screen.getByRole('heading', { level: 1, name: 'Page not found' })).toBeInTheDocument()
  })

  it('should link back to the hub home page', () => {
    renderWithProviders(<NotFound />)

    expect(screen.getByRole('link', { name: 'Back to Fuzion Gaming' })).toHaveAttribute('href', '/')
  })
})
