import type { NewsPage, NewsPost } from '../types/news'
import { apiGet } from './client'

// The API caps a page at 50 posts. Until the News page pages through results,
// the largest page is what keeps the archive complete.
const maxNewsPageSize = 50

async function fetchNewsPosts(limit: number): Promise<NewsPost[]> {
  const page = await apiGet<NewsPage>(`/news?limit=${limit}`)
  return page._embedded.news
}

export function fetchNews(): Promise<NewsPost[]> {
  return fetchNewsPosts(maxNewsPageSize)
}

export function fetchLatestNews(limit: number): Promise<NewsPost[]> {
  return fetchNewsPosts(limit)
}
