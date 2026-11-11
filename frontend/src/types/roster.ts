export type Role = 'tank' | 'healer' | 'dps'

export interface CreateCharacterRequest {
  name: string
  secondaryName: string
  realm?: string
  class: string
  spec: string
  // The API answers 400 for an empty role; the type allows it so the form can be sent as is.
  role: Role | ''
  spec2?: string
  role2?: Role
  isMain: boolean
  raidTeam?: string
  race?: string
  level?: number
  professions?: string[]
}

// An empty string for spec2 and role2 clears the second spec; omitting them leaves it unchanged.
export type UpdateCharacterRequest = Partial<Omit<CreateCharacterRequest, 'role' | 'role2'>> & {
  role?: Role
  role2?: Role | ''
}

export interface CharacterProfession {
  profession: string
  skillLevel: number
}

export interface Character {
  id: string
  name: string
  secondaryName: string
  realm: string
  class: string
  spec: string | null
  // null until the owner picks one: the roster sync cannot know it.
  role: Role | null
  spec2: string | null
  role2: Role | null
  isMain: boolean
  // The in-game guild rank's name; null for characters the roster sync has never seen.
  rank: string | null
  // The in-game guild rank number (0 is the guild master); null when rank is null.
  rankOrder: number | null
  raidTeam: string | null
  race: string | null
  level: number | null
  faction: string | null
  gender: 'male' | 'female' | null
  createdAt: string
  professions: CharacterProfession[]
}

// What a member may set on their own character; an empty string for spec2 and role2
// clears the second spec.
export interface UpdateMyCharacterRequest {
  spec?: string
  role?: Role | ''
  spec2?: string
  role2?: Role | ''
}
