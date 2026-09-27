import type { NewsPost } from '../types/news'
import { apiGet } from './client'

export function fetchNews(): Promise<NewsPost[]> {
  return apiGet<NewsPost[]>('/news')
}

export function fetchLatestNews(limit: number): Promise<NewsPost[]> {
  return apiGet<NewsPost[]>(`/news?limit=${limit}`)
}
