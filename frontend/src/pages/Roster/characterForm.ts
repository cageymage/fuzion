import { isPrimaryProfession, isSecondaryProfession } from '../../lib/professions'
import type {
  Character,
  CreateCharacterRequest,
  Role,
  UpdateCharacterRequest,
} from '../../types/roster'

export interface CharacterFormValues {
  name: string
  secondaryName: string
  realm: string
  class: string
  spec: string
  role: Role
  spec2: string
  role2: Role | ''
  isMain: boolean
  raidTeam: string
  race: string
  level: string
  primaryProfession1: string
  primaryProfession2: string
  secondaryProfessions: string[]
}

export const emptyCharacterForm: CharacterFormValues = {
  name: '',
  secondaryName: '',
  realm: '',
  class: '',
  spec: '',
  role: 'dps',
  spec2: '',
  role2: '',
  isMain: true,
  raidTeam: '',
  race: '',
  level: '',
  primaryProfession1: '',
  primaryProfession2: '',
  secondaryProfessions: [],
}

// Names a character has that the form no longer offers (or a third primary) are not shown.
function professionChoices(character: Character) {
  const names = character.professions.map(({ profession }) => profession)
  const primaries = names.filter(isPrimaryProfession)
  return {
    primaryProfession1: primaries[0] ?? '',
    primaryProfession2: primaries[1] ?? '',
    secondaryProfessions: names.filter(isSecondaryProfession),
  }
}

function professionList(values: CharacterFormValues): string[] {
  return [values.primaryProfession1, values.primaryProfession2, ...values.secondaryProfessions].filter(
    (name) => name !== '',
  )
}

function sameProfessions(a: string[], b: string[]): boolean {
  return a.length === b.length && [...a].sort().join('\n') === [...b].sort().join('\n')
}

export function formValuesFor(character: Character): CharacterFormValues {
  return {
    ...professionChoices(character),
    name: character.name,
    secondaryName: character.secondaryName,
    realm: character.realm,
    class: character.class,
    spec: character.spec ?? '',
    role: character.role,
    spec2: character.spec2 ?? '',
    role2: character.role2 ?? '',
    isMain: character.isMain,
    raidTeam: character.raidTeam ?? '',
    race: character.race ?? '',
    level: character.level === null ? '' : String(character.level),
  }
}

export function buildCreateRequest(values: CharacterFormValues): CreateCharacterRequest {
  const request: CreateCharacterRequest = {
    name: values.name,
    secondaryName: values.secondaryName,
    class: values.class,
    spec: values.spec,
    role: values.role,
    isMain: values.isMain,
  }
  if (values.realm !== '') request.realm = values.realm
  if (values.spec2 !== '' && values.role2 !== '') {
    request.spec2 = values.spec2
    request.role2 = values.role2
  }
  if (values.raidTeam !== '') request.raidTeam = values.raidTeam
  if (values.race !== '') request.race = values.race
  if (values.level !== '') request.level = Number(values.level)
  const professions = professionList(values)
  if (professions.length > 0) request.professions = professions
  return request
}

export function buildUpdateRequest(
  initial: CharacterFormValues,
  current: CharacterFormValues,
): UpdateCharacterRequest {
  const request: UpdateCharacterRequest = {}
  if (current.name !== initial.name) request.name = current.name
  if (current.secondaryName !== initial.secondaryName) request.secondaryName = current.secondaryName
  if (current.realm !== initial.realm) request.realm = current.realm
  if (current.class !== initial.class) request.class = current.class
  if (current.spec !== initial.spec) request.spec = current.spec
  if (current.role !== initial.role) request.role = current.role
  if (current.spec2 !== initial.spec2 || current.role2 !== initial.role2) {
    request.spec2 = current.spec2
    request.role2 = current.role2
  }
  if (current.isMain !== initial.isMain) request.isMain = current.isMain
  if (current.raidTeam !== initial.raidTeam) request.raidTeam = current.raidTeam
  // The backend can set race and level but not clear them, so an emptied field is left out.
  if (current.race !== initial.race && current.race !== '') request.race = current.race
  if (current.level !== initial.level && current.level !== '') request.level = Number(current.level)
  if (!sameProfessions(professionList(initial), professionList(current))) {
    request.professions = professionList(current)
  }
  return request
}
