import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { SuggestedVideoCard } from './SuggestedVideoCard'

describe('SuggestedVideoCard', () => {
  it('should embed the video from YouTube', () => {
    render(<SuggestedVideoCard video={{ id: 'vid-1', title: 'Queen Ansurek kill' }} />)

    const player = screen.getAllByTitle('Queen Ansurek kill').find((element) => element.tagName === 'IFRAME')
    expect(player).toHaveAttribute('src', 'https://www.youtube.com/embed/vid-1')
  })

  it('should lazy-load the embedded player so it does not load until scrolled near', () => {
    render(<SuggestedVideoCard video={{ id: 'vid-1', title: 'Queen Ansurek kill' }} />)

    const player = screen.getAllByTitle('Queen Ansurek kill').find((element) => element.tagName === 'IFRAME')
    expect(player).toHaveAttribute('loading', 'lazy')
  })

  it('should link to the video on YouTube', () => {
    render(<SuggestedVideoCard video={{ id: 'vid-1', title: 'Queen Ansurek kill' }} />)

    expect(screen.getByRole('link', { name: 'Watch on YouTube →' })).toHaveAttribute(
      'href',
      'https://www.youtube.com/watch?v=vid-1',
    )
  })

  it('should encode the video id in the watch link when the id has URL characters', () => {
    render(<SuggestedVideoCard video={{ id: 'a&b=c', title: 'Queen Ansurek kill' }} />)

    expect(screen.getByRole('link', { name: 'Watch on YouTube →' })).toHaveAttribute(
      'href',
      'https://www.youtube.com/watch?v=a%26b%3Dc',
    )
  })
})
