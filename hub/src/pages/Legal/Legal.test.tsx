import { screen, within } from '@testing-library/react'
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

  it('should state that the hub collects no personal data when the page renders', () => {
    renderWithProviders(<Legal />)

    expect(screen.getByRole('region', { name: 'Privacy policy' })).toHaveTextContent(
      'it collects no personal data itself and sets no cookies of its own',
    )
  })

  it('should name the third parties involved in running the hub when the page renders', () => {
    renderWithProviders(<Legal />)

    const privacy = screen.getByRole('region', { name: 'Privacy policy' })
    for (const party of ['Render', 'Twitch', 'YouTube', 'Google Fonts']) {
      expect(privacy).toHaveTextContent(party)
    }
  })

  it('should say embedded YouTube videos may set their own cookies when the page renders', () => {
    renderWithProviders(<Legal />)

    expect(screen.getByRole('region', { name: 'Privacy policy' })).toHaveTextContent(
      'Videos embedded from YouTube load from YouTube, which may set its own cookies.',
    )
  })

  it('should link to the guild site legal page for what the guild site collects when the page renders', () => {
    renderWithProviders(<Legal />)

    const privacy = screen.getByRole('region', { name: 'Privacy policy' })
    expect(
      within(privacy).getByRole('link', { name: 'forever.fuziongaming.gg legal page' }),
    ).toHaveAttribute('href', 'https://forever.fuziongaming.gg/legal')
  })

  it('should say removal is requested through an officer on Discord when the page renders', () => {
    renderWithProviders(<Legal />)

    expect(screen.getByRole('region', { name: 'Copyright and content' })).toHaveTextContent(
      'ask an officer on Discord',
    )
  })

  it('should state the Blizzard disclaimer when the page renders', () => {
    renderWithProviders(<Legal />)

    expect(
      screen.getByRole('region', { name: 'Blizzard and World of Warcraft' }),
    ).toHaveTextContent('not affiliated with or endorsed by Blizzard Entertainment')
  })

  it('should credit the Twitch and YouTube logos to their trademark owners when the page renders', () => {
    renderWithProviders(<Legal />)

    expect(screen.getByRole('region', { name: 'Attributions' })).toHaveTextContent(
      'Twitch and YouTube logos: trademarks of Twitch Interactive, Inc. and Google LLC',
    )
  })

  it('should credit Simple Icons for the platform icon shapes when the page renders', () => {
    renderWithProviders(<Legal />)

    expect(screen.getByRole('region', { name: 'Attributions' })).toHaveTextContent(
      'Icon shapes from Simple Icons (CC0).',
    )
  })

  it('should credit Render for hosting when the page renders', () => {
    renderWithProviders(<Legal />)

    expect(screen.getByRole('region', { name: 'Attributions' })).toHaveTextContent(
      'Hosting: Render Services, Inc.',
    )
  })
})
