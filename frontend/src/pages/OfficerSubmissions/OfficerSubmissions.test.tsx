import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { Route, Routes } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { server } from '../../mocks/server'
import { renderWithProviders } from '../../testUtils'
import type { AuthUser } from '../../types/auth'
import type { Submission } from '../../types/submissions'
import { OfficerSubmissions } from './OfficerSubmissions'

const officer: AuthUser = {
  id: 'officer-1',
  username: 'officerone',
  avatarUrl: null,
  isOfficer: true,
  isGuildMember: true,
  battlenetLinked: false,
  battletag: null,
}

function submission(id: string, username: string, description: string, filename: string): Submission {
  return {
    id,
    submitter: { id: `user-${id}`, username },
    description,
    originalFilename: filename,
    contentType: 'image/png',
    width: 1920,
    height: 1080,
    byteSize: 1000,
    createdAt: '2026-10-26T12:00:00Z',
    thumbUrl: `/api/submissions/${id}/thumb`,
    fullUrl: `/api/submissions/${id}/full`,
    downloadUrl: `/api/submissions/${id}/download`,
  }
}

// A list endpoint backed by an array the delete handlers shrink, so refetches show what is left.
function serveSubmissions(initial: Submission[]) {
  let items = [...initial]
  const deleted: string[][] = []
  server.use(
    http.get('/api/auth/me', () => HttpResponse.json(officer)),
    http.get('/api/submissions', () => HttpResponse.json({ total: items.length, limit: 48, offset: 0, items })),
    http.delete('/api/submissions/:id', ({ params }) => {
      deleted.push([params.id as string])
      items = items.filter((item) => item.id !== params.id)
      return new HttpResponse(null, { status: 204 })
    }),
    http.post('/api/submissions/bulk-delete', async ({ request }) => {
      const { ids } = (await request.json()) as { ids: string[] }
      deleted.push(ids)
      items = items.filter((item) => !ids.includes(item.id))
      return HttpResponse.json({ deleted: ids })
    }),
  )
  return deleted
}

const onyxia = submission('sub-1', 'thundermane', 'Onyxia down!', 'onyxia.png')
const fanart = submission('sub-2', 'mendwell', '', 'fanart.jpg')

async function gridItems() {
  const grid = await screen.findByRole('list', { name: 'Submissions' })
  return within(grid)
    .getAllByRole('listitem')
    .map((item) => item.textContent)
}

