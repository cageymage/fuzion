export type NewsCategory = 'raid-progress' | 'recruitment' | 'guild-news' | 'patch-notes'

export interface NewsPost {
  id: string
  title: string
  excerpt: string
  category: NewsCategory
  imageUrl: string | null
  authorName: string
  publishedAt: string
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
