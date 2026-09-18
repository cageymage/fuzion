import { formatRelativeDate } from '../../lib/format'
import { newsCategoryMeta } from '../../lib/newsCategory'
import type { NewsPost } from '../../types/news'
import { Chip } from '../Chip/Chip'
import styles from './FeaturedNewsCard.module.css'

interface FeaturedNewsCardProps {
  post: NewsPost
}

export function FeaturedNewsCard({ post }: FeaturedNewsCardProps) {
  const { label, tone } = newsCategoryMeta(post.category)

  return (
    <article className={`card ${styles.card}`}>
      <div className={styles.thumbnail}>
        {post.imageUrl ? (
          <img className={styles.thumbnailImage} src={post.imageUrl} alt="" />
        ) : (
          <svg
            className={styles.placeholderTrend}
            viewBox="0 0 210 130"
            preserveAspectRatio="none"
            aria-hidden="true"
          >
            <path
              d="M18 112 L56 56 L84 84 L122 38 L150 74 L192 28"
              stroke="var(--epic)"
              strokeWidth="2"
              fill="none"
            />
          </svg>
        )}
        <span className={styles.expandBadge} aria-hidden="true">
          <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="var(--text)" strokeWidth="2">
            <path d="M9 3 H3 V9 M15 21 H21 V15 M21 3 L13 11 M3 21 L11 13" />
          </svg>
        </span>
      </div>
      <div className={styles.body}>
        <Chip tone={tone}>{label}</Chip>
        <h3 className={styles.title}>{post.title}</h3>
        <p className={styles.excerpt}>{post.excerpt}</p>
        <div className={styles.meta}>
          Posted by {post.authorName} · {formatRelativeDate(post.publishedAt)}
        </div>
      </div>
    </article>
  )
}
