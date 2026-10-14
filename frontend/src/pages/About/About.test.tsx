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
    expect(applyLinks).toHaveLength(2)
    for (const link of applyLinks) {
      expect(link).toHaveAttribute('href', '/applications')
    }
  })

  it('should show the credits section after the main content when the page renders', () => {
    renderWithProviders(<About />)

    const credits = screen.getByRole('region', { name: 'Credits' })
    expect(credits).toHaveTextContent('Hosting: Render.')
    expect(screen.getByRole('article').lastElementChild).toBe(credits)
  })
})
