import { describe, expect, it } from 'vitest'
import { resolveSelectedTab } from './tabSelection'

describe('resolveSelectedTab', () => {
  it('should keep the selected tab when it still has characters', () => {
    expect(resolveSelectedTab('Blacksmithing', { Alchemy: 2, Blacksmithing: 1 })).toBe('Blacksmithing')
  })

  it('should pick the first primary tab with characters when nothing is selected', () => {
    expect(resolveSelectedTab(null, { Blacksmithing: 1, Mining: 3, Cooking: 2 })).toBe('Blacksmithing')
  })

  it('should pick the first tab with characters when the selected tab has none', () => {
    expect(resolveSelectedTab('Alchemy', { Alchemy: 0, Tailoring: 1, Cooking: 2 })).toBe('Tailoring')
  })

  it('should fall through to the secondary row when no primary tab has characters', () => {
    expect(resolveSelectedTab(null, { Fishing: 1, 'First Aid': 4 })).toBe('Fishing')
  })

  it('should return null when no tab has characters', () => {
    expect(resolveSelectedTab('Alchemy', {})).toBeNull()
  })
})
