import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { StreamerAvatar } from './StreamerAvatar'

describe('StreamerAvatar', () => {
  it('should show the profile image when the streamer has an avatar', () => {
    const { container } = render(
      <StreamerAvatar name="Thundermane" avatarUrl="https://cdn.example/thundermane.png" size="small" />,
    )

    expect(container.querySelector('img')).toHaveAttribute(
      'src',
      'https://cdn.example/thundermane.png',
    )
  })

  it('should show the first letter of the name when the streamer has no avatar', () => {
    const { container } = render(<StreamerAvatar name="moonveil" avatarUrl={null} size="small" />)

    expect(container.querySelector('img')).not.toBeInTheDocument()
    expect(screen.getByText('M')).toBeInTheDocument()
  })
})
