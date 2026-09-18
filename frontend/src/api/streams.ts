import type { LiveStream } from '../types/streams'
import { apiGet } from './client'

export function fetchLiveStreams(): Promise<LiveStream[]> {
  return apiGet<LiveStream[]>('/streams/live')
}
