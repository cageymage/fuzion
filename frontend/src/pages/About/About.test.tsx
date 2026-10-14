import { screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { renderWithProviders } from '../../testUtils'
import { About } from './About'

describe('About', () => {
  it('should show the guild pitch when the page renders', () => {
    renderWithProviders(<About />)

    expect(screen.getByRole('heading', { level: 1, name: 'About Fuzion' })).toBeInTheDocument()
    expect(
      screen.getByText(
        'Fuzion is a PvE guild built around raiding, dungeons and a community that sticks together between tiers.',
      ),
    ).toBeInTheDocument()
  })

  it.each(['Raid schedule', 'Progression', 'Culture', 'Who we are looking for'])(
    'should show the %s section heading when the page renders',
    (heading) => {
      renderWithProviders(<About />)

      expect(screen.getByRole('heading', { level: 2, name: heading })).toBeInTheDocument()
    },
  )

  it('should point every apply link at the applications page when the page renders', () => {
    renderWithProviders(<About />)

    const applyLinks = screen.getAllByRole('link', { name: 'Apply to Fuzion' })
    expect(applyLinks.map((link) => link.getAttribute('href'))).toEqual(['/applications', '/applications'])
  })

  it('should link to the calendar and raid progress pages when the page renders', () => {
    renderWithProviders(<About />)

    expect(screen.getByRole('link', { name: 'calendar' })).toHaveAttribute('href', '/calendar')
    expect(screen.getByRole('link', { name: 'raid progress' })).toHaveAttribute(
      'href',
      '/raid-progress',
    )
  })
})
