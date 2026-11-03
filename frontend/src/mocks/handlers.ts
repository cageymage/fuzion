import { http, HttpResponse } from 'msw'
import type { Application, ApplicationStatus } from '../types/applications'
import type { EditablePost, NewsPage, NewsPost } from '../types/news'
import type { RaidProgress, RaidTier } from '../types/raidProgress'
import type { Raid } from '../types/raids'
import type { ProfessionEntry } from '../types/professions'
import type { Character, MyCharacter } from '../types/roster'
import type { Submission } from '../types/submissions'
import type { Stream } from '../types/streams'
import type { OfficerSurveyResponse, Survey, SurveyResults } from '../types/surveys'

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

export const draftPost: EditablePost = {
  id: 'draft-1',
  title: 'Patch 11.0 notes',
  excerpt: 'What changed.',
  category: 'patch-notes',
  body: '## Changes',
  pinned: false,
  authorName: 'Officer',
  publishedAt: null,
  updatedAt: new Date().toISOString(),
}

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
    isLiveOtherGame: false,
    platform: 'twitch',
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
    isLiveOtherGame: false,
    platform: 'twitch',
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
    isLiveOtherGame: false,
    platform: 'twitch',
  },
]

const submissions: Submission[] = [
  {
    id: 'sub-1',
    submitter: { id: 'user-1', username: 'thundermane' },
    description: 'Onyxia down!',
    originalFilename: 'onyxia.png',
    contentType: 'image/png',
    width: 1920,
    height: 1080,
    byteSize: 2_400_000,
    createdAt: '2026-10-26T12:00:00Z',
    thumbUrl: '/api/submissions/sub-1/thumb',
    fullUrl: '/api/submissions/sub-1/full',
    downloadUrl: '/api/submissions/sub-1/download',
  },
  {
    id: 'sub-2',
    submitter: { id: 'user-2', username: 'mendwell' },
    description: '',
    originalFilename: 'fanart.jpg',
    contentType: 'image/jpeg',
    width: 800,
    height: 1200,
    byteSize: 900_000,
    createdAt: '2026-10-26T11:00:00Z',
    thumbUrl: '/api/submissions/sub-2/thumb',
    fullUrl: '/api/submissions/sub-2/full',
    downloadUrl: '/api/submissions/sub-2/download',
  },
]

