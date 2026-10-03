export type Role = 'tank' | 'healer' | 'dps'

export interface CreateCharacterRequest {
  name: string
  secondaryName: string
  realm?: string
  class: string
  spec: string
  role: Role
  spec2?: string
  role2?: Role
  isMain: boolean
  raidTeam?: string
}

// An empty string for spec2 and role2 clears the second spec; omitting them leaves it unchanged.
export type UpdateCharacterRequest = Partial<Omit<CreateCharacterRequest, 'role2'>> & {
  role2?: Role | ''
}

export interface Character {
  id: string
  name: string
  secondaryName: string
  realm: string
  class: string
  spec: string | null
  role: Role
  spec2: string | null
  role2: Role | null
  isMain: boolean
  raidTeam: string | null
  createdAt: string
}
