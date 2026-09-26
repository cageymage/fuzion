import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { fetchNews } from '../../api/news'
import { NewsCard } from '../../components/NewsCard/NewsCard'
import { newsCategoryMeta } from '../../lib/newsCategory'
import type { NewsCategory } from '../../types/news'
import styles from './News.module.css'

type CategoryFilter = NewsCategory | 'all'

const filters: { value: CategoryFilter; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'guild-news', label: newsCategoryMeta('guild-news').label },
  { value: 'raid-progress', label: newsCategoryMeta('raid-progress').label },
  { value: 'recruitment', label: newsCategoryMeta('recruitment').label },
]

export function News() {
  const news = useQuery({ queryKey: ['news'], queryFn: fetchNews })
  const [category, setCategory] = useState<CategoryFilter>('all')

  const visiblePosts = (news.data ?? []).filter(
    (post) => category === 'all' || post.category === category,
  )

  return (
    <section className={styles.page}>
      <h1 className={styles.title}>News</h1>
      <div className={styles.filters}>
        {filters.map(({ value, label }) => (
          <button
            key={value}
            type="button"
            className={styles.filter}
            aria-pressed={category === value}
            onClick={() => setCategory(value)}
          >
            {label}
          </button>
        ))}
      </div>
      {news.isPending && <p className={styles.noticeText}>Loading news…</p>}
      {news.isError && <p className={styles.noticeText}>News could not be loaded.</p>}
      {visiblePosts.map((post) => (
        <NewsCard key={post.id} post={post} />
      ))}
      {news.isSuccess && visiblePosts.length === 0 && (
        <p className={styles.noticeText}>No posts in this category yet.</p>
      )}
    </section>
  )
}
