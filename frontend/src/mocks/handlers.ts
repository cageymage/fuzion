import { http, HttpResponse } from 'msw'
import type { NewsPage, NewsPost } from '../types/news'
import type { RaidProgress, RaidTier } from '../types/raidProgress'
import type { Raid } from '../types/raids'
import type { Character } from '../types/roster'
import type { Stream } from '../types/streams'

const newsPosts: NewsPost[] = [
  {
    id: 'post-1',
    title: 'Fuzion defeated Queen Ansurek on Mythic',
    excerpt:
      "Mythic Nerub'ar Palace, 8/8 down — a huge night for the team after weeks on the enrage.",
    category: 'raid-progress',
    imageUrl: null,
    authorName: 'Officer',
    publishedAt: new Date(Date.now() - 2 * 86_400_000).toISOString(),
  },
  {
    id: 'post-2',
    title: 'Now recruiting: Restoration Druid & Fire Mage',
    excerpt: 'Two raid spots open for the Mythic roster.',
    category: 'recruitment',
    imageUrl: null,
    authorName: 'Officer',
    publishedAt: new Date(Date.now() - 4 * 86_400_000).toISOString(),
  },
  {
    id: 'post-3',
    title: 'Welcome our newest officers',
    excerpt: 'Two promotions from the raid team.',
    category: 'guild-news',
    imageUrl: null,
    authorName: 'Officer',
    publishedAt: new Date(Date.now() - 6 * 86_400_000).toISOString(),
  },
]

export function newsPage(posts: NewsPost[], total = posts.length): NewsPage {
  const onlyPage = { href: '/api/news?limit=10&offset=0' }
  return {
    total,
    _links: { self: onlyPage, first: onlyPage, last: onlyPage },
    _embedded: { news: posts },
  }
}

export function newsHandler(posts: NewsPost[]) {
  return http.get('/api/news', ({ request }) => {
    const params = new URL(request.url).searchParams
    const category = params.get('category')
    const limit = Number(params.get('limit') ?? 10)
    const offset = Number(params.get('offset') ?? 0)
    const matching = posts.filter((post) => !category || post.category === category)
    return HttpResponse.json(newsPage(matching.slice(offset, offset + limit), matching.length))
  })
}

const nextRaid: Raid = {
  id: 'raid-1',
  difficulty: 'Mythic',
  instanceName: "Nerub'ar Palace",
  startsAt: new Date(Date.now() + 2 * 3_600_000 + 14 * 60_000 + 30_000).toISOString(),
  progressSummary: '8/8 Heroic cleared',
}

const atLocalTime = (daysAhead: number, hour: number): string => {
  const date = new Date()
  date.setDate(date.getDate() + daysAhead)
  date.setHours(hour, 0, 0, 0)
  return date.toISOString()
}

const upcomingRaids: Raid[] = [
  {
    id: 'raid-1',
    difficulty: 'Mythic',
    instanceName: "Nerub'ar Palace",
    startsAt: atLocalTime(2, 19),
    progressSummary: '8/8 Heroic cleared',
  },
  {
    id: 'raid-2',
    difficulty: 'Heroic',
    instanceName: "Nerub'ar Palace",
    startsAt: atLocalTime(2, 21),
    progressSummary: '8/8 Heroic cleared',
  },
  {
    id: 'raid-3',
    difficulty: 'Normal',
    instanceName: 'Liberation of Undermine',
    startsAt: atLocalTime(4, 20),
    progressSummary: 'Not started',
  },
]

const liveStreams: Stream[] = [
  {
    id: 'stream-1',
    streamerName: 'Thundermane',
    gameName: 'World of Warcraft: Forever',
    title: 'Mythic Queen Ansurek progress',
    viewerCount: 1_240,
    thumbnailUrl: null,
    avatarUrl: 'https://cdn.example/thundermane.png',
    channelUrl: 'https://twitch.tv/thundermane',
    isLive: true,
  },
]

const allStreams: Stream[] = [
  ...liveStreams,
  {
    id: 'stream-2',
    streamerName: 'Moonveil',
    gameName: '',
    title: '',
    viewerCount: 0,
    thumbnailUrl: null,
    avatarUrl: null,
    channelUrl: 'https://twitch.tv/moonveil',
    isLive: false,
  },
  {
    id: 'stream-3',
    streamerName: 'Aelith',
    gameName: '',
    title: '',
    viewerCount: 0,
    thumbnailUrl: null,
    avatarUrl: 'https://cdn.example/aelith.png',
    channelUrl: 'https://twitch.tv/aelith',
    isLive: false,
  },
]