describe('OfficerSubmissions', () => {
  it('should send a member who is not an officer to the home page', async () => {
    server.use(http.get('/api/auth/me', () => HttpResponse.json({ ...officer, isOfficer: false })))

    renderWithProviders(
      <Routes>
        <Route path="/officer/submissions" element={<OfficerSubmissions />} />
        <Route path="/" element={<p>Home page</p>} />
      </Routes>,
      '/officer/submissions',
    )

    expect(await screen.findByText('Home page')).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Review submissions' })).not.toBeInTheDocument()
  })

  it('should show a thumbnail grid with each submitter and description', async () => {
    serveSubmissions([onyxia, fanart])

    renderWithProviders(<OfficerSubmissions />, '/officer/submissions')

    expect(await gridItems()).toEqual(['thundermaneOnyxia down!DownloadDelete', 'mendwellDownloadDelete'])
    expect(screen.getByText('Showing 1–2 of 2')).toBeInTheDocument()
  })

  it('should show the full image in a lightbox when its thumbnail is clicked', async () => {
    serveSubmissions([onyxia])
    renderWithProviders(<OfficerSubmissions />, '/officer/submissions')

    await userEvent.click(await screen.findByRole('button', { name: 'Open image: Onyxia down!' }))

    const lightbox = screen.getByRole('dialog', { name: 'Image preview' })
    expect(within(lightbox).getByRole('img', { name: 'Onyxia down!' })).toHaveAttribute('src', '/api/submissions/sub-1/full')
  })

  it('should offer the original file as a download on the card', async () => {
    serveSubmissions([onyxia])
    renderWithProviders(<OfficerSubmissions />, '/officer/submissions')

    const download = await screen.findByRole('link', { name: 'Download original of Onyxia down!' })

    expect(download).toHaveAttribute('href', '/api/submissions/sub-1/download')
    expect(download).toHaveAttribute('download', 'onyxia.png')
  })

  it('should tick the submission instead of opening the lightbox when its thumbnail is clicked in select mode', async () => {
    serveSubmissions([onyxia])
    renderWithProviders(<OfficerSubmissions />, '/officer/submissions')
    await userEvent.click(await screen.findByRole('button', { name: 'Select' }))

    await userEvent.click(screen.getByRole('button', { name: 'Toggle Onyxia down!' }))

    expect(screen.getByRole('checkbox', { name: 'Select Onyxia down!' })).toBeChecked()
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('should delete a submission from the grid once the deletion is confirmed', async () => {
    const deleted = serveSubmissions([onyxia, fanart])
    renderWithProviders(<OfficerSubmissions />, '/officer/submissions')

    await userEvent.click(await screen.findByRole('button', { name: 'Delete Onyxia down!' }))
    await userEvent.click(within(screen.getByRole('dialog', { name: 'Delete submission?' })).getByRole('button', { name: 'Delete' }))

    expect(await gridItems()).toEqual(['mendwellDownloadDelete'])
    expect(deleted).toEqual([['sub-1']])
  })

  it('should keep a submission when its deletion is cancelled', async () => {
    const deleted = serveSubmissions([onyxia])
    renderWithProviders(<OfficerSubmissions />, '/officer/submissions')

    await userEvent.click(await screen.findByRole('button', { name: 'Delete Onyxia down!' }))
    await userEvent.click(within(screen.getByRole('dialog', { name: 'Delete submission?' })).getByRole('button', { name: 'Cancel' }))

    expect(screen.queryByRole('dialog', { name: 'Delete submission?' })).not.toBeInTheDocument()
    expect(await gridItems()).toEqual(['thundermaneOnyxia down!DownloadDelete'])
    expect(deleted).toEqual([])
  })

  it('should delete every selected submission in one request when Delete selected is confirmed', async () => {
    const deleted = serveSubmissions([onyxia, fanart, submission('sub-3', 'keeper', 'Keep me', 'keep.png')])
    renderWithProviders(<OfficerSubmissions />, '/officer/submissions')

    await userEvent.click(await screen.findByRole('button', { name: 'Select' }))
    await userEvent.click(screen.getByRole('checkbox', { name: 'Select Onyxia down!' }))
    await userEvent.click(screen.getByRole('checkbox', { name: 'Select fanart.jpg' }))
    await userEvent.click(screen.getByRole('button', { name: 'Delete selected (2)' }))
    await userEvent.click(within(screen.getByRole('dialog', { name: 'Delete 2 submissions?' })).getByRole('button', { name: 'Delete' }))

    expect(await gridItems()).toEqual(['keeperKeep me'])
    expect(deleted).toEqual([['sub-1', 'sub-2']])
  })

  it('should show the error when a deletion fails', async () => {
    serveSubmissions([onyxia])
    server.use(http.delete('/api/submissions/:id', () => HttpResponse.json({ error: 'delete submission failed' }, { status: 500 })))
    renderWithProviders(<OfficerSubmissions />, '/officer/submissions')

    await userEvent.click(await screen.findByRole('button', { name: 'Delete Onyxia down!' }))
    await userEvent.click(within(screen.getByRole('dialog', { name: 'Delete submission?' })).getByRole('button', { name: 'Delete' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('delete submission failed')
  })

  it('should go back to the previous page when the last submission on the last page is deleted', async () => {
    let lastPageDeleted = false
    server.use(
      http.get('/api/auth/me', () => HttpResponse.json(officer)),
      http.get('/api/submissions', ({ request }) => {
        const offset = Number(new URL(request.url).searchParams.get('offset') ?? '0')
        if (lastPageDeleted) {
          return HttpResponse.json({ total: 48, limit: 48, offset, items: offset === 0 ? [onyxia] : [] })
        }
        return HttpResponse.json({ total: 49, limit: 48, offset, items: [offset === 0 ? onyxia : fanart] })
      }),
      http.delete('/api/submissions/:id', () => {
        lastPageDeleted = true
        return new HttpResponse(null, { status: 204 })
      }),
    )
    renderWithProviders(<OfficerSubmissions />, '/officer/submissions')
    await userEvent.click(await screen.findByRole('button', { name: 'Next' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Delete fanart.jpg' }))

    await userEvent.click(within(screen.getByRole('dialog', { name: 'Delete submission?' })).getByRole('button', { name: 'Delete' }))

    expect(await screen.findByText('Showing 1–1 of 48')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Open image: Onyxia down!' })).toBeInTheDocument()
  })

  it('should load the next page when Next is clicked', async () => {
    const offsets: string[] = []
    server.use(
      http.get('/api/auth/me', () => HttpResponse.json(officer)),
      http.get('/api/submissions', ({ request }) => {
        const offset = new URL(request.url).searchParams.get('offset') ?? '0'
        offsets.push(offset)
        const item = offset === '0' ? onyxia : fanart
        return HttpResponse.json({ total: 60, limit: 48, offset: Number(offset), items: [item] })
      }),
    )
    renderWithProviders(<OfficerSubmissions />, '/officer/submissions')
    await screen.findByRole('button', { name: 'Open image: Onyxia down!' })

    await userEvent.click(screen.getByRole('button', { name: 'Next' }))

    expect(await screen.findByRole('button', { name: 'Open image: fanart.jpg' })).toBeInTheDocument()
    expect(offsets).toEqual(['0', '48'])
  })
})
