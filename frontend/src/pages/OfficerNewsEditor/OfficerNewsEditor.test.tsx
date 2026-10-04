import { act, fireEvent, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { Link, Route, Routes } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { draftPost, newsPage } from '../../mocks/handlers'
import { News } from '../News/News'
import { OfficerNews } from '../OfficerNews/OfficerNews'
import { ToastProvider } from '../../components/Toast/Toast'
import type { NewsPost } from '../../types/news'
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
    <ToastProvider>
      <Routes>
        <Route path="/officer/news/:id" element={<OfficerNewsEditor />} />
        <Route path="/" element={<p>Home page</p>} />
        <Route path="/news/:id" element={<p>Post page</p>} />
      </Routes>
    </ToastProvider>,
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
        <Route path="/news/:id" element={<p>Post page</p>} />
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

function pngFile() {
  return new File([new Uint8Array([137, 80, 78, 71])], 'kill.png', { type: 'image/png' })
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

    expect(await screen.findByText('Post page')).toBeInTheDocument()
    expect(published).toEqual(['draft-1'])
  })

  it('should open the published post page when Publish is confirmed', async () => {
    signInAs(officer)
    server.use(
      http.post('/api/news/:id/publish', () =>
        HttpResponse.json({ ...draftPost, publishedAt: new Date().toISOString() }),
      ),
    )
    const user = userEvent.setup()
    renderEditor()

    await user.click(await screen.findByRole('button', { name: 'Publish' }))
    await user.click(screen.getByRole('button', { name: 'Confirm publish' }))

    expect(await screen.findByText('Post page')).toBeInTheDocument()
    expect(screen.queryByRole('textbox', { name: 'Title' })).not.toBeInTheDocument()
  })

  it('should show a success toast when Publish is confirmed', async () => {
    signInAs(officer)
    server.use(
      http.post('/api/news/:id/publish', () =>
        HttpResponse.json({ ...draftPost, publishedAt: new Date().toISOString() }),
      ),
    )
    const user = userEvent.setup()
    renderEditor()

    await user.click(await screen.findByRole('button', { name: 'Publish' }))
    await user.click(screen.getByRole('button', { name: 'Confirm publish' }))

    expect(await screen.findByRole('status')).toHaveTextContent('Post published')
  })

  it('should stay on the editor without a toast when publishing fails', async () => {
    signInAs(officer)
    server.use(
      http.post('/api/news/:id/publish', () =>
        HttpResponse.json({ error: 'news post not found' }, { status: 404 }),
      ),
    )
    const user = userEvent.setup()
    renderEditor()

    await user.click(await screen.findByRole('button', { name: 'Publish' }))
    await user.click(screen.getByRole('button', { name: 'Confirm publish' }))

    expect(await screen.findByRole('alert')).toBeInTheDocument()
    expect(screen.getByRole('textbox', { name: 'Title' })).toBeInTheDocument()
    expect(screen.queryByText('Post published')).not.toBeInTheDocument()
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

    expect(await screen.findByText('Post page')).toBeInTheDocument()
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

  it('should insert a markdown image with the pasted url when an officer chooses the from-URL option', async () => {
    signInAs(officer)
    vi.spyOn(window, 'prompt').mockReturnValue('https://cdn.example/boss.png')
    const user = userEvent.setup()
    renderEditor()
    const body = await screen.findByRole('textbox', { name: 'Body' })

    await user.click(screen.getByRole('button', { name: 'Insert image' }))
    await user.click(screen.getByRole('button', { name: 'From URL' }))

    await waitFor(() => expect(body).toHaveValue('![](https://cdn.example/boss.png)## Changes'))
  })

  it('should reject the image url when it does not start with https', async () => {
    signInAs(officer)
    vi.spyOn(window, 'prompt').mockReturnValue('javascript:alert(1)')
    const user = userEvent.setup()
    renderEditor()
    const body = await screen.findByRole('textbox', { name: 'Body' })

    await user.click(screen.getByRole('button', { name: 'Insert image' }))
    await user.click(screen.getByRole('button', { name: 'From URL' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Image URL must start with https://')
    expect(body).toHaveValue('## Changes')
  })

  it('should insert the uploaded image url at the cursor when an officer picks a file', async () => {
    signInAs(officer)
    server.use(
      http.post('/api/images', () =>
        HttpResponse.json({ id: 'img-1', url: '/api/images/img-1' }, { status: 201 }),
      ),
    )
    const user = userEvent.setup()
    renderEditor()
    const body = await screen.findByRole('textbox', { name: 'Body' })

    await user.click(screen.getByRole('button', { name: 'Insert image' }))
    await user.click(screen.getByRole('button', { name: 'Upload image' }))
    await user.upload(screen.getByLabelText('Image file'), pngFile())

    await waitFor(() => expect(body).toHaveValue('![](/api/images/img-1)## Changes'))
  })

  it('should insert the uploaded image url when an officer pastes an image from the clipboard', async () => {
    signInAs(officer)
    server.use(
      http.post('/api/images', () =>
        HttpResponse.json({ id: 'img-2', url: '/api/images/img-2' }, { status: 201 }),
      ),
    )
    const user = userEvent.setup()
    renderEditor()
    const body = await screen.findByRole('textbox', { name: 'Body' })

    await user.click(body)
    fireEvent.paste(body, { clipboardData: { files: [pngFile()], types: ['Files'] } })

    await waitFor(() => expect(body).toHaveValue('## Changes![](/api/images/img-2)'))
  })

  it('should accept a dragged image file when it is held over the editor', async () => {
    signInAs(officer)
    renderEditor()
    await screen.findByRole('textbox', { name: 'Body' })

    const allowed = fireEvent.dragOver(screen.getByTestId('body-editor'), {
      dataTransfer: { files: [pngFile()], types: ['Files'] },
    })

    expect(allowed).toBe(false)
  })

  it('should insert the uploaded image url when an officer drops an image file on the editor', async () => {
    signInAs(officer)
    server.use(
      http.post('/api/images', () =>
        HttpResponse.json({ id: 'img-4', url: '/api/images/img-4' }, { status: 201 }),
      ),
    )
    renderEditor()
    const body = await screen.findByRole('textbox', { name: 'Body' })

    fireEvent.drop(screen.getByTestId('body-editor'), {
      dataTransfer: { files: [pngFile()], types: ['Files'] },
    })

    await waitFor(() => expect(body).toHaveValue('![](/api/images/img-4)## Changes'))
  })

  it('should show an upload error when the server rejects the file', async () => {
    signInAs(officer)
    server.use(
      http.post('/api/images', () =>
        HttpResponse.json({ error: 'file: must be at most 10 MiB' }, { status: 413 }),
      ),
    )
    const user = userEvent.setup()
    renderEditor()
    const body = await screen.findByRole('textbox', { name: 'Body' })

    await user.click(screen.getByRole('button', { name: 'Insert image' }))
    await user.click(screen.getByRole('button', { name: 'Upload image' }))
    await user.upload(screen.getByLabelText('Image file'), pngFile())

    expect(await screen.findByRole('alert')).toHaveTextContent('file: must be at most 10 MiB')
    expect(body).toHaveValue('## Changes')
  })

  it('should disable the insert action while an upload is in progress', async () => {
    signInAs(officer)
    let finishUpload: () => void = () => {}
    server.use(
      http.post('/api/images', async () => {
        await new Promise<void>((resolve) => {
          finishUpload = resolve
        })
        return HttpResponse.json({ id: 'img-3', url: '/api/images/img-3' }, { status: 201 })
      }),
    )
    const user = userEvent.setup()
    renderEditor()
    await screen.findByRole('textbox', { name: 'Body' })

    await user.click(screen.getByRole('button', { name: 'Insert image' }))
    await user.click(screen.getByRole('button', { name: 'Upload image' }))
    await user.upload(screen.getByLabelText('Image file'), pngFile())

    expect(await screen.findByRole('button', { name: 'Insert image' })).toBeDisabled()
    expect(screen.getByText('Uploading image…')).toBeInTheDocument()
    finishUpload()
    await waitFor(() => expect(screen.getByRole('button', { name: 'Insert image' })).toBeEnabled())
  })

  it('should show images in the live preview as lightbox thumbnails', async () => {
    signInAs(officer)
    server.use(
      http.get('/api/news/drafts', () =>
        HttpResponse.json([{ ...draftPost, body: '![Boss kill](https://cdn.example/boss.png)' }]),
      ),
    )

    renderEditor()

    expect(await screen.findByRole('button', { name: 'Open image: Boss kill' })).toBeInTheDocument()
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

  it('should list the post on the news page when it is published after the news page was already loaded', async () => {
    signInAs(officer)
    const published = [
      { ...draftPost, id: 'post-old', title: 'Older post', publishedAt: '2026-01-01T00:00:00Z' },
    ]
    server.use(
      http.get('/api/news', () =>
        HttpResponse.json(newsPage(published as unknown as NewsPost[])),
      ),
      http.post('/api/news/:id/publish', () => {
        const nowPublished = { ...draftPost, publishedAt: '2026-09-01T00:00:00Z' }
        published.unshift(nowPublished)
        return HttpResponse.json(nowPublished)
      }),
    )
    const user = userEvent.setup()
    renderWithProviders(
      <>
        <Link to="/news">Go to news</Link>
        <Link to="/officer/news/draft-1">Go to editor</Link>
        <Routes>
          <Route path="/officer/news/:id" element={<OfficerNewsEditor />} />
          <Route path="/news" element={<News />} />
          <Route path="/news/:id" element={<p>Post page</p>} />
        </Routes>
      </>,
      '/news',
    )
    await screen.findByText('Older post')
    await user.click(screen.getByRole('link', { name: 'Go to editor' }))
    await user.click(await screen.findByRole('button', { name: 'Publish' }))
    await user.click(screen.getByRole('button', { name: 'Confirm publish' }))
    await screen.findByText('Post page')

    await user.click(screen.getByRole('link', { name: 'Go to news' }))

    expect(await screen.findByText('Patch 11.0 notes')).toBeInTheDocument()
  })

  it('should move the post from drafts to published on the officer list when it is published', async () => {
    signInAs(officer)
    const published = [
      { ...draftPost, id: 'post-old', title: 'Older post', publishedAt: '2026-01-01T00:00:00Z' },
    ]
    let drafts = [draftPost]
    server.use(
      http.get('/api/news/drafts', () => HttpResponse.json(drafts)),
      http.get('/api/news', () => HttpResponse.json(newsPage(published as unknown as NewsPost[]))),
      http.post('/api/news/:id/publish', () => {
        const nowPublished = { ...draftPost, publishedAt: '2026-09-01T00:00:00Z' }
        published.unshift(nowPublished)
        drafts = []
        return HttpResponse.json(nowPublished)
      }),
    )
    const user = userEvent.setup()
    renderWithProviders(
      <>
        <Link to="/officer/news">Officer list</Link>
        <Routes>
          <Route path="/officer/news" element={<OfficerNews />} />
          <Route path="/officer/news/:id" element={<OfficerNewsEditor />} />
          <Route path="/news/:id" element={<p>Post page</p>} />
        </Routes>
      </>,
      '/officer/news',
    )
    await user.click(await screen.findByRole('link', { name: 'Patch 11.0 notes' }))
    await user.click(await screen.findByRole('button', { name: 'Publish' }))
    await user.click(screen.getByRole('button', { name: 'Confirm publish' }))
    await screen.findByText('Post page')

    await user.click(screen.getByRole('link', { name: 'Officer list' }))

    expect(await screen.findByText('No drafts.')).toBeInTheDocument()
    expect(screen.getAllByRole('link', { name: 'Patch 11.0 notes' })).toHaveLength(1)
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
