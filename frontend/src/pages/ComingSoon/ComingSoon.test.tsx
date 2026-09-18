import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { ComingSoon } from './ComingSoon'

describe('ComingSoon', () => {
  it('should show the name of the page that is not built yet', () => {
    render(<ComingSoon title="Roster" />)

    expect(screen.getByRole('heading', { name: 'Roster' })).toBeInTheDocument()
    expect(screen.getByText('This page has not been built yet.')).toBeInTheDocument()
  })
})
