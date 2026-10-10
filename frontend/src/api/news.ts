import type {
  EditablePost,
  NewsCategory,
  NewsPage,
  NewsPost,
  NewsPostDetail,
  PostFields,
} from '../types/news'
import { firstMarkdownImageUrl } from '../lib/markdown'
import { apiDelete, apiGet, apiSend } from './client'

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
  return { posts: result._embedded.news.map(withFirstImage), total: result.total }
}

// The API returns each post's Markdown body in the list; the first image in it becomes the card thumbnail.
function withFirstImage<T extends { imageUrl?: string | null; body?: string }>(post: T): T {
  return {
    ...post,
    imageUrl: post.imageUrl ?? (post.body ? firstMarkdownImageUrl(post.body) : null),
  }
}

export async function fetchLatestNews(limit: number): Promise<NewsPost[]> {
  const result = await apiGet<NewsPage>(`/news?limit=${limit}`)
  return result._embedded.news.map(withFirstImage)
}

export async function fetchDrafts(): Promise<EditablePost[]> {
  const drafts = await apiGet<EditablePost[]>('/news/drafts')
  return drafts.map(withFirstImage)
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

export function deleteNewsPost(id: string): Promise<void> {
  return apiDelete(`/news/${id}`)
}

export const editablePostKey = (id: string) => ['officer-news', 'post', id] as const

export function fetchNewsPost(id: string): Promise<NewsPostDetail> {
  return apiGet<NewsPostDetail>(`/news/${id}`)
}
