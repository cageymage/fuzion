import type { Stream, SuggestedVideo } from '../types/streams'
import { apiGet, apiGetOptional } from './client'

export function fetchLiveStreams(): Promise<Stream[]> {
  return apiGet<Stream[]>('/streams/live')
}

export function fetchStreams(): Promise<Stream[]> {
  return apiGet<Stream[]>('/streams')
}

export function fetchSuggestedVideo(): Promise<SuggestedVideo | null> {
  return apiGetOptional<SuggestedVideo>('/streams/suggested-video')
}
