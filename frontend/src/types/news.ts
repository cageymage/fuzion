export type NewsCategory = 'raid-progress' | 'recruitment' | 'guild-news' | 'patch-notes'

export interface NewsPost {
  id: string
  title: string
  excerpt: string
  category: NewsCategory
  imageUrl: string | null
  authorName: string
  publishedAt: string
  body?: string
}

export interface NewsLink {
  href: string
}

export interface NewsPage {
  total: number
  _links: Partial<Record<'self' | 'first' | 'prev' | 'next' | 'last', NewsLink>>
  _embedded: { news: NewsPost[] }
}

export interface EditablePost {
  id: string
  title: string
  excerpt: string
  category: NewsCategory
  body: string
  pinned: boolean
  imageUrl?: string | null
  authorName: string
  publishedAt: string | null
  updatedAt: string
}

export interface PostFields {
  title: string
  excerpt: string
  category: NewsCategory
  body: string
  pinned: boolean
}

export interface NewsPostDetail extends NewsPost {
  body: string
}
