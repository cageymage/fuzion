import { screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { renderWithProviders } from '../testUtils'
import { App } from './App'

describe('App', () => {
  it('should show the legal page when the path is /legal', () => {
    renderWithProviders(<App />, '/legal')

    expect(screen.getByRole('heading', { level: 1, name: 'Legal' })).toBeInTheDocument()
  })

  it('should show page not found when the path matches no page', () => {
    renderWithProviders(<App />, '/roster')

    expect(screen.getByRole('heading', { level: 1, name: 'Page not found' })).toBeInTheDocument()
  })
})
