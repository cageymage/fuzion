// Region isn't stored per character; the guild plays on one US realm.
export const WOW_REGION = 'us'

// Blizzard drops apostrophes from realm slugs, e.g. Mal'Ganis becomes malganis.
export function realmSlug(realm: string): string {
  return realm.trim().toLowerCase().replace(/'/g, '').replace(/\s+/g, '-')
}

// Forever's Armory URL is unconfirmed until launch. Both parts are guesses: the
// /forever/ path follows retail's /worldsoul/ and Classic's /classic/, and the two
// names are joined with a hyphen.
export function armoryUrl(realm: string, name: string, secondaryName: string): string {
  const characterSlug = [name, secondaryName]
    .map((part) => part.trim().toLowerCase())
    .filter(Boolean)
    .join('-')
  return `https://worldofwarcraft.blizzard.com/en-us/forever/${WOW_REGION}/armory/character/${realmSlug(realm)}/${encodeURIComponent(characterSlug)}`
}
