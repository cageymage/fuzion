import { describe, expect, it } from 'vitest'
import { formatCountdown, formatRelativeDate, formatViewerCount } from './format'

describe('formatCountdown', () => {
  it('should show hours and minutes when the raid is less than a day away', () => {
    expect(formatCountdown(2 * 3_600_000 + 14 * 60_000)).toBe('2h 14m')
  })

  it('should show days and hours when the raid is more than a day away', () => {
    expect(formatCountdown(3 * 86_400_000 + 4 * 3_600_000 + 30 * 60_000)).toBe('3d 4h')
  })

  it('should show minutes only when the raid is less than an hour away', () => {
    expect(formatCountdown(12 * 60_000 + 45_000)).toBe('12m')
  })

  it('should show a starting-now message when the start time has passed', () => {
    expect(formatCountdown(-5_000)).toBe('Starting now')
  })
})

describe('formatViewerCount', () => {
  it('should show the exact count when there are fewer than a thousand viewers', () => {
    expect(formatViewerCount(842)).toBe('842')
  })

  it('should show one decimal place when the count is between one and ten thousand', () => {
    expect(formatViewerCount(1_240)).toBe('1.2K')
  })

  it('should round to whole thousands when the count is ten thousand or more', () => {
    expect(formatViewerCount(24_600)).toBe('25K')
  })
})

describe('formatRelativeDate', () => {
  const now = new Date('2026-09-17T12:00:00Z').getTime()

  it('should show a day count when the post is several days old', () => {
    expect(formatRelativeDate('2026-09-15T12:00:00Z', now)).toBe('2 days ago')
  })

  it('should show yesterday when the post is one day old', () => {
    expect(formatRelativeDate('2026-09-16T12:00:00Z', now)).toBe('yesterday')
  })

  it('should show today when the post is less than a day old', () => {
    expect(formatRelativeDate('2026-09-17T09:00:00Z', now)).toBe('today')
  })
})
