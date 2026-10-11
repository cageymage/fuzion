import { describe, expect, it } from 'vitest'
import { classColor } from './classColor'

function relativeLuminance(hex: string): number {
  const [red, green, blue] = [1, 3, 5].map((start) => {
    const channel = parseInt(hex.slice(start, start + 2), 16) / 255
    return channel <= 0.03928 ? channel / 12.92 : ((channel + 0.055) / 1.055) ** 2.4
  })
  return 0.2126 * red + 0.7152 * green + 0.0722 * blue
}

function contrastRatio(foreground: string, background: string): number {
  const [lighter, darker] = [relativeLuminance(foreground), relativeLuminance(background)].sort(
    (a, b) => b - a,
  )
  return (lighter + 0.05) / (darker + 0.05)
}

describe('classColor', () => {
  it('should return the WoW class colour for Warrior', () => {
    expect(classColor('Warrior')).toBe('#C79C6E')
  })

  it('should return the WoW class colour for Mage', () => {
    expect(classColor('Mage')).toBe('#69CCF0')
  })

  it('should keep every class colour at 4.5:1 contrast or better on the card surface', () => {
    const surface = '#201811'
    const classes = ['Warrior', 'Paladin', 'Hunter', 'Rogue', 'Priest', 'Shaman', 'Mage', 'Warlock', 'Druid']

    const contrasts = Object.fromEntries(
      classes.map((className) => [className, contrastRatio(classColor(className), surface)]),
    )

    for (const className of classes) {
      expect(contrasts[className], className).toBeGreaterThanOrEqual(4.5)
    }
  })

  it('should fall back to the theme text colour when the class is unknown', () => {
    expect(classColor('Unknown')).toBe('var(--text)')
  })
})
