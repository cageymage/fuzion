import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from 'react-router-dom'
import { createDraft, fetchDrafts, fetchNewsPage } from '../../api/news'
import { Chip } from '../../components/Chip/Chip'
import { OfficerOnly } from '../../components/OfficerOnly/OfficerOnly'
import { newsCategoryMeta } from '../../lib/newsCategory'
import type { NewsCategory } from '../../types/news'
import styles from './OfficerNews.module.css'

const publishedLimit = 50

interface PostRowProps {
  id: string
  title: string
  category: NewsCategory
}

function PostRow({ id, title, category }: PostRowProps) {
  const { label, tone } = newsCategoryMeta(category)

  return (
    <li className={`card ${styles.row}`}>
      <Chip tone={tone}>{label}</Chip>
      <Link to={`/officer/news/${id}`} className={styles.rowTitle}>
        {title}
      </Link>
    </li>
  )
}

function NewsLists() {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const drafts = useQuery({ queryKey: ['officer-news', 'drafts'], queryFn: fetchDrafts })
  const published = useQuery({
    queryKey: ['news', 'officer-published'],
    queryFn: () => fetchNewsPage({ page: 1, pageSize: publishedLimit }),
  })
  const newPost = useMutation({
    mutationFn: createDraft,
    onSuccess: async (post) => {
      await queryClient.invalidateQueries({ queryKey: ['officer-news', 'drafts'] })
      navigate(`/officer/news/${post.id}`)
    },
  })

  return (
    <>
      <button
        type="button"
        className={styles.button}
        disabled={newPost.isPending}
        onClick={() => newPost.mutate()}
      >
        New post
      </button>
      {newPost.isError && (
        <p role="alert" className={styles.error}>
          The new post could not be created.
        </p>
      )}

      <h2 className={styles.sectionTitle}>Drafts</h2>
      {drafts.isPending && <p className={styles.notice}>Loading drafts…</p>}
      {drafts.isError && <p className={styles.notice}>Drafts could not be loaded.</p>}
      {drafts.isSuccess && drafts.data.length === 0 && <p className={styles.notice}>No drafts.</p>}
      {drafts.isSuccess && (
        <ul className={styles.list}>
          {drafts.data.map((post) => (
            <PostRow key={post.id} id={post.id} title={post.title} category={post.category} />
          ))}
        </ul>
      )}

      <h2 className={styles.sectionTitle}>Published</h2>
      {published.isPending && <p className={styles.notice}>Loading published posts…</p>}
      {published.isError && <p className={styles.notice}>Published posts could not be loaded.</p>}
      {published.isSuccess && published.data.posts.length === 0 && (
        <p className={styles.notice}>No published posts.</p>
      )}
      {published.isSuccess && (
        <ul className={styles.list}>
          {published.data.posts.map((post) => (
            <PostRow key={post.id} id={post.id} title={post.title} category={post.category} />
          ))}
        </ul>
      )}
    </>
  )
}

export function OfficerNews() {
  return (
    <OfficerOnly>
      <section className={styles.page}>
        <h1 className={styles.title}>Manage News</h1>
        <NewsLists />
      </section>
    </OfficerOnly>
  )
}
