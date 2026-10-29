import { act, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { LaunchCountdownCard } from './LaunchCountdownCard'

const launchAt = '2026-11-26T17:00:00Z'

describe('LaunchCountdownCard', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('should show the days, hours, minutes and seconds until launch when launch is in the future', () => {
    vi.setSystemTime(new Date('2026-10-20T12:47:51Z'))

    render(<LaunchCountdownCard launchAt={launchAt} />)

    expect(screen.getByText('WoW Forever launches in')).toBeInTheDocument()
    expect(screen.getByText('days').parentElement).toHaveTextContent('37days')
    expect(screen.getByText('hours').parentElement).toHaveTextContent('04hours')
    expect(screen.getByText('minutes').parentElement).toHaveTextContent('12minutes')
    expect(screen.getByText('seconds').parentElement).toHaveTextContent('09seconds')
  })

  it('should count the remaining seconds down when the clock advances', () => {
    vi.setSystemTime(new Date('2026-10-20T12:47:51Z'))
    render(<LaunchCountdownCard launchAt={launchAt} />)

    act(() => vi.advanceTimersByTime(2_000))

    expect(screen.getByText('seconds').parentElement).toHaveTextContent('07seconds')
  })

  it('should show the live message when launch has passed', () => {
    vi.setSystemTime(new Date('2026-11-26T17:00:01Z'))

    render(<LaunchCountdownCard launchAt={launchAt} />)

    expect(screen.getByText('WoW Forever is live')).toBeInTheDocument()
    expect(screen.queryByText('WoW Forever launches in')).not.toBeInTheDocument()
  })

  it('should still show the live message when 47 hours have passed since launch', () => {
    vi.setSystemTime(new Date('2026-11-28T16:00:00Z'))

    render(<LaunchCountdownCard launchAt={launchAt} />)

    expect(screen.getByText('WoW Forever is live')).toBeInTheDocument()
  })

  it('should render nothing when 48 hours have passed since launch', () => {
    vi.setSystemTime(new Date('2026-11-28T17:00:00Z'))

    const { container } = render(<LaunchCountdownCard launchAt={launchAt} />)

    expect(container).toBeEmptyDOMElement()
  })

  it('should switch from the countdown to the live message when the clock crosses launch', () => {
    vi.setSystemTime(new Date('2026-11-26T16:59:58Z'))
    render(<LaunchCountdownCard launchAt={launchAt} />)

    act(() => vi.advanceTimersByTime(2_000))

    expect(screen.getByText('WoW Forever is live')).toBeInTheDocument()
  })

  it('should disappear when the clock crosses 48 hours after launch', () => {
    vi.setSystemTime(new Date('2026-11-28T16:59:59Z'))
    const { container } = render(<LaunchCountdownCard launchAt={launchAt} />)

    act(() => vi.advanceTimersByTime(1_000))

    expect(container).toBeEmptyDOMElement()
  })
})
