import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { Route, Routes } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { draftPost } from '../../mocks/handlers'
import { server } from '../../mocks/server'
import { renderWithProviders } from '../../testUtils'
import { OfficerNews } from './OfficerNews'

const officer = { id: 'user-1', username: 'Officer', avatarUrl: null, isOfficer: true }
const member = { id: 'user-2', username: 'Member', avatarUrl: null, isOfficer: false }

function signInAs(user: typeof officer) {
  server.use(http.get('/api/auth/me', () => HttpResponse.json(user)))
}

function renderList() {
  return renderWithProviders(
    <Routes>
      <Route path="/officer/news" element={<OfficerNews />} />
      <Route path="/officer/news/:id" element={<p>Editor for the post</p>} />
      <Route path="/" element={<p>Home page</p>} />
    </Routes>,
    '/officer/news',
  )
}

describe('OfficerNews', () => {
  it('should list drafts and published posts with links to the editor when the visitor is an officer', async () => {
    signInAs(officer)

    renderList()

    expect(await screen.findByRole('link', { name: 'Patch 11.0 notes' })).toHaveAttribute(
      'href',
      '/officer/news/draft-1',
    )
    expect(
      await screen.findByRole('link', { name: 'Fuzion defeated Queen Ansurek on Mythic' }),
    ).toHaveAttribute('href', '/officer/news/post-1')
  })

  it('should show an empty message when there are no drafts', async () => {
    signInAs(officer)
    server.use(http.get('/api/news/drafts', () => HttpResponse.json([])))

    renderList()

    expect(await screen.findByText('No drafts.')).toBeInTheDocument()
  })

  it('should open the editor for a new draft when New post is clicked', async () => {
    signInAs(officer)
    const user = userEvent.setup()
    let created: unknown
    server.use(
      http.post('/api/news', async ({ request }) => {
        created = await request.json()
        return HttpResponse.json(draftPost, { status: 201 })
      }),
    )
    renderList()

    await user.click(await screen.findByRole('button', { name: 'New post' }))

    expect(await screen.findByText('Editor for the post')).toBeInTheDocument()
    expect(created).toEqual({
      title: 'Untitled post',
      excerpt: '',
      category: 'guild-news',
      body: '',
      pinned: false,
    })
  })

  it('should show an error message when the new post cannot be created', async () => {
    signInAs(officer)
    const user = userEvent.setup()
    server.use(http.post('/api/news', () => new HttpResponse(null, { status: 500 })))
    renderList()

    await user.click(await screen.findByRole('button', { name: 'New post' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('The new post could not be created.')
  })

  it('should show an error message when the drafts cannot be loaded', async () => {
    signInAs(officer)
    server.use(http.get('/api/news/drafts', () => new HttpResponse(null, { status: 500 })))

    renderList()

    expect(await screen.findByText('Drafts could not be loaded.')).toBeInTheDocument()
  })

  it('should redirect to the home page when the visitor is not an officer', async () => {
    signInAs(member)

    renderList()

    expect(await screen.findByText('Home page')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'New post' })).not.toBeInTheDocument()
  })
})
