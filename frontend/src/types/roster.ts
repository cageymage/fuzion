export type Role = 'tank' | 'healer' | 'dps'

export interface Character {
  id: string
  name: string
  secondaryName: string
  realm: string
  class: string
  spec: string | null
  role: Role
  isMain: boolean
  raidTeam: string | null
  createdAt: string
}
