import type { ChipTone } from '../components/Chip/Chip'
import type { NewsCategory } from '../types/news'

const categoryMeta: Record<NewsCategory, { label: string; tone: ChipTone }> = {
  'raid-progress': { label: 'Raid Progress', tone: 'epic' },
  recruitment: { label: 'Recruitment', tone: 'legendary' },
  'guild-news': { label: 'Guild News', tone: 'gold' },
}

export function newsCategoryMeta(category: NewsCategory): { label: string; tone: ChipTone } {
  return categoryMeta[category]
}
