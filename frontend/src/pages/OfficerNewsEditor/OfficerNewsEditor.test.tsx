import { act, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { Link, Route, Routes } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { draftPost } from '../../mocks/handlers'
import { server } from '../../mocks/server'
import { renderWithProviders } from '../../testUtils'
import { OfficerNewsEditor } from './OfficerNewsEditor'

const officer = { id: 'user-1', username: 'Officer', avatarUrl: null, isOfficer: true }
const member = { id: 'user-2', username: 'Member', avatarUrl: null, isOfficer: false }

function signInAs(user: typeof officer) {
  server.use(http.get('/api/auth/me', () => HttpResponse.json(user)))
}

function renderEditor(route = '/officer/news/draft-1') {
  return renderWithProviders(
    <Routes>
      <Route path="/officer/news/:id" element={<OfficerNewsEditor />} />
      <Route path="/" element={<p>Home page</p>} />
    </Routes>,
    route,
  )
}

function renderEditorWithHomeLink() {
  return renderWithProviders(
    <>
      <Link to="/">Go home</Link>
      <Routes>
        <Route path="/officer/news/:id" element={<OfficerNewsEditor />} />
        <Route path="/" element={<p>Home page</p>} />
      </Routes>
    </>,
    '/officer/news/draft-1',
  )
}

function recordPatches() {
  const patches: unknown[] = []
  server.use(
    http.patch('/api/news/:id', async ({ request }) => {
      patches.push(await request.json())
      return HttpResponse.json(draftPost)
    }),
  )
  return patches
}

afterEach(() => {
  vi.useRealTimers()
})

describe('OfficerNewsEditor', () => {
  it('should autosave the draft after the user stops typing', async () => {
    signInAs(officer)
    const patches = recordPatches()
    vi.useFakeTimers({ shouldAdvanceTime: true })
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })
    renderEditor()
    const body = await screen.findByRole('textbox', { name: 'Body' })

    await user.type(body, ' and more')
    expect(patches).toHaveLength(0)
    await act(async () => {
      vi.advanceTimersByTime(2000)
    })

    await waitFor(() => expect(patches).toHaveLength(1))
    expect(patches[0]).toEqual({
      title: 'Patch 11.0 notes',
      excerpt: 'What changed.',
      category: 'patch-notes',
      body: '## Changes and more',
      pinned: false,
    })
  })

  it('should wait for typing to pause before saving when the user keeps typing', async () => {
    signInAs(officer)
    const patches = recordPatches()
    vi.useFakeTimers({ shouldAdvanceTime: true })
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })
    renderEditor()
    const title = await screen.findByRole('textbox', { name: 'Title' })

    await user.type(title, ' a')
    await act(async () => {
      vi.advanceTimersByTime(1500)
    })
    await user.type(title, 'b')
    await act(async () => {
      vi.advanceTimersByTime(1500)
    })

    expect(patches).toHaveLength(0)
  })

  it('should show a saved status after a successful autosave', async () => {
    signInAs(officer)
    recordPatches()
    vi.useFakeTimers({ shouldAdvanceTime: true })
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })
    renderEditor()

    await user.type(await screen.findByRole('textbox', { name: 'Title' }), '!')
    await act(async () => {
      vi.advanceTimersByTime(2000)
    })

    expect(await screen.findByText(/^Saved \d\d:\d\d:\d\d$/)).toBeInTheDocument()
  })

  it('should show a save-failed status when the autosave request fails', async () => {
    signInAs(officer)
    server.use(
      http.patch('/api/news/:id', () =>
        HttpResponse.json({ error: 'title: must be 1 to 120 characters' }, { status: 400 }),
      ),
    )
    vi.useFakeTimers({ shouldAdvanceTime: true })
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })
    renderEditor()

    await user.type(await screen.findByRole('textbox', { name: 'Title' }), '!')
    await act(async () => {
      vi.advanceTimersByTime(2000)
    })

    expect(await screen.findByText('Save failed')).toBeInTheDocument()
  })

  it('should call the publish endpoint when Publish is confirmed', async () => {
    signInAs(officer)
    const published: string[] = []
    server.use(
      http.post('/api/news/:id/publish', ({ params }) => {
        published.push(String(params.id))
        return HttpResponse.json({ ...draftPost, publishedAt: new Date().toISOString() })
      }),
    )
    const user = userEvent.setup()
    renderEditor()

    await user.click(await screen.findByRole('button', { name: 'Publish' }))
    await user.click(screen.getByRole('button', { name: 'Confirm publish' }))

    expect(await screen.findByText('Published')).toBeInTheDocument()
    expect(published).toEqual(['draft-1'])
  })

  it('should not call the publish endpoint when the publish confirmation is cancelled', async () => {
    signInAs(officer)
    const published: string[] = []
    server.use(
      http.post('/api/news/:id/publish', ({ params }) => {
        published.push(String(params.id))
        return HttpResponse.json(draftPost)
      }),
    )
    const user = userEvent.setup()
    renderEditor()

    await user.click(await screen.findByRole('button', { name: 'Publish' }))
    await user.click(screen.getByRole('button', { name: 'Cancel' }))

    expect(screen.getByRole('button', { name: 'Publish' })).toBeInTheDocument()
    expect(published).toEqual([])
  })

  it('should save unsaved edits before publishing when Publish is confirmed', async () => {
    signInAs(officer)
    const patches = recordPatches()
    server.use(
      http.post('/api/news/:id/publish', () =>
        HttpResponse.json({ ...draftPost, publishedAt: new Date().toISOString() }),
      ),
    )
    const user = userEvent.setup()
    renderEditor()
    await user.type(await screen.findByRole('textbox', { name: 'Title' }), '!')

    await user.click(screen.getByRole('button', { name: 'Publish' }))
    await user.click(screen.getByRole('button', { name: 'Confirm publish' }))

    expect(await screen.findByText('Published')).toBeInTheDocument()
    expect(patches).toHaveLength(1)
  })

  it('should show the post fields when the post is already published', async () => {
    signInAs(officer)

    renderEditor('/officer/news/post-1')

    expect(await screen.findByRole('textbox', { name: 'Title' })).toHaveValue(
      'Fuzion defeated Queen Ansurek on Mythic',
    )
    expect(screen.getByText('Published')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Publish' })).not.toBeInTheDocument()
  })

  it('should insert a Markdown image into the body when an image URL is entered', async () => {
    signInAs(officer)
    vi.spyOn(window, 'prompt').mockReturnValue('https://cdn.example/boss.png')
    const user = userEvent.setup()
    renderEditor()
    const body = await screen.findByRole('textbox', { name: 'Body' })

    await user.click(screen.getByRole('button', { name: 'Insert image' }))

    await waitFor(() => expect(body).toHaveValue('![](https://cdn.example/boss.png)## Changes'))
  })

  it('should redirect to the home page when the visitor is not an officer', async () => {
    signInAs(member)

    renderEditor()

    expect(await screen.findByText('Home page')).toBeInTheDocument()
    expect(screen.queryByRole('textbox', { name: 'Title' })).not.toBeInTheDocument()
  })

  it('should redirect to the home page when the visitor is anonymous', async () => {
    renderEditor()

    expect(await screen.findByText('Home page')).toBeInTheDocument()
  })

  it('should block in-app navigation when there are unsaved changes', async () => {
    signInAs(officer)
    const user = userEvent.setup()
    renderEditorWithHomeLink()
    await user.type(await screen.findByRole('textbox', { name: 'Title' }), '!')

    await user.click(screen.getByRole('link', { name: 'Go home' }))

    expect(await screen.findByRole('alertdialog', { name: 'Unsaved changes' })).toBeInTheDocument()
    expect(screen.queryByText('Home page')).not.toBeInTheDocument()
    expect(screen.getByRole('textbox', { name: 'Title' })).toHaveValue('Patch 11.0 notes!')
  })

  it('should stay on the editor when Stay on this page is chosen', async () => {
    signInAs(officer)
    const user = userEvent.setup()
    renderEditorWithHomeLink()
    await user.type(await screen.findByRole('textbox', { name: 'Title' }), '!')
    await user.click(screen.getByRole('link', { name: 'Go home' }))

    await user.click(await screen.findByRole('button', { name: 'Stay on this page' }))

    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    expect(screen.getByRole('textbox', { name: 'Title' })).toHaveValue('Patch 11.0 notes!')
  })

  it('should show the autosaved text when the editor is reopened after leaving', async () => {
    signInAs(officer)
    let stored = draftPost
    server.use(
      http.get('/api/news/drafts', () => HttpResponse.json([stored])),
      http.patch('/api/news/:id', async ({ request }) => {
        stored = { ...stored, ...((await request.json()) as object) }
        return HttpResponse.json(stored)
      }),
    )
    vi.useFakeTimers({ shouldAdvanceTime: true })
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })
    renderWithProviders(
      <>
        <Link to="/">Go home</Link>
        <Routes>
          <Route path="/officer/news/:id" element={<OfficerNewsEditor />} />
          <Route path="/" element={<Link to="/officer/news/draft-1">Reopen editor</Link>} />
        </Routes>
      </>,
      '/officer/news/draft-1',
    )
    await user.type(await screen.findByRole('textbox', { name: 'Title' }), '!')
    await act(async () => {
      vi.advanceTimersByTime(2000)
    })
    await screen.findByText(/^Saved /)
    await user.click(screen.getByRole('link', { name: 'Go home' }))

    await user.click(await screen.findByRole('link', { name: 'Reopen editor' }))

    expect(await screen.findByRole('textbox', { name: 'Title' })).toHaveValue('Patch 11.0 notes!')
  })

  it('should block navigation again when the user chose to stay and then tries to leave', async () => {
    signInAs(officer)
    const user = userEvent.setup()
    renderEditorWithHomeLink()
    await user.type(await screen.findByRole('textbox', { name: 'Title' }), '!')
    await user.click(screen.getByRole('link', { name: 'Go home' }))
    await user.click(await screen.findByRole('button', { name: 'Stay on this page' }))

    await user.click(screen.getByRole('link', { name: 'Go home' }))

    expect(await screen.findByRole('alertdialog', { name: 'Unsaved changes' })).toBeInTheDocument()
    expect(screen.queryByText('Home page')).not.toBeInTheDocument()
  })

  it('should focus Stay on this page when the unsaved-changes dialog opens', async () => {
    signInAs(officer)
    const user = userEvent.setup()
    renderEditorWithHomeLink()
    await user.type(await screen.findByRole('textbox', { name: 'Title' }), '!')

    await user.click(screen.getByRole('link', { name: 'Go home' }))

    expect(await screen.findByRole('button', { name: 'Stay on this page' })).toHaveFocus()
  })

  it('should stay on the editor when Escape is pressed in the unsaved-changes dialog', async () => {
    signInAs(officer)
    const user = userEvent.setup()
    renderEditorWithHomeLink()
    await user.type(await screen.findByRole('textbox', { name: 'Title' }), '!')
    await user.click(screen.getByRole('link', { name: 'Go home' }))
    await screen.findByRole('alertdialog', { name: 'Unsaved changes' })

    await user.keyboard('{Escape}')

    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    expect(screen.queryByText('Home page')).not.toBeInTheDocument()
  })

  it('should leave the editor when Leave without saving is chosen', async () => {
    signInAs(officer)
    const user = userEvent.setup()
    renderEditorWithHomeLink()
    await user.type(await screen.findByRole('textbox', { name: 'Title' }), '!')
    await user.click(screen.getByRole('link', { name: 'Go home' }))

    await user.click(await screen.findByRole('button', { name: 'Leave without saving' }))

    expect(await screen.findByText('Home page')).toBeInTheDocument()
  })

  it('should navigate freely when there are no unsaved changes', async () => {
    signInAs(officer)
    const user = userEvent.setup()
    renderEditorWithHomeLink()
    await screen.findByRole('textbox', { name: 'Title' })

    await user.click(screen.getByRole('link', { name: 'Go home' }))

    expect(await screen.findByText('Home page')).toBeInTheDocument()
  })

  it('should navigate freely when the changes have been autosaved', async () => {
    signInAs(officer)
    recordPatches()
    vi.useFakeTimers({ shouldAdvanceTime: true })
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })
    renderEditorWithHomeLink()
    await user.type(await screen.findByRole('textbox', { name: 'Title' }), '!')
    await act(async () => {
      vi.advanceTimersByTime(2000)
    })
    await screen.findByText(/^Saved /)

    await user.click(screen.getByRole('link', { name: 'Go home' }))

    expect(await screen.findByText('Home page')).toBeInTheDocument()
  })

  it('should ask the browser to confirm closing the tab when there are unsaved changes', async () => {
    signInAs(officer)
    const user = userEvent.setup()
    renderEditor()
    await user.type(await screen.findByRole('textbox', { name: 'Title' }), '!')

    const closing = new Event('beforeunload', { cancelable: true })
    window.dispatchEvent(closing)

    expect(closing.defaultPrevented).toBe(true)
  })

  it('should not ask the browser to confirm closing the tab when there are no unsaved changes', async () => {
    signInAs(officer)
    renderEditor()
    await screen.findByRole('textbox', { name: 'Title' })

    const closing = new Event('beforeunload', { cancelable: true })
    window.dispatchEvent(closing)

    expect(closing.defaultPrevented).toBe(false)
  })

  it('should show an error message when the post cannot be loaded', async () => {
    signInAs(officer)
    server.use(http.get('/api/news/drafts', () => new HttpResponse(null, { status: 500 })))

    renderEditor()

    expect(await screen.findByText('Post could not be loaded.')).toBeInTheDocument()
  })
})
