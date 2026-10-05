import type { ProfessionEntry } from '../../types/professions'

export interface ProfessionGroup {
  profession: string
  entries: ProfessionEntry[]
}

function matches(entry: ProfessionEntry, query: string): boolean {
  return [entry.profession, entry.character.name, entry.character.secondaryName].some((text) =>
    text.toLowerCase().includes(query),
  )
}

export function groupProfessions(rows: ProfessionEntry[], query: string): ProfessionGroup[] {
  const normalizedQuery = query.trim().toLowerCase()
  const groups = new Map<string, ProfessionEntry[]>()

  for (const entry of rows) {
    if (!matches(entry, normalizedQuery)) continue
    const entries = groups.get(entry.profession)
    if (entries) {
      entries.push(entry)
    } else {
      groups.set(entry.profession, [entry])
    }
  }

  return Array.from(groups, ([profession, entries]) => ({ profession, entries }))
}
