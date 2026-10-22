import { screen, within } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { renderWithProviders } from '../../testUtils'
import { About } from './About'

describe('About', () => {
  it('should show the guild pitch when the page renders', () => {
    renderWithProviders(<About />)

    expect(screen.getByRole('heading', { level: 1, name: 'About Fuzion' })).toBeInTheDocument()
    expect(screen.getByText('Serious about clearing. Not about ourselves.')).toBeInTheDocument()
    expect(
      screen.getByText('A PvE guild that takes the raid seriously and everything else lightly.'),
    ).toBeInTheDocument()
  })

  it('should describe the hero image for screen readers when the page renders', () => {
    renderWithProviders(<About />)

    expect(
      screen.getByRole('img', {
        name: 'A group of adventurers silhouetted in a snowy canyon, facing a glowing amber gateway in an ancient fortress.',
      }),
    ).toBeInTheDocument()
  })

  it.each(["Who's behind Fuzion", 'Culture', 'Questions', 'Who we are looking for'])(
    'should show the %s section heading when the page renders',
    (heading) => {
      renderWithProviders(<About />)

      expect(screen.getByRole('heading', { level: 2, name: heading })).toBeInTheDocument()
    },
  )

  it('should show the core team, partner raid and undecided schedule tiles when the page renders', () => {
    renderWithProviders(<About />)

    const stats = within(screen.getByRole('list', { name: 'Guild at a glance' }))
    expect(stats.getAllByRole('listitem').map((item) => item.textContent)).toEqual([
      '20person core team',
      '40with partner guilds',
      'TBAraid nights, set with the roster',
    ])
  })

  it('should name the wrestling show when the page renders', () => {
    renderWithProviders(<About />)

    const founder = within(screen.getByRole('region', { name: "Who's behind Fuzion" }))
    expect(founder.getByText(/hosts the CCW Pro Wrestling Supershow on Twitch/)).toBeInTheDocument()
  })

  it('should say Fuzion has raided retail since 2024 when the page renders', () => {
    renderWithProviders(<About />)

    const founder = within(screen.getByRole('region', { name: "Who's behind Fuzion" }))
    expect(
      founder.getByText(/Fuzion has raided retail together since 2024, and Forever is the next chapter/),
    ).toBeInTheDocument()
  })

  it('should link to the streams page from the founder section when the page renders', () => {
    renderWithProviders(<About />)

    const founder = within(screen.getByRole('region', { name: "Who's behind Fuzion" }))
    expect(founder.getByRole('link', { name: 'streams page' })).toHaveAttribute('href', '/streams')
  })

  it('should answer the server question with the PvE server when the page renders', () => {
    renderWithProviders(<About />)

    const faq = within(screen.getByRole('region', { name: 'Questions' }))
    expect(faq.getByRole('heading', { level: 3, name: 'Which server will you be on?' })).toBeInTheDocument()
    expect(faq.getByText(/^The PvE server\. There is only one PvE megaserver/)).toBeInTheDocument()
  })

  it('should say the realm name is not known yet when the page renders', () => {
    renderWithProviders(<About />)

    const faq = within(screen.getByRole('region', { name: 'Questions' }))
    expect(faq.getByText(/we will update this once we know whether it gets its own realm name/)).toBeInTheDocument()
  })

  it('should point to the calendar for raid nights when the page renders', () => {
    renderWithProviders(<About />)

    const faq = within(screen.getByRole('region', { name: 'Questions' }))
    expect(faq.getByRole('heading', { level: 3, name: 'When do you raid?' })).toBeInTheDocument()
    expect(faq.getByRole('link', { name: 'calendar' })).toHaveAttribute('href', '/calendar')
  })

  it('should explain how to join when the page renders', () => {
    renderWithProviders(<About />)

    const faq = within(screen.getByRole('region', { name: 'Questions' }))
    expect(faq.getByRole('heading', { level: 3, name: 'How do I join?' })).toBeInTheDocument()
    expect(faq.getByText('Send in an application and an officer will get back to you.')).toBeInTheDocument()
  })

  it('should link to the founder site in a new tab when the page renders', () => {
    renderWithProviders(<About />)

    const link = screen.getByRole('link', { name: 'Centrifuze' })
    expect(link).toHaveAttribute('href', 'https://fuziongaming.gg/')
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveAttribute('rel', 'noopener noreferrer')
  })

  it('should point every apply link at the applications page when the page renders', () => {
    renderWithProviders(<About />)

    const applyLinks = screen.getAllByRole('link', { name: 'Apply to Fuzion' })
    expect(applyLinks.map((link) => link.getAttribute('href'))).toEqual(['/applications', '/applications'])
  })
})
