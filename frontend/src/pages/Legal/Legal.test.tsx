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

  it('should name forever.fuziongaming.gg as the site the privacy policy covers when the page renders', () => {
    renderWithProviders(<Legal />)

    expect(screen.getByRole('region', { name: 'Privacy policy' })).toHaveTextContent(
      'This policy covers the Fuzion guild site, forever.fuziongaming.gg.',
    )
  })

  it('should say other fuziongaming.gg sites have their own notices when the page renders', () => {
    renderWithProviders(<Legal />)

    expect(screen.getByRole('region', { name: 'Privacy policy' })).toHaveTextContent(
      'fuziongaming.gg and other fuziongaming.gg sites are run by the same people and have their own notices.',
    )
  })

  it('should say a Discord login on one fuziongaming.gg site does not log you in on another when the page renders', () => {
    renderWithProviders(<Legal />)

    expect(screen.getByRole('region', { name: 'Privacy policy' })).toHaveTextContent(
      'Logging in with Discord on one of them does not log you in on another.',
    )
  })

  it('should name the third parties that handle data when the page renders', () => {
    renderWithProviders(<Legal />)

    const privacy = screen.getByRole('region', { name: 'Privacy policy' })
    for (const party of ['Cloudflare Turnstile', 'Twitch', 'YouTube', 'Render']) {
      expect(privacy).toHaveTextContent(party)
    }
  })

  it('should list the expected playstyle among the application fields when the page renders', () => {
    renderWithProviders(<Legal />)

    expect(screen.getByRole('region', { name: 'Privacy policy' })).toHaveTextContent(
      'role, expected playstyle, availability',
    )
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

  it('should credit Midjourney for the about page artwork when the page renders', () => {
    renderWithProviders(<Legal />)

    expect(screen.getByRole('region', { name: 'Attributions' })).toHaveTextContent(
      'About page artwork: generated with Midjourney.',
    )
  })

  it('should credit YouTube for live status and channel avatars when the page renders', () => {
    renderWithProviders(<Legal />)

    expect(screen.getByRole('region', { name: 'Attributions' })).toHaveTextContent(
      'Embedded videos, live status and channel avatars: YouTube, a Google LLC service.',
    )
  })

  it('should credit the GitHub logo to GitHub when the page renders', () => {
    renderWithProviders(<Legal />)

    expect(screen.getByRole('region', { name: 'Attributions' })).toHaveTextContent(
      'GitHub and the GitHub logo: trademarks of GitHub, Inc.',
    )
  })

  it('should state the Blizzard disclaimer when the page renders', () => {
    renderWithProviders(<Legal />)

    expect(
      screen.getByRole('region', { name: 'Blizzard and World of Warcraft' }),
    ).toHaveTextContent('not affiliated with or endorsed by Blizzard Entertainment')
  })
})
