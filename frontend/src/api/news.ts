import type { EditablePost, NewsCategory, NewsPage, NewsPost, PostFields } from '../types/news'
import { apiGet, apiSend } from './client'

interface NewsPageQuery {
  category?: NewsCategory
  page: number
  pageSize: number
}

export async function fetchNewsPage({
  category,
  page,
  pageSize,
}: NewsPageQuery): Promise<{ posts: NewsPost[]; total: number }> {
  const query = new URLSearchParams({
    limit: String(pageSize),
    offset: String((page - 1) * pageSize),
  })
  if (category) query.set('category', category)

  const result = await apiGet<NewsPage>(`/news?${query}`)
  return { posts: result._embedded.news, total: result.total }
}

export async function fetchLatestNews(limit: number): Promise<NewsPost[]> {
  const result = await apiGet<NewsPage>(`/news?limit=${limit}`)
  return result._embedded.news
}

export function fetchDrafts(): Promise<EditablePost[]> {
  return apiGet<EditablePost[]>('/news/drafts')
}

// Drafts are not served by GET /news/{id}, so look there first.
export async function fetchEditablePost(id: string): Promise<EditablePost> {
  const drafts = await fetchDrafts()
  return drafts.find((draft) => draft.id === id) ?? apiGet<EditablePost>(`/news/${id}`)
}

export function createDraft(): Promise<EditablePost> {
  const fields: PostFields = {
    title: 'Untitled post',
    excerpt: '',
    category: 'guild-news',
    body: '',
    pinned: false,
  }
  return apiSend<EditablePost>('POST', '/news', fields)
}

export function saveNewsPost(id: string, fields: PostFields): Promise<EditablePost> {
  return apiSend<EditablePost>('PATCH', `/news/${id}`, fields)
}

export function publishNewsPost(id: string): Promise<EditablePost> {
  return apiSend<EditablePost>('POST', `/news/${id}/publish`, {})
}
