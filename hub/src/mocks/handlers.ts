import { http, HttpResponse } from 'msw'
import type { Stream } from '../types/streams'

export const liveStream: Stream = {
  id: 'stream-1',
  streamerName: 'Thundermane',
  gameName: 'World of Warcraft: Forever',
  title: 'Mythic Queen Ansurek progress',
  viewerCount: 1_240,
  thumbnailUrl: 'https://cdn.example/thundermane-preview.jpg',
  avatarUrl: 'https://cdn.example/thundermane.png',
  channelUrl: 'https://twitch.tv/thundermane',
  isLive: true,
  isLiveOtherGame: false,
  platform: 'twitch',
}

export const handlers = [
  http.get('/api/streams/live', () => HttpResponse.json([liveStream])),
  http.get('/api/streams/suggested-video', () =>
    HttpResponse.json({ id: 'vid-1', title: 'Queen Ansurek kill' }),
  ),
]
