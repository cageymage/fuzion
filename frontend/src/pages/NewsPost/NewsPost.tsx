import { useQuery } from '@tanstack/react-query'
import Markdown from 'react-markdown'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { fetchNewsPost } from '../../api/news'
import { ApiError } from '../../api/client'
import { Chip } from '../../components/Chip/Chip'
import { DeleteNewsPost } from '../../components/DeleteNewsPost/DeleteNewsPost'
import { LightboxImage } from '../../components/LightboxImage/LightboxImage'
import { useCurrentUser } from '../../hooks/useCurrentUser'
import { formatRelativeDate } from '../../lib/format'
import { newsCategoryMeta } from '../../lib/newsCategory'
import styles from './NewsPost.module.css'

export function NewsPostPage() {
  const { id = '' } = useParams()
  const navigate = useNavigate()
  const currentUser = useCurrentUser()
  const post = useQuery({ queryKey: ['news', 'post', id], queryFn: () => fetchNewsPost(id) })

  return (
    <article className={styles.page}>
      <Link to="/news" className={styles.backLink}>
        Back to all news
      </Link>

      {post.isPending && <p className={styles.noticeText}>Loading post…</p>}
      {post.isError && post.error instanceof ApiError && post.error.status === 404 && (
        <p className={styles.noticeText}>Post not found.</p>
      )}
      {post.isError && !(post.error instanceof ApiError && post.error.status === 404) && (
        <p className={styles.noticeText}>Post could not be loaded.</p>
      )}

      {post.isSuccess && (
        <>
          <div className={styles.categoryRow}>
            <Chip tone={newsCategoryMeta(post.data.category).tone}>
              {newsCategoryMeta(post.data.category).label}
            </Chip>
          </div>
          <h1 className={styles.title}>{post.data.title}</h1>
          <div className={styles.meta}>
            Posted by {post.data.authorName} · {formatRelativeDate(post.data.publishedAt)}
          </div>
          {currentUser.data?.isOfficer && (
            <div className={styles.officerActions}>
              <Link to={`/officer/news/${post.data.id}`} className={styles.editLink}>
                Edit
              </Link>
              <DeleteNewsPost postId={post.data.id} onDeleted={() => navigate('/news')} />
            </div>
          )}
          <div className={styles.body}>
            <Markdown components={{ img: LightboxImage }}>{post.data.body}</Markdown>
          </div>
        </>
      )}
    </article>
  )
}
