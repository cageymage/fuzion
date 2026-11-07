import { describe, expect, it } from 'vitest'
import { wowheadRecipeSearchUrl } from './wowhead'

describe('wowheadRecipeSearchUrl', () => {
  it('should build a Forever search url when given a recipe name', () => {
    expect(wowheadRecipeSearchUrl('Light Armor Kit')).toBe(
      'https://www.wowhead.com/forever/search?q=Light%20Armor%20Kit',
    )
  })

  it('should escape reserved characters when the recipe name contains them', () => {
    expect(wowheadRecipeSearchUrl('Transmute: Iron & Gold')).toBe(
      'https://www.wowhead.com/forever/search?q=Transmute%3A%20Iron%20%26%20Gold',
    )
  })
})
