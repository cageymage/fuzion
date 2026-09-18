import { render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { NewsPost } from '../../types/news'
import { FeaturedNewsCard } from './FeaturedNewsCard'

const post: NewsPost = {
  id: 'post-1',
  title: 'Fuzion defeated Queen Ansurek on Mythic',
  excerpt: "Mythic Nerub'ar Palace, 8/8 down — a huge night for the team after weeks on the enrage.",
  category: 'raid-progress',
  imageUrl: null,
  authorName: 'Officer',
  publishedAt: '2026-09-15T18:00:00Z',
}

describe('FeaturedNewsCard', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-09-17T20:00:00Z'))
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('should show the headline, excerpt and category label', () => {
    render(<FeaturedNewsCard post={post} />)

    expect(
      screen.getByRole('heading', { name: 'Fuzion defeated Queen Ansurek on Mythic' }),
    ).toBeInTheDocument()
    expect(screen.getByText(/a huge night for the team/)).toBeInTheDocument()
    expect(screen.getByText('Raid Progress')).toBeInTheDocument()
  })

  it('should show the author and how long ago the post was published', () => {
    render(<FeaturedNewsCard post={post} />)

    expect(screen.getByText('Posted by Officer · 2 days ago')).toBeInTheDocument()
  })

  it('should render the post image when the post has one', () => {
    const { container } = render(
      <FeaturedNewsCard post={{ ...post, imageUrl: 'https://cdn.example/ansurek.jpg' }} />,
    )

    expect(container.querySelector('img')).toHaveAttribute('src', 'https://cdn.example/ansurek.jpg')
  })
})
