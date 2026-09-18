import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { Chip } from './Chip'

describe('Chip', () => {
  it('should render its label text', () => {
    render(<Chip tone="epic">Raid Progress</Chip>)

    expect(screen.getByText('Raid Progress')).toBeInTheDocument()
  })

  it('should apply the legendary tone class when the tone is legendary', () => {
    render(<Chip tone="legendary">Recruitment</Chip>)

    expect(screen.getByText('Recruitment').className).toContain('legendary')
  })
})
