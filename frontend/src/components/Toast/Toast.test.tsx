import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ToastProvider, useToast } from './Toast'

function ShowToastButton() {
  const { showToast } = useToast()
  return (
    <button type="button" onClick={() => showToast('Post published')}>
      Show toast
    </button>
  )
}

afterEach(() => {
  vi.useRealTimers()
})

describe('Toast', () => {
  it('should show the message when a toast is requested', async () => {
    const user = userEvent.setup()
    render(
      <ToastProvider>
        <ShowToastButton />
      </ToastProvider>,
    )

    await user.click(screen.getByRole('button', { name: 'Show toast' }))

    expect(screen.getByRole('status')).toHaveTextContent('Post published')
  })

  it('should hide the toast after four seconds', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })
    render(
      <ToastProvider>
        <ShowToastButton />
      </ToastProvider>,
    )
    await user.click(screen.getByRole('button', { name: 'Show toast' }))

    await act(async () => {
      vi.advanceTimersByTime(4000)
    })

    expect(screen.queryByRole('status')).not.toBeInTheDocument()
  })

  it('should hide the toast when it is dismissed', async () => {
    const user = userEvent.setup()
    render(
      <ToastProvider>
        <ShowToastButton />
      </ToastProvider>,
    )
    await user.click(screen.getByRole('button', { name: 'Show toast' }))

    await user.click(screen.getByRole('button', { name: 'Dismiss notification' }))

    expect(screen.queryByRole('status')).not.toBeInTheDocument()
  })

  it('should do nothing when there is no provider', async () => {
    const user = userEvent.setup()
    render(<ShowToastButton />)

    await user.click(screen.getByRole('button', { name: 'Show toast' }))

    expect(screen.queryByRole('status')).not.toBeInTheDocument()
  })
})
