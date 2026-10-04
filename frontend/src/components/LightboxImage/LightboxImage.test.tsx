import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { LightboxImage } from './LightboxImage'

describe('LightboxImage', () => {
  it('should show the image as a thumbnail when it has not been clicked', () => {
    render(<LightboxImage src="https://cdn.example/boss.png" alt="Boss kill" />)

    expect(screen.getByRole('img', { name: 'Boss kill' })).toHaveAttribute(
      'src',
      'https://cdn.example/boss.png',
    )
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('should open the lightbox when the thumbnail is clicked', async () => {
    const user = userEvent.setup()
    render(<LightboxImage src="https://cdn.example/boss.png" alt="Boss kill" />)

    await user.click(screen.getByRole('button', { name: 'Open image: Boss kill' }))

    const lightbox = screen.getByRole('dialog', { name: 'Image preview' })
    expect(lightbox).toBeInTheDocument()
    expect(screen.getAllByRole('img', { name: 'Boss kill' })).toHaveLength(2)
  })

  it('should close the lightbox when Escape is pressed', async () => {
    const user = userEvent.setup()
    render(<LightboxImage src="https://cdn.example/boss.png" alt="Boss kill" />)
    await user.click(screen.getByRole('button', { name: 'Open image: Boss kill' }))

    await user.keyboard('{Escape}')

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('should close the lightbox when the backdrop is clicked', async () => {
    const user = userEvent.setup()
    render(<LightboxImage src="https://cdn.example/boss.png" alt="Boss kill" />)
    await user.click(screen.getByRole('button', { name: 'Open image: Boss kill' }))

    await user.click(screen.getByRole('dialog', { name: 'Image preview' }))

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('should label the thumbnail button generically when the image has no alt text', () => {
    render(<LightboxImage src="https://cdn.example/boss.png" />)

    expect(screen.getByRole('button', { name: 'Open image' })).toBeInTheDocument()
  })

  it('should show the thumbnail variant and open the full-size image when the image is an uploaded one', async () => {
    const user = userEvent.setup()
    render(<LightboxImage src="/api/images/2f1c0b9e-1111-4222-8333-444455556666" alt="Boss kill" />)

    expect(screen.getByRole('img', { name: 'Boss kill' })).toHaveAttribute(
      'src',
      '/api/images/2f1c0b9e-1111-4222-8333-444455556666/thumb',
    )
    await user.click(screen.getByRole('button', { name: 'Open image: Boss kill' }))

    const lightbox = screen.getByRole('dialog', { name: 'Image preview' })
    expect(within(lightbox).getByRole('img', { name: 'Boss kill' })).toHaveAttribute(
      'src',
      '/api/images/2f1c0b9e-1111-4222-8333-444455556666',
    )
  })
})
