import { primaryProfessions, secondaryProfessions } from '../../lib/professions'

export const tabRows: readonly (readonly string[])[] = [primaryProfessions, secondaryProfessions]

export function resolveSelectedTab(selected: string | null, counts: Readonly<Record<string, number>>): string | null {
  if (selected !== null && (counts[selected] ?? 0) > 0) return selected
  return tabRows.flat().find((profession) => (counts[profession] ?? 0) > 0) ?? null
}
