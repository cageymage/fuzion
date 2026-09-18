import { act, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { NextRaid } from '../../types/raids'
import { NextRaidCard } from './NextRaidCard'

const raid: NextRaid = {
  id: 'raid-1',
  difficulty: 'Mythic',
  instanceName: "Nerub'ar Palace",
  startsAt: '2026-09-17T22:14:00Z',
  progressSummary: '8/8 Heroic cleared',
}

describe('NextRaidCard', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-09-17T20:00:00Z'))
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('should show the hours and minutes until the raid starts', () => {
    render(<NextRaidCard raid={raid} />)

    expect(screen.getByText('2h 14m')).toBeInTheDocument()
  })

  it('should show the difficulty, instance and current progress', () => {
    render(<NextRaidCard raid={raid} />)

    expect(screen.getByText("Mythic · Nerub'ar Palace")).toBeInTheDocument()
    expect(screen.getByText('8/8 Heroic cleared')).toBeInTheDocument()
  })

  it('should count the remaining time down as the clock advances', () => {
    render(<NextRaidCard raid={raid} />)

    act(() => vi.advanceTimersByTime(60_000))

    expect(screen.getByText('2h 13m')).toBeInTheDocument()
  })

  it('should show a starting-now message when the start time has passed', () => {
    vi.setSystemTime(new Date('2026-09-17T22:20:00Z'))

    render(<NextRaidCard raid={raid} />)

    expect(screen.getByText('Starting now')).toBeInTheDocument()
  })
})
