// Region isn't stored per character; the guild plays on one US realm.
export const WOW_REGION = 'us'

// Forever's Armory has no realm in the path; the character is identified by both names
// joined with a hyphen, e.g. .../armory/character/psyence-spiritbound.
export function armoryUrl(name: string, secondaryName: string): string {
  const characterSlug = [name, secondaryName]
    .map((part) => part.trim().toLowerCase())
    .filter(Boolean)
    .join('-')
  return `https://worldofwarcraft.blizzard.com/en-us/forever/${WOW_REGION}/armory/character/${encodeURIComponent(characterSlug)}`
}
