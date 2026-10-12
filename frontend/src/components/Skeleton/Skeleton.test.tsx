import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { LoadingPlaceholder, Skeleton } from './Skeleton'

describe('LoadingPlaceholder', () => {
  it('should announce its label as a status when rendered', () => {
    render(<LoadingPlaceholder label="Loading news…" />)

    expect(screen.getByRole('status')).toHaveTextContent('Loading news…')
  })

  it('should hide the skeleton blocks from assistive technology when they are children', () => {
    const { container } = render(
      <LoadingPlaceholder label="Loading news…">
        <Skeleton height={40} />
      </LoadingPlaceholder>,
    )

    const block = container.querySelector('[data-skeleton]')
    expect(block).toHaveAttribute('aria-hidden', 'true')
    expect(block).toHaveStyle({ height: '40px' })
  })
})
