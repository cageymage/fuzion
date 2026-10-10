import { screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { renderWithProviders } from '../../testUtils'
import type { NewsPost } from '../../types/news'
import { NewsCard } from './NewsCard'

const post: NewsPost = {
  id: 'post-2',
  title: 'Now recruiting: Restoration Druid & Fire Mage',
  excerpt: 'Two raid spots open for the Mythic roster.',
  category: 'recruitment',
  imageUrl: null,
  authorName: 'Officer',
  publishedAt: '2026-09-13T18:00:00Z',
}

describe('NewsCard', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-09-17T20:00:00Z'))
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('should show the headline with its category chip', () => {
    renderWithProviders(<NewsCard post={post} />)

    expect(
      screen.getByRole('heading', { name: 'Now recruiting: Restoration Druid & Fire Mage' }),
    ).toBeInTheDocument()
    expect(screen.getByText('Recruitment')).toBeInTheDocument()
  })

  it('should show the author and how long ago the post was published', () => {
    renderWithProviders(<NewsCard post={post} />)

    expect(screen.getByText('Posted by Officer · 4 days ago')).toBeInTheDocument()
  })

  it('should leave out the excerpt so the compact card stays a single line of meta', () => {
    renderWithProviders(<NewsCard post={post} />)

    expect(screen.queryByText('Two raid spots open for the Mythic roster.')).not.toBeInTheDocument()
  })

  it('should link a draft to the editor and label it as a draft when the post is not published', () => {
    renderWithProviders(<NewsCard post={{ ...post, publishedAt: null }} />)

    expect(
      screen.getByRole('link', { name: 'Now recruiting: Restoration Druid & Fire Mage' }),
    ).toHaveAttribute('href', '/officer/news/post-2')
    expect(screen.getByText('Draft by Officer')).toBeInTheDocument()
  })

  it('should link the title to the post page', () => {
    renderWithProviders(<NewsCard post={post} />)

    expect(
      screen.getByRole('link', { name: 'Now recruiting: Restoration Druid & Fire Mage' }),
    ).toHaveAttribute('href', '/news/post-2')
  })
})
