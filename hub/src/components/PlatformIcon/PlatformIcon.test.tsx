import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { PlatformIcon } from './PlatformIcon'

describe('PlatformIcon', () => {
  it('should hide the icon from assistive technology so the link text names the link', () => {
    const { container } = render(<PlatformIcon platform="twitch" />)

    expect(container.querySelector('svg')).toHaveAttribute('aria-hidden', 'true')
  })
})
