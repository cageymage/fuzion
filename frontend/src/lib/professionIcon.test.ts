import { describe, expect, it } from 'vitest'
import { anyProfessionIcon, primaryProfessionIcons, professionIcon } from './professionIcon'

describe('anyProfessionIcon', () => {
  it('should return the Blizzard icon url when the profession is a secondary profession', () => {
    expect(anyProfessionIcon('First Aid')).toBe(
      'https://render.worldofwarcraft.com/us/icons/36/spell_holy_sealofsacrifice.jpg',
    )
  })

  it('should return the Blizzard icon url when the profession is a primary profession', () => {
    expect(anyProfessionIcon('Mining')).toBe('https://render.worldofwarcraft.com/us/icons/36/trade_mining.jpg')
  })

  it('should return null when the profession is not known', () => {
    expect(anyProfessionIcon('Basket Weaving')).toBeNull()
  })
})

describe('professionIcon', () => {
  it('should return the Blizzard icon url when the profession is a primary trade profession', () => {
    expect(professionIcon('Alchemy')).toBe(
      'https://render.worldofwarcraft.com/us/icons/36/trade_alchemy.jpg',
    )
  })

  it('should return null when the profession is a secondary profession', () => {
    expect(professionIcon('Cooking')).toBeNull()
  })

  it.each(['Jewelcrafting', 'Inscription'])(
    'should return null when the profession is %s, which is not in Forever yet',
    (profession) => {
      expect(professionIcon(profession)).toBeNull()
    },
  )

  it('should return null when the profession is not known', () => {
    expect(professionIcon('Basket Weaving')).toBeNull()
  })
})

describe('primaryProfessionIcons', () => {
  it('should return an icon for each primary profession in the given order', () => {
    const icons = primaryProfessionIcons([
      { id: 'herbalism-row', profession: 'Herbalism', skillLevel: 300, specialization: null, specializationSource: null },
      { id: 'alchemy-row', profession: 'Alchemy', skillLevel: 225, specialization: null, specializationSource: null },
    ])

    expect(icons.map((icon) => icon.profession)).toEqual(['Herbalism', 'Alchemy'])
    expect(icons[0].iconUrl).toBe('https://render.worldofwarcraft.com/us/icons/36/trade_herbalism.jpg')
  })

  it('should skip secondary professions when the character has both kinds', () => {
    const icons = primaryProfessionIcons([
      { id: 'cooking-row', profession: 'Cooking', skillLevel: 300, specialization: null, specializationSource: null },
      { id: 'mining-row', profession: 'Mining', skillLevel: 150, specialization: null, specializationSource: null },
    ])

    expect(icons.map((icon) => icon.profession)).toEqual(['Mining'])
  })

  it('should return at most two icons when the character has more than two primary professions', () => {
    const icons = primaryProfessionIcons([
      { id: 'mining-row', profession: 'Mining', skillLevel: 300, specialization: null, specializationSource: null },
      { id: 'skinning-row', profession: 'Skinning', skillLevel: 225, specialization: null, specializationSource: null },
      { id: 'tailoring-row', profession: 'Tailoring', skillLevel: 100, specialization: null, specializationSource: null },
    ])

    expect(icons.map((icon) => icon.profession)).toEqual(['Mining', 'Skinning'])
  })

  it('should return no icons when the character has only secondary professions', () => {
    expect(primaryProfessionIcons([{ id: 'fishing-row', profession: 'Fishing', skillLevel: 75, specialization: null, specializationSource: null }])).toEqual([])
  })

  it('should return no icons when the character has no professions', () => {
    expect(primaryProfessionIcons([])).toEqual([])
  })
})
