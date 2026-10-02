import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { useLocation } from 'react-router-dom'
import { afterEach, describe, expect, it } from 'vitest'
import { newsHandler } from '../../mocks/handlers'
import { server } from '../../mocks/server'
import type { NewsCategory, NewsPost } from '../../types/news'
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

function makePosts(count: number, category: NewsCategory): NewsPost[] {
  return Array.from({ length: count }, (_, index) => ({
    id: `${category}-${index + 1}`,
    title: `${category} post ${String(index + 1).padStart(2, '0')}`,
    excerpt: 'Excerpt',
    category,
    imageUrl: null,
    authorName: 'Officer',
    publishedAt: new Date(Date.now() - (index + 1) * 3_600_000).toISOString(),
  }))
}

function LocationSearch() {
  return <p data-testid="location-search">{useLocation().search}</p>
}

function renderNews(route = '/news') {
  return renderWithProviders(
    <>
      <News />
      <LocationSearch />
    </>,
    route,
  )
}

const newsRequests: URL[] = []

function recordNewsRequests() {
  server.events.on('request:start', ({ request }) => {
    const url = new URL(request.url)
    if (url.pathname === '/api/news') newsRequests.push(url)
  })
}

afterEach(() => {
  server.events.removeAllListeners()
  newsRequests.length = 0
})

