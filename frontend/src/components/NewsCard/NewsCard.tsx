import { Link } from 'react-router-dom'
import { formatRelativeDate } from '../../lib/format'
import { newsCategoryMeta } from '../../lib/newsCategory'
import type { NewsPost } from '../../types/news'
import { Chip } from '../Chip/Chip'
import styles from './NewsCard.module.css'

interface NewsCardProps {
  post: NewsPost
}

export function NewsCard({ post }: NewsCardProps) {
  const { label, tone } = newsCategoryMeta(post.category)

  return (
    <article className={`card ${styles.card}`}>
      <Chip tone={tone}>{label}</Chip>
      <h3 className={styles.title}>
        <Link to={`/news/${post.id}`} className={styles.titleLink}>
          {post.title}
        </Link>
      </h3>
      <div className={styles.meta}>
        Posted by {post.authorName} · {formatRelativeDate(post.publishedAt)}
      </div>
    </article>
  )
}
