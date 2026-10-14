// Region isn't stored per character; the guild plays on one US realm.
export const WOW_REGION = 'us'

// Blizzard drops apostrophes from realm slugs, e.g. Mal'Ganis becomes malganis.
export function realmSlug(realm: string): string {
  return realm.trim().toLowerCase().replace(/'/g, '').replace(/\s+/g, '-')
}

// Forever's Armory path is predicted from retail's (/worldsoul/) and Classic's (/classic/)
// until Forever launches.
export function armoryUrl(realm: string, name: string): string {
  return `https://worldofwarcraft.blizzard.com/en-us/forever/${WOW_REGION}/armory/character/${realmSlug(realm)}/${encodeURIComponent(name.toLowerCase())}`
}
