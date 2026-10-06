import { describe, expect, it } from 'vitest'
import type { Character } from '../../types/roster'
import {
  buildCreateRequest,
  buildUpdateRequest,
  emptyCharacterForm,
  formValuesFor,
  type CharacterFormValues,
} from './characterForm'

const aeliana: Character = {
  id: 'char-1',
  name: 'Aeliana',
  secondaryName: 'Dawnsong',
  realm: 'Emberreach',
  class: 'Priest',
  spec: 'Holy',
  role: 'healer',
  spec2: null,
  role2: null,
  isMain: true,
  raidTeam: 'Team 1',
  createdAt: '2026-01-01T00:00:00.000Z',
  professions: [],
}

describe('buildCreateRequest', () => {
  it('should omit the realm, second spec and raid team when they are left blank', () => {
    const values: CharacterFormValues = {
      ...emptyCharacterForm,
      name: 'Aeliana',
      secondaryName: 'Dawnsong',
      class: 'Priest',
      spec: 'Holy',
      role: 'healer',
    }

    expect(buildCreateRequest(values)).toEqual({
      name: 'Aeliana',
      secondaryName: 'Dawnsong',
      class: 'Priest',
      spec: 'Holy',
      role: 'healer',
      isMain: true,
    })
  })

  it('should include the second spec and role when a second spec is chosen', () => {
    const values: CharacterFormValues = {
      ...emptyCharacterForm,
      name: 'Aeliana',
      secondaryName: 'Dawnsong',
      class: 'Priest',
      spec: 'Holy',
      role: 'healer',
      spec2: 'Shadow',
      role2: 'dps',
      realm: 'Stormrage',
      raidTeam: 'Team 2',
      isMain: false,
    }

    expect(buildCreateRequest(values)).toEqual({
      name: 'Aeliana',
      secondaryName: 'Dawnsong',
      realm: 'Stormrage',
      class: 'Priest',
      spec: 'Holy',
      role: 'healer',
      spec2: 'Shadow',
      role2: 'dps',
      isMain: false,
      raidTeam: 'Team 2',
    })
  })
})

describe('buildUpdateRequest', () => {
  it('should send nothing when no field changed', () => {
    const initial = formValuesFor(aeliana)

    expect(buildUpdateRequest(initial, initial)).toEqual({})
  })

  it('should send only the fields that changed', () => {
    const initial = formValuesFor(aeliana)

    const request = buildUpdateRequest(initial, { ...initial, raidTeam: 'Team 2', isMain: false })

    expect(request).toEqual({ raidTeam: 'Team 2', isMain: false })
  })

  it('should send both the second spec and role when a second spec is added', () => {
    const initial = formValuesFor(aeliana)

    const request = buildUpdateRequest(initial, { ...initial, spec2: 'Shadow', role2: 'dps' })

    expect(request).toEqual({ spec2: 'Shadow', role2: 'dps' })
  })

  it('should send both the second spec and role when only the second role changed', () => {
    const initial = formValuesFor({ ...aeliana, spec2: 'Shadow', role2: 'dps' })

    const request = buildUpdateRequest(initial, { ...initial, role2: 'healer' })

    expect(request).toEqual({ spec2: 'Shadow', role2: 'healer' })
  })

  it('should send empty strings for both the second spec and role when the second spec is removed', () => {
    const initial = formValuesFor({ ...aeliana, spec2: 'Shadow', role2: 'dps' })

    const request = buildUpdateRequest(initial, { ...initial, spec2: '', role2: '' })

    expect(request).toEqual({ spec2: '', role2: '' })
  })
})

describe('formValuesFor professions', () => {
  it('should split the professions into two primary choices and the secondary ones when the character has both kinds', () => {
    const values = formValuesFor({
      ...aeliana,
      professions: [
        { profession: 'Mining', skillLevel: 300 },
        { profession: 'Cooking', skillLevel: 150 },
        { profession: 'Skinning', skillLevel: 100 },
      ],
    })

    expect(values.primaryProfession1).toBe('Mining')
    expect(values.primaryProfession2).toBe('Skinning')
    expect(values.secondaryProfessions).toEqual(['Cooking'])
  })

  it('should ignore a profession that is no longer offered when the character still has it stored', () => {
    const values = formValuesFor({ ...aeliana, professions: [{ profession: 'Jewelcrafting', skillLevel: 300 }] })

    expect(values.primaryProfession1).toBe('')
    expect(values.primaryProfession2).toBe('')
    expect(values.secondaryProfessions).toEqual([])
  })
})

describe('profession requests', () => {
  const named: CharacterFormValues = {
    ...emptyCharacterForm,
    name: 'Aeliana',
    secondaryName: 'Dawnsong',
    class: 'Priest',
    spec: 'Holy',
    role: 'healer',
  }

  it('should send the primary professions before the secondary ones when a character is created', () => {
    const request = buildCreateRequest({
      ...named,
      primaryProfession1: 'Mining',
      primaryProfession2: 'Herbalism',
      secondaryProfessions: ['Cooking'],
    })

    expect(request.professions).toEqual(['Mining', 'Herbalism', 'Cooking'])
  })

  it('should send the full professions list when a profession is added to a character', () => {
    const initial = formValuesFor({ ...aeliana, professions: [{ profession: 'Mining', skillLevel: 300 }] })

    const request = buildUpdateRequest(initial, { ...initial, primaryProfession2: 'Herbalism' })

    expect(request).toEqual({ professions: ['Mining', 'Herbalism'] })
  })

  it('should send an empty list when every profession is removed from a character', () => {
    const initial = formValuesFor({ ...aeliana, professions: [{ profession: 'Mining', skillLevel: 300 }] })

    const request = buildUpdateRequest(initial, { ...initial, primaryProfession1: '' })

    expect(request).toEqual({ professions: [] })
  })

  it('should send nothing for professions when only their order changed', () => {
    const initial = formValuesFor({
      ...aeliana,
      professions: [
        { profession: 'Mining', skillLevel: 300 },
        { profession: 'Herbalism', skillLevel: 200 },
      ],
    })

    const request = buildUpdateRequest(initial, {
      ...initial,
      primaryProfession1: 'Herbalism',
      primaryProfession2: 'Mining',
    })

    expect(request).toEqual({})
  })
})
