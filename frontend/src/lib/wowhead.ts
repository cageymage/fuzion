// Blizzard's recipe ids are not Wowhead spell ids, so a recipe links to a search for its name.
export function wowheadRecipeSearchUrl(recipeName: string): string {
  return `https://www.wowhead.com/forever/search?q=${encodeURIComponent(recipeName)}`
}
