const classColors: Record<string, string> = {
  Warrior: '#C79C6E',
  Paladin: '#F58CBA',
  Hunter: '#ABD473',
  Rogue: '#FFF569',
  Priest: '#FFFFFF',
  // Blizzard's #0070DE is only 3.6:1 on the dark surface, so this is lightened to pass AA.
  Shaman: '#2E8CF0',
  Mage: '#69CCF0',
  Warlock: '#9482C9',
  Druid: '#FF7D0A',
}

export function classColor(className: string): string {
  return classColors[className] ?? 'var(--text)'
}
