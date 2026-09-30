export type NewsCategory = 'raid-progress' | 'recruitment' | 'guild-news'

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
