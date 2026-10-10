import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { NewsBackdrop } from './NewsBackdrop'

describe('NewsBackdrop', () => {
  it('should show the post image when the post has one', () => {
    const { container } = render(
      <NewsBackdrop category="recruitment" imageUrl="https://cdn.example/ansurek.jpg" />,
    )

    expect(container.querySelector('img')).toHaveAttribute('src', 'https://cdn.example/ansurek.jpg')
    expect(container.querySelector('source')).toBeNull()
  })

  it('should show the raid progress art when the category is raid progress and the post has no image', () => {
    const { container } = render(<NewsBackdrop category="raid-progress" imageUrl={null} />)

    expect(container.querySelector('img')?.getAttribute('src')).toContain('raid-progress')
    expect(container.querySelector('source')?.getAttribute('srcset')).toContain('raid-progress')
  })

  it('should show the recruitment art when the category is recruitment and imageUrl is omitted', () => {
    const { container } = render(<NewsBackdrop category="recruitment" />)

    expect(container.querySelector('img')?.getAttribute('src')).toContain('recruitment')
  })

  it('should show the guild news art when the category is guild news and the post has no image', () => {
    const { container } = render(<NewsBackdrop category="guild-news" imageUrl={null} />)

    expect(container.querySelector('img')?.getAttribute('src')).toContain('guild-news')
  })

  it('should show the patch notes art when the category is patch notes and the post has no image', () => {
    const { container } = render(<NewsBackdrop category="patch-notes" imageUrl={null} />)

    expect(container.querySelector('img')?.getAttribute('src')).toContain('patch-notes')
  })

  it('should hide the artwork from assistive technology', () => {
    const { container } = render(<NewsBackdrop category="guild-news" />)

    expect(container.firstElementChild).toHaveAttribute('aria-hidden', 'true')
    expect(container.querySelector('img')).toHaveAttribute('alt', '')
  })
})
