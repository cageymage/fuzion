import { describe, expect, it } from 'vitest'
import { craftingProfessions, gatheringProfessions, primaryProfessions } from './professions'

describe('primaryProfessions', () => {
  it('should contain exactly the crafting and gathering professions', () => {
    expect([...primaryProfessions].sort()).toEqual([...craftingProfessions, ...gatheringProfessions].sort())
  })
})
