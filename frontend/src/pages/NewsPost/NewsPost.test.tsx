import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { Route, Routes } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { server } from '../../mocks/server'
import { renderWithProviders } from '../../testUtils'
import type { NewsPostDetail } from '../../types/news'
import { NewsPostPage } from './NewsPost'

const post: NewsPostDetail = {
  id: 'post-1',
  title: 'Fuzion defeated Queen Ansurek on Mythic',
  excerpt: '8/8 down.',
  category: 'raid-progress',
  imageUrl: null,
  authorName: 'Officer',
  publishedAt: new Date(Date.now() - 2 * 86_400_000).toISOString(),
  body: '## The kill\n\nWe **finally** did it. See [the log](https://logs.example/ansurek).\n\n- Tanks held\n- Healers healed\n\n![Kill screenshot](https://cdn.example/kill.png)',
}

function servePost(detail: NewsPostDetail) {
  server.use(http.get('/api/news/:id', () => HttpResponse.json(detail)))
}

const officer = { id: 'user-1', username: 'Officer', avatarUrl: null, isOfficer: true }
const member = { id: 'user-2', username: 'Member', avatarUrl: null, isOfficer: false }

function signInAs(user: typeof officer) {
  server.use(http.get('/api/auth/me', () => HttpResponse.json(user)))
}

function renderPostPage() {
  return renderWithProviders(
    <Routes>
      <Route path="/news/:id" element={<NewsPostPage />} />
      <Route path="/news" element={<p>News page</p>} />
    </Routes>,
    '/news/post-1',
  )
}

describe('NewsPostPage', () => {
  it('should show the title, category, author and a link back to all news when the post loads', async () => {
    servePost(post)

    renderPostPage()

    expect(
      await screen.findByRole('heading', { level: 1, name: 'Fuzion defeated Queen Ansurek on Mythic' }),
    ).toBeInTheDocument()
    expect(screen.getByText('Raid Progress')).toBeInTheDocument()
    expect(screen.getByText(/Posted by Officer/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Back to all news' })).toHaveAttribute('href', '/news')
  })

  it('should render the post body as HTML from Markdown', async () => {
    servePost(post)

    renderPostPage()

    expect(await screen.findByRole('heading', { level: 2, name: 'The kill' })).toBeInTheDocument()
    expect(screen.getByText('finally').tagName).toBe('STRONG')
    expect(screen.getByRole('link', { name: 'the log' })).toHaveAttribute(
      'href',
      'https://logs.example/ansurek',
    )
    expect(screen.getAllByRole('listitem').map((item) => item.textContent)).toEqual([
      'Tanks held',
      'Healers healed',
    ])
  })

  it('should not inject raw HTML from the post body', async () => {
    servePost({ ...post, body: 'Before <b>bold html</b> <script>window.hacked = true</script> after' })

    const { container } = renderPostPage()

    await screen.findByRole('heading', { level: 1 })
    expect(container.querySelector('b')).toBeNull()
    expect(container.querySelector('script')).toBeNull()
    expect((window as unknown as { hacked?: boolean }).hacked).toBeUndefined()
  })

  it('should open the lightbox when an image is clicked', async () => {
    servePost(post)
    const user = userEvent.setup()
    renderPostPage()

    await user.click(await screen.findByRole('button', { name: 'Open image: Kill screenshot' }))

    expect(screen.getByRole('dialog', { name: 'Image preview' })).toBeInTheDocument()
  })

  it('should close the lightbox when Escape is pressed', async () => {
    servePost(post)
    const user = userEvent.setup()
    renderPostPage()
    await user.click(await screen.findByRole('button', { name: 'Open image: Kill screenshot' }))

    await user.keyboard('{Escape}')

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('should link to the editor when the visitor is an officer', async () => {
    servePost(post)
    signInAs(officer)

    renderPostPage()

    expect(await screen.findByRole('link', { name: 'Edit' })).toHaveAttribute(
      'href',
      '/officer/news/post-1',
    )
  })

  it('should not offer Edit or Delete when the visitor is not an officer', async () => {
    servePost(post)
    signInAs(member)

    renderPostPage()

    await screen.findByRole('heading', { level: 1 })
    expect(screen.queryByRole('link', { name: 'Edit' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Delete' })).not.toBeInTheDocument()
  })

  it('should delete the post and return to the news page when an officer confirms deletion', async () => {
    servePost(post)
    signInAs(officer)
    const deletedIds: string[] = []
    server.use(
      http.delete('/api/news/:id', ({ params }) => {
        deletedIds.push(String(params.id))
        return new HttpResponse(null, { status: 204 })
      }),
    )
    const user = userEvent.setup()
    renderPostPage()

    await user.click(await screen.findByRole('button', { name: 'Delete' }))
    await user.click(screen.getByRole('button', { name: 'Confirm delete' }))

    expect(await screen.findByText('News page')).toBeInTheDocument()
    expect(deletedIds).toEqual(['post-1'])
  })

  it('should show a loading message when the request is still in flight', () => {
    servePost(post)

    renderPostPage()

    expect(screen.getByText('Loading post…')).toBeInTheDocument()
  })

  it('should show a not-found message when the post does not exist', async () => {
    server.use(
      http.get('/api/news/:id', () => HttpResponse.json({ error: 'news post not found' }, { status: 404 })),
    )

    renderPostPage()

    expect(await screen.findByText('Post not found.')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Back to all news' })).toBeInTheDocument()
  })

  it('should show an error message when the request fails', async () => {
    server.use(http.get('/api/news/:id', () => new HttpResponse(null, { status: 500 })))

    renderPostPage()

    expect(await screen.findByText('Post could not be loaded.')).toBeInTheDocument()
    expect(screen.queryByText('Post not found.')).not.toBeInTheDocument()
  })
})
