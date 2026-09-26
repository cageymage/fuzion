import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '../../mocks/server'
import type { NewsPost } from '../../types/news'
import { renderWithProviders } from '../../testUtils'
import { News } from './News'

const recruitmentPost: NewsPost = {
  id: 'post-recruit',
  title: 'Now recruiting: Holy Priest',
  excerpt: 'One healer spot open.',
  category: 'recruitment',
  imageUrl: null,
  authorName: 'Officer',
  publishedAt: new Date(Date.now() - 86_400_000).toISOString(),
}

describe('News', () => {
  it('should list every post newest first when the request succeeds', async () => {
    renderWithProviders(<News />)

    const titles = (await screen.findAllByRole('heading', { level: 3 })).map(
      (heading) => heading.textContent,
    )
    expect(titles).toEqual([
      'Fuzion defeated Queen Ansurek on Mythic',
      'Now recruiting: Restoration Druid & Fire Mage',
      'Welcome our newest officers',
    ])
  })

  it('should show a loading message when the request is still in flight', () => {
    renderWithProviders(<News />)

    expect(screen.getByText('Loading news…')).toBeInTheDocument()
  })

  it('should only show recruitment posts when the Recruitment filter is selected', async () => {
    const user = userEvent.setup()
    renderWithProviders(<News />)
    await screen.findByText('Welcome our newest officers')

    await user.click(screen.getByRole('button', { name: 'Recruitment' }))

    expect(
      screen.getByRole('heading', { name: 'Now recruiting: Restoration Druid & Fire Mage' }),
    ).toBeInTheDocument()
    expect(screen.queryByText('Welcome our newest officers')).not.toBeInTheDocument()
    expect(screen.queryByText('Fuzion defeated Queen Ansurek on Mythic')).not.toBeInTheDocument()
  })

  it('should show every post again when All is selected after a category filter', async () => {
    const user = userEvent.setup()
    renderWithProviders(<News />)
    await screen.findByText('Welcome our newest officers')
    await user.click(screen.getByRole('button', { name: 'Recruitment' }))

    await user.click(screen.getByRole('button', { name: 'All' }))

    expect(screen.getAllByRole('heading', { level: 3 })).toHaveLength(3)
  })

  it('should mark only the selected filter as pressed when a category is selected', async () => {
    const user = userEvent.setup()
    renderWithProviders(<News />)
    await screen.findByText('Welcome our newest officers')
    expect(screen.getByRole('button', { name: 'All' })).toHaveAttribute('aria-pressed', 'true')

    await user.click(screen.getByRole('button', { name: 'Guild News' }))

    expect(screen.getByRole('button', { name: 'Guild News' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
    expect(screen.getByRole('button', { name: 'All' })).toHaveAttribute('aria-pressed', 'false')
  })

  it('should show an empty message when no posts match the selected category', async () => {
    server.use(http.get('/api/news', () => HttpResponse.json([recruitmentPost])))
    const user = userEvent.setup()
    renderWithProviders(<News />)
    await screen.findByText('Now recruiting: Holy Priest')

    await user.click(screen.getByRole('button', { name: 'Raid Progress' }))

    expect(screen.getByText('No posts in this category yet.')).toBeInTheDocument()
  })

  it('should show an error message when the news request fails', async () => {
    server.use(http.get('/api/news', () => new HttpResponse(null, { status: 500 })))

    renderWithProviders(<News />)

    expect(await screen.findByText('News could not be loaded.')).toBeInTheDocument()
  })
})
