import { screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { renderWithProviders } from '../../testUtils'
import { Legal } from './Legal'

describe('Legal', () => {
  it.each([
    'Privacy policy',
    'Copyright and content',
    'Blizzard and World of Warcraft',
    'Attributions',
  ])('should show the %s section heading when the page renders', (heading) => {
    renderWithProviders(<Legal />)

    expect(screen.getByRole('heading', { level: 2, name: heading })).toBeInTheDocument()
  })

  it('should name the third parties that handle data when the page renders', () => {
    renderWithProviders(<Legal />)

    const privacy = screen.getByRole('region', { name: 'Privacy policy' })
    for (const party of ['Cloudflare Turnstile', 'Twitch', 'YouTube', 'Render']) {
      expect(privacy).toHaveTextContent(party)
    }
  })

  it('should describe the data stored when a member links Battle.net when the page renders', () => {
    renderWithProviders(<Legal />)

    const privacy = screen.getByRole('region', { name: 'Privacy policy' })
    expect(privacy).toHaveTextContent('Battle.net ID and BattleTag')
    expect(privacy).toHaveTextContent('Unlinking deletes both')
  })

  it('should say removal is requested through an officer on Discord when the page renders', () => {
    renderWithProviders(<Legal />)

    expect(screen.getByRole('region', { name: 'Privacy policy' })).toHaveTextContent(
      'ask an officer on Discord',
    )
  })

  it('should credit Midjourney for the home page artwork when the page renders', () => {
    renderWithProviders(<Legal />)

    expect(screen.getByRole('region', { name: 'Attributions' })).toHaveTextContent(
      'Home page artwork: generated with Midjourney.',
    )
  })

  it('should credit Midjourney for the news card background artwork when the page renders', () => {
    renderWithProviders(<Legal />)

    expect(screen.getByRole('region', { name: 'Attributions' })).toHaveTextContent(
      'News card background artwork: generated with Midjourney.',
    )
  })

  it('should state the Blizzard disclaimer when the page renders', () => {
    renderWithProviders(<Legal />)

    expect(
      screen.getByRole('region', { name: 'Blizzard and World of Warcraft' }),
    ).toHaveTextContent('not affiliated with or endorsed by Blizzard Entertainment')
  })
})
