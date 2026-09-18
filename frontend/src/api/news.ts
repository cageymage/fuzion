import type { NewsPost } from '../types/news'
import { apiGet } from './client'

export function fetchNews(): Promise<NewsPost[]> {
  return apiGet<NewsPost[]>('/news')
}
