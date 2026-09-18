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
