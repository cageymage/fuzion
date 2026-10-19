import { screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { renderWithProviders } from '../../testUtils'
import { HubHeader } from './HubHeader'

describe('HubHeader', () => {
  it('should link the wordmark to the hub home page', () => {
    renderWithProviders(<HubHeader />)

    expect(screen.getByRole('link', { name: 'FUZION GAMING' })).toHaveAttribute('href', '/')
  })
})
