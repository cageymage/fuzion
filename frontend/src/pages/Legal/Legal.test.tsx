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

  it('should describe the short-lived cookie that remembers your page during login when the page renders', () => {
    renderWithProviders(<Legal />)

    expect(screen.getByRole('region', { name: 'Privacy policy' })).toHaveTextContent(
      'a short-lived cookie that remembers which page you were on so we can send you back to it after you log in, deleted when login finishes',
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

  it('should say linking Battle.net records which roster characters are yours when the page renders', () => {
    renderWithProviders(<Legal />)

    const privacy = screen.getByRole('region', { name: 'Privacy policy' })
    expect(privacy).toHaveTextContent('to record which guild roster characters are yours')
    expect(privacy).toHaveTextContent("not the list or Battle.net's access token")
    expect(privacy).toHaveTextContent('removes you as the owner of those characters')
  })

  it('should say submitted originals are kept with their file data and are visible only to officers when the page renders', () => {
    renderWithProviders(<Legal />)

    const privacy = screen.getByRole('region', { name: 'Privacy policy' })
    expect(privacy).toHaveTextContent('we keep each exact original file')
    expect(privacy).toHaveTextContent('the GPS location where it was taken')
    expect(privacy).toHaveTextContent('Only officers can see submissions; they are never public.')
    expect(privacy).toHaveTextContent('which re-encodes it and drops that data')
  })

  it('should state the license submitters grant and what they attest when the page renders', () => {
    renderWithProviders(<Legal />)

    const copyright = screen.getByRole('region', { name: 'Copyright and content' })
    expect(copyright).toHaveTextContent(
      'you confirm that you created each image or bought the rights to it (game screenshots are fine)',
    )
    expect(copyright).toHaveTextContent(
      'grant the guild an unrestricted license to use it on this site and in related social media posts',
    )
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

  it('should describe the survey data stored and who can see it when the page renders', () => {
    renderWithProviders(<Legal />)

    expect(screen.getByRole('region', { name: 'Privacy policy' })).toHaveTextContent(
      'Survey responses: your answers, the guild name you optionally enter, and your Discord username at the time you respond. Officers can see these per person, and the public results show totals only.',
    )
  })

  it('should explain the Discord server check at login and what is kept when the page renders', () => {
    renderWithProviders(<Legal />)

    expect(screen.getByRole('region', { name: 'Privacy policy' })).toHaveTextContent(
      "Discord server check: when you log in, we also ask Discord which servers you are in, only to record whether you are in the guild's Discord server. We store that yes or no, not the list, and use it to decide who can answer guild surveys.",
    )
  })

  it('should state the Blizzard disclaimer when the page renders', () => {
    renderWithProviders(<Legal />)

    expect(
      screen.getByRole('region', { name: 'Blizzard and World of Warcraft' }),
    ).toHaveTextContent('not affiliated with or endorsed by Blizzard Entertainment')
  })
})
