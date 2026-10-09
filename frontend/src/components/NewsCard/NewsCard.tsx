import { Link } from 'react-router-dom'
import { formatRelativeDate } from '../../lib/format'
import { newsCategoryMeta } from '../../lib/newsCategory'
import type { NewsPost } from '../../types/news'
import { Chip } from '../Chip/Chip'
import styles from './NewsCard.module.css'

type CardPost = Pick<NewsPost, 'id' | 'title' | 'category' | 'authorName'> & {
  publishedAt: string | null
}

interface NewsCardProps {
  post: CardPost
}

export function NewsCard({ post }: NewsCardProps) {
  const { label, tone } = newsCategoryMeta(post.category)
  const { publishedAt } = post

  return (
    <article className={`card ${styles.card}`}>
      <Chip tone={tone}>{label}</Chip>
      <h3 className={styles.title}>
        <Link
          to={publishedAt === null ? `/officer/news/${post.id}` : `/news/${post.id}`}
          className={styles.titleLink}
        >
          {post.title}
        </Link>
      </h3>
      <div className={styles.meta}>
        {publishedAt === null
          ? `Draft by ${post.authorName}`
          : `Posted by ${post.authorName} · ${formatRelativeDate(publishedAt)}`}
      </div>
    </article>
  )
}
