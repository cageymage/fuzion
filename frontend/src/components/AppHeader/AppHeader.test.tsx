import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { AppHeader } from './AppHeader'

describe('AppHeader', () => {
  it('should render a link for every navigation item', () => {
    render(
      <MemoryRouter initialEntries={['/']}>
        <AppHeader />
      </MemoryRouter>,
    )

    expect(screen.getByRole('link', { name: 'Home' })).toHaveAttribute('href', '/')
    expect(screen.getByRole('link', { name: 'Roster' })).toHaveAttribute('href', '/roster')
    expect(screen.getByRole('link', { name: 'Calendar & Events' })).toHaveAttribute(
      'href',
      '/calendar',
    )
  })

  it('should mark the roster link as current when the roster route is active', () => {
    render(
      <MemoryRouter initialEntries={['/roster']}>
        <AppHeader />
      </MemoryRouter>,
    )

    expect(screen.getByRole('link', { name: 'Roster' })).toHaveAttribute('aria-current', 'page')
    expect(screen.getByRole('link', { name: 'Home' })).not.toHaveAttribute('aria-current')
  })
})
