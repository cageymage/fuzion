export interface ProfessionCharacter {
  id: string
  name: string
  secondaryName: string
  class: string
}

export type SpecializationSource = 'blizzard' | 'manual'

export interface ProfessionEntry {
  id: string
  profession: string
  skillLevel: number
  specialization: string | null
  // null without a specialization; a 'blizzard' one cannot be changed by hand.
  specializationSource: SpecializationSource | null
  character: ProfessionCharacter
}

export interface Recipe {
  id: number
  name: string
  profession: string
  characters: ProfessionCharacter[]
}