const rosterCharacters: Character[] = [
  {
    id: 'char-1',
    name: 'Ragnok',
    secondaryName: '',
    realm: 'Emberreach',
    class: 'Warrior',
    spec: 'Protection',
    role: 'tank',
    isMain: true,
    raidTeam: 'Team Alpha',
    createdAt: new Date().toISOString(),
  },
  {
    id: 'char-2',
    name: 'Korrath',
    secondaryName: '',
    realm: 'Emberreach',
    class: 'Paladin',
    spec: 'Protection',
    role: 'tank',
    isMain: true,
    raidTeam: 'Team Beta',
    createdAt: new Date().toISOString(),
  },
  {
    id: 'char-3',
    name: 'Selene',
    secondaryName: '',
    realm: 'Emberreach',
    class: 'Priest',
    spec: 'Holy',
    role: 'healer',
    isMain: true,
    raidTeam: 'Team Alpha',
    createdAt: new Date().toISOString(),
  },
  {
    id: 'char-4',
    name: 'Mirelle',
    secondaryName: '',
    realm: 'Emberreach',
    class: 'Druid',
    spec: 'Restoration',
    role: 'healer',
    isMain: true,
    raidTeam: 'Team Beta',
    createdAt: new Date().toISOString(),
  },
  {
    id: 'char-5',
    name: 'Zaldrin',
    secondaryName: '',
    realm: 'Emberreach',
    class: 'Mage',
    spec: 'Fire',
    role: 'dps',
    isMain: true,
    raidTeam: 'Team Alpha',
    createdAt: new Date().toISOString(),
  },
  {
    id: 'char-6',
    name: 'Zaldrix',
    secondaryName: '',
    realm: 'Emberreach',
    class: 'Warlock',
    spec: 'Affliction',
    role: 'dps',
    isMain: false,
    raidTeam: null,
    createdAt: new Date().toISOString(),
  },
]

const raidProgress: RaidProgress = {
  tier: { name: 'Molten Depths' },
  bosses: [
    { id: 'boss-1', name: 'Grimjaw', killedAt: new Date(Date.now() - 3 * 86_400_000).toISOString() },
    { id: 'boss-2', name: 'Ashveil', killedAt: null },
    { id: 'boss-3', name: 'Pyrelord', killedAt: null },
  ],
  killed: 1,
  total: 3,
}

const raidTiers: RaidTier[] = [
  {
    id: 'tier-1',
    name: raidProgress.tier.name,
    isCurrent: true,
    sortOrder: 2,
    bosses: raidProgress.bosses,
  },
  {
    id: 'tier-2',
    name: 'Shattered Spire',
    isCurrent: false,
    sortOrder: 1,
    bosses: [
      { id: 'boss-4', name: 'Voidshard Sentinel', killedAt: new Date(Date.now() - 30 * 86_400_000).toISOString() },
      { id: 'boss-5', name: 'Thornqueen Ilyra', killedAt: new Date(Date.now() - 28 * 86_400_000).toISOString() },
    ],
  },
]

export const handlers = [
  http.get('/api/auth/me', () => HttpResponse.json({ error: 'login required' }, { status: 401 })),
  http.post('/api/auth/logout', () => new HttpResponse(null, { status: 204 })),
  newsHandler(newsPosts),
  http.get('/api/raid-progress', () => HttpResponse.json([raidProgress])),
  http.get('/api/raid-tiers', () => HttpResponse.json(raidTiers)),
  http.post('/api/raid-tiers', () =>
    HttpResponse.json(
      { id: 'tier-3', name: 'New Tier', isCurrent: false, sortOrder: 3, bosses: [] },
      { status: 201 },
    ),
  ),
  http.put('/api/raid-tiers/order', () => new HttpResponse(null, { status: 204 })),
  http.patch('/api/raid-tiers/:id', () => HttpResponse.json(raidTiers[0])),
  http.delete('/api/raid-tiers/:id', () => new HttpResponse(null, { status: 204 })),
  http.post('/api/raid-tiers/:id/bosses', () =>
    HttpResponse.json({ id: 'boss-6', name: 'New Boss', killedAt: null }, { status: 201 }),
  ),
  http.put('/api/raid-tiers/:id/bosses/order', () => new HttpResponse(null, { status: 204 })),
  http.patch('/api/raid-bosses/:id', () => HttpResponse.json(raidProgress.bosses[0])),
  http.delete('/api/raid-bosses/:id', () => new HttpResponse(null, { status: 204 })),
  http.get('/api/raids', () => HttpResponse.json(upcomingRaids)),
  http.get('/api/raids/next', () => HttpResponse.json(nextRaid)),
  http.get('/api/roster', () => HttpResponse.json(rosterCharacters)),
  http.get('/api/streams', () => HttpResponse.json(allStreams)),
  http.get('/api/streams/live', () => HttpResponse.json(liveStreams)),
]