const myCharacters: MyCharacter[] = [
  {
    id: 'mine-1',
    name: 'Thundermane',
    secondaryName: 'Stormcaller',
    realm: 'Emberreach',
    class: 'Warrior',
    spec: 'Protection',
    role: 'tank',
    spec2: null,
    role2: null,
    isMain: true,
    raidTeam: 'Team Alpha',
    race: 'Orc',
    level: 60,
    faction: 'Horde',
    createdAt: '2026-10-01T12:00:00Z',
    professions: [{ id: 'mine-1-mining', profession: 'Mining', skillLevel: 300 }],
  },
  {
    id: 'mine-2',
    name: 'Thundermend',
    secondaryName: 'Lightward',
    realm: 'Emberreach',
    class: 'Priest',
    spec: 'Holy',
    role: 'healer',
    spec2: null,
    role2: null,
    isMain: false,
    raidTeam: null,
    race: 'Undead',
    level: 52,
    faction: 'Horde',
    createdAt: '2026-10-01T12:00:00Z',
    professions: [],
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
    spec2: null,
    role2: null,
    isMain: true,
    raidTeam: 'Team Alpha',
    race: 'Orc',
    level: 60,
    faction: 'Horde',
    createdAt: new Date().toISOString(),
    professions: [{ profession: 'Mining', skillLevel: 300 }, { profession: 'Blacksmithing', skillLevel: 275 }],
  },
  {
    id: 'char-2',
    name: 'Korrath',
    secondaryName: '',
    realm: 'Emberreach',
    class: 'Paladin',
    spec: 'Protection',
    role: 'tank',
    spec2: null,
    role2: null,
    isMain: true,
    raidTeam: 'Team Beta',
    race: 'Tauren',
    level: 58,
    faction: 'Horde',
    createdAt: new Date().toISOString(),
    professions: [{ profession: 'Herbalism', skillLevel: 225 }, { profession: 'Alchemy', skillLevel: 150 }],
  },
  {
    id: 'char-3',
    name: 'Selene',
    secondaryName: '',
    realm: 'Emberreach',
    class: 'Priest',
    spec: 'Holy',
    role: 'healer',
    spec2: null,
    role2: null,
    isMain: true,
    raidTeam: 'Team Alpha',
    race: 'Undead',
    level: 60,
    faction: 'Horde',
    createdAt: new Date().toISOString(),
    professions: [],
  },
  {
    id: 'char-4',
    name: 'Mirelle',
    secondaryName: '',
    realm: 'Emberreach',
    class: 'Druid',
    spec: 'Restoration',
    role: 'healer',
    spec2: null,
    role2: null,
    isMain: true,
    raidTeam: 'Team Beta',
    race: null,
    level: null,
    faction: null,
    createdAt: new Date().toISOString(),
    professions: [],
  },
  {
    id: 'char-5',
    name: 'Zaldrin',
    secondaryName: '',
    realm: 'Emberreach',
    class: 'Mage',
    spec: 'Fire',
    role: 'dps',
    spec2: null,
    role2: null,
    isMain: true,
    raidTeam: 'Team Alpha',
    race: null,
    level: null,
    faction: null,
    createdAt: new Date().toISOString(),
    professions: [],
  },
  {
    id: 'char-6',
    name: 'Zaldrix',
    secondaryName: '',
    realm: 'Emberreach',
    class: 'Warlock',
    spec: 'Affliction',
    role: 'dps',
    spec2: null,
    role2: null,
    isMain: false,
    raidTeam: null,
    race: null,
    level: null,
    faction: null,
    createdAt: new Date().toISOString(),
    professions: [],
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

const professionEntries: ProfessionEntry[] = [
  {
    id: 'profession-1',
    profession: 'Alchemy',
    skillLevel: 300,
    character: { id: 'char-1', name: 'Aeliana', secondaryName: 'Dawnsong', class: 'Priest' },
  },
  {
    id: 'profession-2',
    profession: 'Blacksmithing',
    skillLevel: 225,
    character: { id: 'char-2', name: 'Zephyrion', secondaryName: 'Ironhide', class: 'Warrior' },
  },
]

export const pendingApplication: Application = {
  id: '11111111-1111-4111-8111-111111111111',
  applicantName: 'Mira',
  characterName: 'Thornleaf',
  class: 'Druid',
  role: 'healer',
  playstyle: 'raider',
  availability: 'Tue/Thu 8-11pm ET',
  discordHandle: 'mira.heals',
  notes: 'Raided Karazhan on my old realm.\nHappy to reroll if needed.',
  status: 'pending',
  submittedAt: '2026-10-06T18:30:00Z',
  reviewedBy: null,
  reviewedAt: null,
  reviewNote: '',
}

export const acceptedApplication: Application = {
  ...pendingApplication,
  id: '22222222-2222-4222-8222-222222222222',
  applicantName: 'Brek',
  characterName: 'Ironhide',
  class: 'Warrior',
  role: 'tank',
  playstyle: 'casual',
  status: 'accepted',
  submittedAt: '2026-10-04T12:00:00Z',
  reviewedBy: '33333333-3333-4333-8333-333333333333',
  reviewedAt: '2026-10-05T09:00:00Z',
  reviewNote: 'Great fit for tank spot',
}

const applications = [pendingApplication, acceptedApplication]

export const openSurvey: Survey = {
  id: 'survey-1',
  title: 'Spring roster',
  description: 'Help us plan the raid teams',
  closesAt: new Date(Date.now() + 5 * 86_400_000).toISOString(),
  status: 'open',
  questions: [
    { id: 'q-class', prompt: 'Class', type: 'single', required: true, options: ['Mage', 'Priest', 'Warrior'] },
    {
      id: 'q-secondary',
      prompt: 'Secondary class',
      type: 'single',
      required: false,
      options: ['Mage', 'Priest', 'Warrior'],
    },
    {
      id: 'q-nights',
      prompt: 'Raid nights',
      type: 'multi',
      required: true,
      options: ['Tuesday', 'Wednesday', 'Thursday'],
    },
  ],
  myResponse: null,
}

export const closedSurvey: Survey = {
  ...openSurvey,
  closesAt: new Date(Date.now() - 86_400_000).toISOString(),
  status: 'closed',
}

export const closedSurveyResults: SurveyResults = {
  surveyId: 'survey-1',
  title: 'Spring roster',
  turnout: 3,
  questions: [
    {
      id: 'q-class',
      prompt: 'Class',
      type: 'single',
      options: [
        { label: 'Mage', count: 2 },
        { label: 'Priest', count: 0 },
        { label: 'Warrior', count: 1 },
      ],
    },
    {
      id: 'q-nights',
      prompt: 'Raid nights',
      type: 'multi',
      options: [
        { label: 'Tuesday', count: 3 },
        { label: 'Wednesday', count: 1 },
        { label: 'Thursday', count: 0 },
      ],
    },
  ],
}

export const liveSurveyResults: SurveyResults = {
  surveyId: 'survey-1',
  title: 'Spring roster',
  turnout: 2,
  questions: [
    {
      id: 'q-class',
      prompt: 'Class',
      type: 'single',
      options: [
        { label: 'Mage', count: 1 },
        { label: 'Priest', count: 1 },
        { label: 'Warrior', count: 0 },
      ],
    },
    {
      id: 'q-nights',
      prompt: 'Raid nights',
      type: 'multi',
      options: [
        { label: 'Tuesday', count: 2 },
        { label: 'Wednesday', count: 0 },
        { label: 'Thursday', count: 1 },
      ],
    },
  ],
}

export const officerSurveyResponses: OfficerSurveyResponse[] = [
  {
    id: 'response-1',
    discordUsername: 'ana',
    guildName: 'Anaria',
    inDiscordServer: true,
    submittedAt: '2026-03-10T12:00:00Z',
    updatedAt: '2026-03-10T12:00:00Z',
    voidedAt: null,
    voidedBy: null,
    voidReason: '',
    answers: [
      { questionId: 'q-class', values: ['Mage'] },
      { questionId: 'q-nights', values: ['Tuesday', 'Thursday'] },
    ],
  },
  {
    id: 'response-2',
    discordUsername: 'ben',
    guildName: '',
    inDiscordServer: false,
    submittedAt: '2026-03-10T13:00:00Z',
    updatedAt: '2026-03-10T13:00:00Z',
    voidedAt: '2026-03-11T09:00:00Z',
    voidedBy: 'Officer',
    voidReason: 'joke answers',
    answers: [
      { questionId: 'q-class', values: ['Priest'] },
      { questionId: 'q-nights', values: ['Wednesday'] },
    ],
  },
]

export const handlers = [
  http.post('/api/applications', () => HttpResponse.json({ id: 'app-1' }, { status: 201 })),
  http.get('/api/applications', ({ request }) => {
    const status = new URL(request.url).searchParams.get('status')
    return HttpResponse.json(status ? applications.filter((application) => application.status === status) : applications)
  }),
  http.patch('/api/applications/:id', async ({ params, request }) => {
    const body = (await request.json()) as { status: ApplicationStatus; reviewNote?: string }
    const application = applications.find((candidate) => candidate.id === params.id)
    return HttpResponse.json({ ...application, status: body.status, reviewNote: body.reviewNote ?? '' })
  }),
  http.get('/api/auth/me', () => HttpResponse.json({ error: 'login required' }, { status: 401 })),
  http.post('/api/auth/logout', () => new HttpResponse(null, { status: 204 })),
  newsHandler(newsPosts),
  http.get('/api/news/drafts', () => HttpResponse.json([draftPost])),
  http.get('/api/news/:id', ({ params }) => {
    const post = newsPosts.find((candidate) => candidate.id === params.id)
    return post
      ? HttpResponse.json({ ...post, body: '', pinned: false, updatedAt: post.publishedAt })
      : HttpResponse.json({ error: 'news post not found' }, { status: 404 })
  }),
  http.post('/api/news', () => HttpResponse.json(draftPost, { status: 201 })),
  http.patch('/api/news/:id', () => HttpResponse.json(draftPost)),
  http.post('/api/news/:id/publish', () =>
    HttpResponse.json({ ...draftPost, publishedAt: new Date().toISOString() }),
  ),
  http.delete('/api/news/:id', () => new HttpResponse(null, { status: 204 })),
  http.get('/api/professions', () => HttpResponse.json(professionEntries)),
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
  http.get('/api/me/characters', () => HttpResponse.json(myCharacters)),
  http.patch('/api/me/characters/:id', async ({ params, request }) => {
    const body = (await request.json()) as { isMain: boolean }
    const character = myCharacters.find((candidate) => candidate.id === params.id)
    return HttpResponse.json({ ...character, isMain: body.isMain })
  }),
  http.post('/api/me/characters/:id/professions', async ({ params, request }) => {
    const body = (await request.json()) as { profession: string; skillLevel: number }
    const character = myCharacters.find((candidate) => candidate.id === params.id)
    return HttpResponse.json(
      { id: 'new-profession', ...body, character: { id: character?.id, name: character?.name, secondaryName: character?.secondaryName, class: character?.class } },
      { status: 201 },
    )
  }),
  http.patch('/api/me/professions/:id', async ({ params, request }) => {
    const body = (await request.json()) as { skillLevel?: number }
    return HttpResponse.json({ id: params.id, profession: 'Mining', skillLevel: body.skillLevel ?? 300 })
  }),
  http.delete('/api/me/professions/:id', () => new HttpResponse(null, { status: 204 })),
  http.get('/api/roster', () => HttpResponse.json(rosterCharacters)),
  http.post('/api/roster', () => HttpResponse.json(rosterCharacters[0], { status: 201 })),
  http.patch('/api/roster/:id', () => HttpResponse.json(rosterCharacters[0])),
  http.delete('/api/roster/:id', () => new HttpResponse(null, { status: 204 })),
  http.get('/api/streams', () => HttpResponse.json(allStreams)),
  http.get('/api/streams/live', () => HttpResponse.json(liveStreams)),
  http.get('/api/streams/suggested-video', () =>
    HttpResponse.json({ id: 'vid-1', title: 'Queen Ansurek kill' }),
  ),
  http.get('/api/submissions/status', () => HttpResponse.json({ open: true, attestationVersion: '2026-10-26' })),
  http.post('/api/submissions', () =>
    HttpResponse.json({ results: [{ filename: 'shot.png', submission: submissions[0] }] }, { status: 201 }),
  ),
  http.get('/api/submissions', () =>
    HttpResponse.json({ total: submissions.length, limit: 48, offset: 0, items: submissions }),
  ),
  http.delete('/api/submissions/:id', () => new HttpResponse(null, { status: 204 })),
  http.post('/api/submissions/bulk-delete', async ({ request }) => {
    const body = (await request.json()) as { ids: string[] }
    return HttpResponse.json({ deleted: body.ids })
  }),
  http.get('/api/surveys', () =>
    HttpResponse.json([
      { id: openSurvey.id, title: openSurvey.title, closesAt: openSurvey.closesAt, status: 'open' },
    ]),
  ),
  http.get('/api/surveys/:id', () => HttpResponse.json(openSurvey)),
  http.put('/api/surveys/:id/response', async ({ request }) => HttpResponse.json(await request.json())),
  http.get('/api/surveys/:id/results', () => HttpResponse.json(closedSurveyResults)),
  http.post('/api/surveys', () => HttpResponse.json(openSurvey, { status: 201 })),
  http.patch('/api/surveys/:id', () => HttpResponse.json(openSurvey)),
  http.post('/api/surveys/:id/questions/:questionId/options', () => HttpResponse.json(openSurvey)),
  http.get('/api/surveys/:id/live-results', () => HttpResponse.json(liveSurveyResults)),
  http.get('/api/surveys/:id/responses', () => HttpResponse.json(officerSurveyResponses)),
  http.patch('/api/surveys/:id/responses/:responseId', () => HttpResponse.json(officerSurveyResponses[0])),
]
