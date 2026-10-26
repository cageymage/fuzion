import { describe, expect, it } from 'vitest'
import { formatViewerCount } from './format'

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
