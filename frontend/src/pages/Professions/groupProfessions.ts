import type { ProfessionEntry } from '../../types/professions'

export interface ProfessionGroup {
  profession: string
  entries: ProfessionEntry[]
}

// A recipe only keeps a character in the profession it belongs to, so the key pairs both.
export function crafterKey(profession: string, characterId: string): string {
  return `${profession}\u0000${characterId}`
}

function matches(entry: ProfessionEntry, query: string): boolean {
  return [entry.profession, entry.character.name, entry.character.secondaryName].some((text) =>
    text.toLowerCase().includes(query),
  )
}

export function groupProfessions(
  rows: ProfessionEntry[],
  query: string,
  crafters: ReadonlySet<string> = new Set(),
): ProfessionGroup[] {
  const normalizedQuery = query.trim().toLowerCase()
  const groups = new Map<string, ProfessionEntry[]>()

  for (const entry of rows) {
    const isCrafter = crafters.has(crafterKey(entry.profession, entry.character.id))
    if (!matches(entry, normalizedQuery) && !isCrafter) continue
    const entries = groups.get(entry.profession)
    if (entries) {
      entries.push(entry)
    } else {
      groups.set(entry.profession, [entry])
    }
  }

  return Array.from(groups, ([profession, entries]) => ({ profession, entries }))
}
