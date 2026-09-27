import type { Stream } from '../types/streams'
import { apiGet } from './client'

export function fetchLiveStreams(): Promise<Stream[]> {
  return apiGet<Stream[]>('/streams/live')
}

export function fetchStreams(): Promise<Stream[]> {
  return apiGet<Stream[]>('/streams')
}
