import { http, HttpResponse } from 'msw'
import type { NewsPost } from '../types/news'
import type { NextRaid } from '../types/raids'
import type { LiveStream } from '../types/streams'

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

const nextRaid: NextRaid = {
  id: 'raid-1',
  difficulty: 'Mythic',
  instanceName: "Nerub'ar Palace",
  startsAt: new Date(Date.now() + 2 * 3_600_000 + 14 * 60_000 + 30_000).toISOString(),
  progressSummary: '8/8 Heroic cleared',
}

const liveStreams: LiveStream[] = [
  {
    id: 'stream-1',
    streamerName: 'Thundermane',
    gameName: 'World of Warcraft: Forever',
    viewerCount: 1_240,
    thumbnailUrl: null,
    channelUrl: 'https://twitch.tv/thundermane',
  },
]

export const handlers = [
  http.get('/api/news', () => HttpResponse.json(newsPosts)),
  http.get('/api/raids/next', () => HttpResponse.json(nextRaid)),
  http.get('/api/streams/live', () => HttpResponse.json(liveStreams)),
]
