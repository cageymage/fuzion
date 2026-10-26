// Region isn't stored per character; the guild plays on one US realm.
export const WOW_REGION = 'us'

// Blizzard drops apostrophes from realm slugs, e.g. Mal'Ganis becomes malganis.
export function realmSlug(realm: string): string {
  return realm.trim().toLowerCase().replace(/'/g, '').replace(/\s+/g, '-')
}

// Both names joined with a hyphen, lowercased and percent-encoded. A guess for Forever's
// two-part names until a real Armory or Warcraft Logs profile URL can be checked.
function characterSlug(name: string, secondaryName: string): string {
  const slug = [name, secondaryName]
    .map((part) => part.trim().toLowerCase())
    .filter(Boolean)
    .join('-')
  return encodeURIComponent(slug)
}

// Forever's Armory URL is unconfirmed until launch: the /forever/ path follows retail's
// /worldsoul/ and Classic's /classic/.
export function armoryUrl(realm: string, name: string, secondaryName: string): string {
  return `https://worldofwarcraft.blizzard.com/en-us/forever/${WOW_REGION}/armory/character/${realmSlug(realm)}/${characterSlug(name, secondaryName)}`
}

// Whether Warcraft Logs supports Forever, and on which host, is unconfirmed; this is
// its retail character URL.
export function warcraftLogsUrl(realm: string, name: string, secondaryName: string): string {
  return `https://www.warcraftlogs.com/character/${WOW_REGION}/${realmSlug(realm)}/${characterSlug(name, secondaryName)}`
}