describe('News', () => {
  it('should list every post newest first when the request succeeds', async () => {
    renderNews()

    const titles = (await screen.findAllByRole('heading', { level: 3 })).map(
      (heading) => heading.textContent,
    )
    expect(titles).toEqual([
      'Fuzion defeated Queen Ansurek on Mythic',
      'Now recruiting: Restoration Druid & Fire Mage',
      'Welcome our newest officers',
    ])
  })

  it('should request the first page of five posts without a category when the page opens', async () => {
    recordNewsRequests()

    renderNews()

    await screen.findByText('Welcome our newest officers')
    expect(newsRequests).toHaveLength(1)
    expect(newsRequests[0].searchParams.get('limit')).toBe('5')
    expect(newsRequests[0].searchParams.get('offset')).toBe('0')
    expect(newsRequests[0].searchParams.has('category')).toBe(false)
  })

  it('should show a loading message when the request is still in flight', () => {
    renderNews()

    expect(screen.getByText('Loading news…')).toBeInTheDocument()
  })

  it('should only show recruitment posts when the Recruitment filter is selected', async () => {
    const user = userEvent.setup()
    renderNews()
    await screen.findByText('Welcome our newest officers')

    await user.click(screen.getByRole('button', { name: 'Recruitment' }))

    expect(
      await screen.findByRole('heading', { name: 'Now recruiting: Restoration Druid & Fire Mage' }),
    ).toBeInTheDocument()
    expect(screen.queryByText('Welcome our newest officers')).not.toBeInTheDocument()
    expect(screen.queryByText('Fuzion defeated Queen Ansurek on Mythic')).not.toBeInTheDocument()
  })

  it('should show every post again when All is selected after a category filter', async () => {
    const user = userEvent.setup()
    renderNews()
    await screen.findByText('Welcome our newest officers')
    await user.click(screen.getByRole('button', { name: 'Recruitment' }))
    await screen.findByText('Now recruiting: Restoration Druid & Fire Mage')
    expect(screen.queryByText('Welcome our newest officers')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'All' }))

    expect(await screen.findAllByRole('heading', { level: 3 })).toHaveLength(3)
  })

  it('should mark only the selected filter as pressed when a category is selected', async () => {
    const user = userEvent.setup()
    renderNews()
    await screen.findByText('Welcome our newest officers')
    expect(screen.getByRole('button', { name: 'All' })).toHaveAttribute('aria-pressed', 'true')

    await user.click(screen.getByRole('button', { name: 'Guild News' }))

    expect(screen.getByRole('button', { name: 'Guild News' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
    expect(screen.getByRole('button', { name: 'All' })).toHaveAttribute('aria-pressed', 'false')
  })

  it('should show the next page of posts when Next is clicked', async () => {
    server.use(newsHandler(makePosts(25, 'recruitment')))
    const user = userEvent.setup()
    renderNews()
    await screen.findByText('recruitment post 01')
    expect(screen.getByText('Page 1 of 5')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Next' }))

    expect(await screen.findByText('recruitment post 06')).toBeInTheDocument()
    expect(screen.queryByText('recruitment post 01')).not.toBeInTheDocument()
    expect(screen.getByText('Page 2 of 5')).toBeInTheDocument()
    expect(screen.getByTestId('location-search')).toHaveTextContent('?page=2')
  })

  it('should show the previous page of posts when Previous is clicked', async () => {
    server.use(newsHandler(makePosts(25, 'recruitment')))
    const user = userEvent.setup()
    renderNews('/news?page=2')
    await screen.findByText('recruitment post 06')

    await user.click(screen.getByRole('button', { name: 'Previous' }))

    expect(await screen.findByText('recruitment post 01')).toBeInTheDocument()
    expect(screen.getByText('Page 1 of 5')).toBeInTheDocument()
    expect(screen.getByTestId('location-search')).toHaveTextContent(/^$/)
  })

  it('should request only that category from the server and return to the first page when a category filter is selected', async () => {
    server.use(newsHandler([...makePosts(12, 'recruitment'), ...makePosts(14, 'guild-news')]))
    recordNewsRequests()
    const user = userEvent.setup()
    renderNews('/news?page=3')
    await screen.findByText('Page 3 of 6')

    await user.click(screen.getByRole('button', { name: 'Recruitment' }))

    expect(await screen.findByText('Page 1 of 3')).toBeInTheDocument()
    const lastRequest = newsRequests[newsRequests.length - 1]
    expect(lastRequest.searchParams.get('category')).toBe('recruitment')
    expect(lastRequest.searchParams.get('offset')).toBe('0')
    expect(screen.getByTestId('location-search')).toHaveTextContent('?category=recruitment')
  })

  it('should restore the category and page when the URL contains them', async () => {
    server.use(newsHandler([...makePosts(12, 'recruitment'), ...makePosts(14, 'guild-news')]))

    renderNews('/news?category=recruitment&page=2')

    expect(await screen.findByText('recruitment post 06')).toBeInTheDocument()
    expect(screen.getByText('recruitment post 10')).toBeInTheDocument()
    expect(screen.getByText('Page 2 of 3')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Recruitment' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
  })

  it('should disable Previous when on the first page', async () => {
    server.use(newsHandler(makePosts(25, 'recruitment')))

    renderNews()

    await screen.findByText('Page 1 of 5')
    expect(screen.getByRole('button', { name: 'Previous' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Next' })).toBeEnabled()
  })

  it('should disable Next when on the last page', async () => {
    server.use(newsHandler(makePosts(25, 'recruitment')))

    renderNews('/news?page=5')

    await screen.findByText('Page 5 of 5')
    expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Previous' })).toBeEnabled()
  })

  it('should offer page sizes of 5, 10, 25 and 50 with 5 selected when the page opens', () => {
    renderNews()

    const pageSize = screen.getByRole('combobox', { name: 'Posts per page' })
    expect(pageSize).toHaveValue('5')
    expect(within(pageSize).getAllByRole('option').map((option) => option.textContent)).toEqual([
      '5',
      '10',
      '25',
      '50',
    ])
  })

  it('should request the chosen number of posts and return to the first page when a page size is selected', async () => {
    server.use(newsHandler(makePosts(40, 'recruitment')))
    recordNewsRequests()
    const user = userEvent.setup()
    renderNews('/news?page=3')
    await screen.findByText('Page 3 of 8')

    await user.selectOptions(screen.getByRole('combobox', { name: 'Posts per page' }), '10')

    expect(await screen.findByText('recruitment post 10')).toBeInTheDocument()
    expect(screen.getByText('Page 1 of 4')).toBeInTheDocument()
    const lastRequest = newsRequests[newsRequests.length - 1]
    expect(lastRequest.searchParams.get('limit')).toBe('10')
    expect(lastRequest.searchParams.get('offset')).toBe('0')
    expect(screen.getByTestId('location-search')).toHaveTextContent('?pageSize=10')
  })

  it('should restore the page size when the URL contains it', async () => {
    server.use(newsHandler(makePosts(60, 'recruitment')))

    renderNews('/news?pageSize=25')

    expect(await screen.findByText('recruitment post 25')).toBeInTheDocument()
    expect(screen.getByText('Page 1 of 3')).toBeInTheDocument()
    expect(screen.getByRole('combobox', { name: 'Posts per page' })).toHaveValue('25')
  })

  it('should use the default page size when the page size in the URL is not an offered size', async () => {
    server.use(newsHandler(makePosts(25, 'recruitment')))

    renderNews('/news?pageSize=7')

    expect(await screen.findByText('Page 1 of 5')).toBeInTheDocument()
    expect(screen.getByRole('combobox', { name: 'Posts per page' })).toHaveValue('5')
  })

  it('should keep the chosen page size when a category filter is selected', async () => {
    const user = userEvent.setup()
    renderNews('/news?pageSize=10')
    await screen.findByText('Welcome our newest officers')

    await user.click(screen.getByRole('button', { name: 'Recruitment' }))

    expect(screen.getByTestId('location-search')).toHaveTextContent(
      '?category=recruitment&pageSize=10',
    )
  })

  it('should show the first page when the page in the URL is not a positive integer', async () => {
    server.use(newsHandler(makePosts(25, 'recruitment')))

    renderNews('/news?page=abc')

    expect(await screen.findByText('Page 1 of 5')).toBeInTheDocument()
    expect(screen.getByText('recruitment post 01')).toBeInTheDocument()
  })

  it('should jump to the last page when the page in the URL is past the end', async () => {
    server.use(newsHandler(makePosts(25, 'recruitment')))

    renderNews('/news?page=9')

    expect(await screen.findByText('recruitment post 25')).toBeInTheDocument()
    expect(screen.getByText('Page 5 of 5')).toBeInTheDocument()
    expect(screen.getByTestId('location-search')).toHaveTextContent('?page=5')
  })

  it('should show every category when the category in the URL is unknown', async () => {
    renderNews('/news?category=patch-notes')

    expect(await screen.findAllByRole('heading', { level: 3 })).toHaveLength(3)
    expect(screen.getByRole('button', { name: 'All' })).toHaveAttribute('aria-pressed', 'true')
  })

  it('should show an empty message when no posts match the selected category', async () => {
    server.use(newsHandler([recruitmentPost]))
    const user = userEvent.setup()
    renderNews()
    await screen.findByText('Now recruiting: Holy Priest')

    await user.click(screen.getByRole('button', { name: 'Raid Progress' }))

    expect(await screen.findByText('No posts in this category yet.')).toBeInTheDocument()
  })

  it('should hide the paging controls when there are no posts', async () => {
    server.use(newsHandler([]))

    renderNews()

    await screen.findByText('No posts in this category yet.')
    expect(screen.queryByRole('button', { name: 'Previous' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Next' })).not.toBeInTheDocument()
  })

  it('should link to the officer news page when the visitor is an officer', async () => {
    server.use(
      http.get('/api/auth/me', () =>
        HttpResponse.json({ id: 'user-1', username: 'Officer', avatarUrl: null, isOfficer: true }),
      ),
    )

    renderNews()

    expect(await screen.findByRole('link', { name: 'Manage news' })).toHaveAttribute(
      'href',
      '/officer/news',
    )
  })

  it('should not link to the officer news page when the visitor is not an officer', async () => {
    server.use(
      http.get('/api/auth/me', () =>
        HttpResponse.json({ id: 'user-2', username: 'Member', avatarUrl: null, isOfficer: false }),
      ),
    )

    renderNews()

    await screen.findByText('Welcome our newest officers')
    expect(screen.queryByRole('link', { name: 'Manage news' })).not.toBeInTheDocument()
  })

  it('should show an error message when the news request fails', async () => {
    server.use(http.get('/api/news', () => new HttpResponse(null, { status: 500 })))

    renderNews()

    expect(await screen.findByText('News could not be loaded.')).toBeInTheDocument()
  })
})
