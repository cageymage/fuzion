export interface ProfessionCharacter {
  id: string
  name: string
  secondaryName: string
  class: string
}

export interface ProfessionEntry {
  id: string
  profession: string
  skillLevel: number
  character: ProfessionCharacter
}
