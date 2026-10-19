import { screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { renderWithProviders } from '../../testUtils'
import { HubFooter } from './HubFooter'

describe('HubFooter', () => {
  it('should link to the legal page', () => {
    renderWithProviders(<HubFooter />)

    expect(screen.getByRole('link', { name: 'Legal' })).toHaveAttribute('href', '/legal')
  })
})
