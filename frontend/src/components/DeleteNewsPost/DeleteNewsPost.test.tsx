import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it, vi } from 'vitest'
import { server } from '../../mocks/server'
import { renderWithProviders } from '../../testUtils'
import { DeleteNewsPost } from './DeleteNewsPost'

describe('DeleteNewsPost', () => {
  it('should ask for confirmation without calling the API when Delete is clicked', async () => {
    const deletedIds: string[] = []
    server.use(
      http.delete('/api/news/:id', ({ params }) => {
        deletedIds.push(String(params.id))
        return new HttpResponse(null, { status: 204 })
      }),
    )
    const user = userEvent.setup()
    renderWithProviders(<DeleteNewsPost postId="post-1" onDeleted={() => {}} />)

    await user.click(screen.getByRole('button', { name: 'Delete' }))

    expect(screen.getByText('Delete this post? This cannot be undone.')).toBeInTheDocument()
    expect(deletedIds).toEqual([])
  })

  it('should delete the post and report it when deletion is confirmed', async () => {
    const deletedIds: string[] = []
    server.use(
      http.delete('/api/news/:id', ({ params }) => {
        deletedIds.push(String(params.id))
        return new HttpResponse(null, { status: 204 })
      }),
    )
    const onDeleted = vi.fn()
    const user = userEvent.setup()
    renderWithProviders(<DeleteNewsPost postId="post-1" onDeleted={onDeleted} />)

    await user.click(screen.getByRole('button', { name: 'Delete' }))
    await user.click(screen.getByRole('button', { name: 'Confirm delete' }))

    await vi.waitFor(() => expect(onDeleted).toHaveBeenCalledTimes(1))
    expect(deletedIds).toEqual(['post-1'])
  })

  it('should go back to the Delete button when the confirmation is cancelled', async () => {
    const onDeleted = vi.fn()
    const user = userEvent.setup()
    renderWithProviders(<DeleteNewsPost postId="post-1" onDeleted={onDeleted} />)

    await user.click(screen.getByRole('button', { name: 'Delete' }))
    await user.click(screen.getByRole('button', { name: 'Cancel delete' }))

    expect(screen.getByRole('button', { name: 'Delete' })).toBeInTheDocument()
    expect(onDeleted).not.toHaveBeenCalled()
  })

  it('should show the server error and not report deletion when the request fails', async () => {
    server.use(
      http.delete('/api/news/:id', () =>
        HttpResponse.json({ error: 'news post not found' }, { status: 404 }),
      ),
    )
    const onDeleted = vi.fn()
    const user = userEvent.setup()
    renderWithProviders(<DeleteNewsPost postId="post-1" onDeleted={onDeleted} />)

    await user.click(screen.getByRole('button', { name: 'Delete' }))
    await user.click(screen.getByRole('button', { name: 'Confirm delete' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('news post not found')
    expect(onDeleted).not.toHaveBeenCalled()
  })
})
