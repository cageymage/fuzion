export interface NavItem {
  label: string
  path: string
}

export const navItems: NavItem[] = [
  { label: 'Home', path: '/' },
  { label: 'News', path: '/news' },
  { label: 'Roster', path: '/roster' },
  { label: 'Raid Progress', path: '/raid-progress' },
  { label: 'Calendar & Events', path: '/calendar' },
  { label: 'Applications', path: '/applications' },
  { label: 'Professions', path: '/professions' },
  { label: 'PvP', path: '/pvp' },
  { label: 'Streams', path: '/streams' },
]
