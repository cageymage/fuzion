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

// Forever's Warcraft Logs site has no realms: "normal" sits where a realm slug would, and
// the two names are joined with a space, e.g. .../character/us/normal/terrorskull%20gloomstep.
export function warcraftLogsUrl(name: string, secondaryName: string): string {
  const characterName = [name, secondaryName]
    .map((part) => part.trim().toLowerCase())
    .filter(Boolean)
    .join(' ')
  return `https://forever.warcraftlogs.com/character/${WOW_REGION}/normal/${encodeURIComponent(characterName)}`
}
