import { describe, expect, it } from 'vitest'
import { classColor } from './classColor'

describe('classColor', () => {
  it('should return the WoW class colour for Warrior', () => {
    expect(classColor('Warrior')).toBe('#C79C6E')
  })

  it('should return the WoW class colour for Mage', () => {
    expect(classColor('Mage')).toBe('#69CCF0')
  })

  it('should fall back to the theme text colour when the class is unknown', () => {
    expect(classColor('Unknown')).toBe('var(--text)')
  })
})
