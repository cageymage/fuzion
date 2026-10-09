import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Navigate, useNavigate, useSearchParams } from 'react-router-dom'
import { createDraft, fetchDrafts, fetchNewsPage } from '../../api/news'
import { NewsCard } from '../../components/NewsCard/NewsCard'
import { useCurrentUser } from '../../hooks/useCurrentUser'
import { newsCategoryMeta } from '../../lib/newsCategory'
import type { NewsCategory } from '../../types/news'
import styles from './News.module.css'

type CategoryFilter = NewsCategory | 'all'

const pageSizes = [5, 10, 25, 50]
const defaultPageSize = pageSizes[0]

const filters: { value: CategoryFilter; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'guild-news', label: newsCategoryMeta('guild-news').label },
  { value: 'raid-progress', label: newsCategoryMeta('raid-progress').label },
  { value: 'recruitment', label: newsCategoryMeta('recruitment').label },
]

function toSearchParams(category: CategoryFilter, page: number, pageSize: number): URLSearchParams {
  const params = new URLSearchParams()
  if (category !== 'all') params.set('category', category)
  if (page > 1) params.set('page', String(page))
  if (pageSize !== defaultPageSize) params.set('pageSize', String(pageSize))
  return params
}

function OfficerTools() {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const drafts = useQuery({ queryKey: ['officer-news', 'drafts'], queryFn: fetchDrafts })
  const newPost = useMutation({
    mutationFn: createDraft,
    onSuccess: async (post) => {
      await queryClient.invalidateQueries({ queryKey: ['officer-news', 'drafts'] })
      navigate(`/officer/news/${post.id}`)
    },
  })

  return (
    <div className={styles.officerTools}>
      <button
        type="button"
        className={styles.newPostButton}
        disabled={newPost.isPending}
        onClick={() => newPost.mutate()}
      >
        New post
      </button>
      {newPost.isError && (
        <p role="alert" className={styles.errorText}>
          The new post could not be created.
        </p>
      )}
      {drafts.isError && <p className={styles.noticeText}>Drafts could not be loaded.</p>}
      {drafts.isSuccess && drafts.data.length > 0 && (
        <section aria-labelledby="news-drafts-heading">
          <h2 id="news-drafts-heading" className={styles.draftsTitle}>
            Drafts
          </h2>
          {drafts.data.map((draft) => (
            <NewsCard key={draft.id} post={draft} />
          ))}
        </section>
      )}
    </div>
  )
}

export function News() {
  const [searchParams, setSearchParams] = useSearchParams()
  const currentUser = useCurrentUser()
  const category =
    filters.find((filter) => filter.value === searchParams.get('category'))?.value ?? 'all'
  const requestedPage = Number(searchParams.get('page'))
  const page = Number.isInteger(requestedPage) && requestedPage >= 1 ? requestedPage : 1
  const requestedPageSize = Number(searchParams.get('pageSize'))
  const pageSize = pageSizes.includes(requestedPageSize) ? requestedPageSize : defaultPageSize

  const news = useQuery({
    queryKey: ['news', category, page, pageSize],
    queryFn: () =>
      fetchNewsPage({ category: category === 'all' ? undefined : category, page, pageSize }),
    placeholderData: keepPreviousData,
  })

  const posts = news.data?.posts ?? []
  const total = news.data?.total ?? 0
  const pageCount = Math.max(1, Math.ceil(total / pageSize))

  if (news.isSuccess && !news.isPlaceholderData && page > pageCount) {
    return <Navigate replace to={{ search: toSearchParams(category, pageCount, pageSize).toString() }} />
  }

  return (
    <section className={styles.page}>
      <h1 className={styles.title}>News</h1>
      {currentUser.data?.isOfficer && <OfficerTools />}
      <div className={styles.toolbar}>
        <div className={styles.filters}>
          {filters.map(({ value, label }) => (
            <button
              key={value}
              type="button"
              className={styles.filter}
              aria-pressed={category === value}
              onClick={() => setSearchParams(toSearchParams(value, 1, pageSize))}
            >
              {label}
            </button>
          ))}
        </div>
        <label className={styles.pageSizeField} htmlFor="news-page-size">
          Posts per page
          <select
            id="news-page-size"
            className={styles.pageSizeSelect}
            value={pageSize}
            onChange={(event) =>
              setSearchParams(toSearchParams(category, 1, Number(event.target.value)))
            }
          >
            {pageSizes.map((size) => (
              <option key={size} value={size}>
                {size}
              </option>
            ))}
          </select>
        </label>
      </div>
      {news.isPending && <p className={styles.noticeText}>Loading news…</p>}
      {news.isError && <p className={styles.noticeText}>News could not be loaded.</p>}
      {posts.map((post) => (
        <NewsCard key={post.id} post={post} />
      ))}
      {news.isSuccess && total === 0 && (
        <p className={styles.noticeText}>No posts in this category yet.</p>
      )}
      {news.isSuccess && total > 0 && (
        <nav className={styles.pager} aria-label="News pages">
          <button
            type="button"
            className={styles.pagerButton}
            disabled={page <= 1}
            onClick={() => setSearchParams(toSearchParams(category, page - 1, pageSize))}
          >
            Previous
          </button>
          <span className={styles.pageLabel}>
            Page {page} of {pageCount}
          </span>
          <button
            type="button"
            className={styles.pagerButton}
            disabled={page >= pageCount}
            onClick={() => setSearchParams(toSearchParams(category, page + 1, pageSize))}
          >
            Next
          </button>
        </nav>
      )}
    </section>
  )
}
