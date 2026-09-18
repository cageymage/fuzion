import { describe, expect, it } from 'vitest'
import { newsCategoryMeta } from './newsCategory'

describe('newsCategoryMeta', () => {
  it('should label raid progress with the epic tone', () => {
    expect(newsCategoryMeta('raid-progress')).toEqual({ label: 'Raid Progress', tone: 'epic' })
  })

  it('should label recruitment with the legendary tone', () => {
    expect(newsCategoryMeta('recruitment')).toEqual({ label: 'Recruitment', tone: 'legendary' })
  })

  it('should label guild news with the gold tone', () => {
    expect(newsCategoryMeta('guild-news')).toEqual({ label: 'Guild News', tone: 'gold' })
  })
})
