import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Link } from 'react-router-dom'
import { deleteSubmission, deleteSubmissions, fetchSubmissions } from '../../api/submissions'
import { Dialog } from '../../components/Dialog/Dialog'
import { LightboxImage } from '../../components/LightboxImage/LightboxImage'
import { OfficerOnly } from '../../components/OfficerOnly/OfficerOnly'
import type { Submission } from '../../types/submissions'
import styles from './OfficerSubmissions.module.css'

export const PAGE_SIZE = 48

function describe(submission: Submission): string {
  return submission.description || submission.originalFilename
}

function useDeleteMany(onDeleted: () => void) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (ids: string[]) =>
      ids.length === 1 ? deleteSubmission(ids[0]).then(() => ({ deleted: ids })) : deleteSubmissions(ids),
    onSuccess: async () => {
      onDeleted()
      await queryClient.invalidateQueries({ queryKey: ['submissions', 'list'] })
    },
  })
}

function ConfirmDelete({ count, onConfirm, onCancel, pending, error }: {
  count: number
  onConfirm: () => void
  onCancel: () => void
  pending: boolean
  error: Error | null
}) {
  const label = count === 1 ? 'this submission' : `these ${count} submissions`
  return (
    <Dialog title={count === 1 ? 'Delete submission?' : `Delete ${count} submissions?`} onClose={onCancel}>
      <p>This permanently deletes {label}: the original file and its preview. It cannot be undone.</p>
      {error && (
        <p role="alert" className={styles.error}>
          {error.message}
        </p>
      )}
      <div className={styles.actions}>
        <button type="button" className={styles.danger} onClick={onConfirm} disabled={pending}>
          {pending ? 'Deleting…' : 'Delete'}
        </button>
        <button type="button" className={styles.button} onClick={onCancel} disabled={pending}>
          Cancel
        </button>
      </div>
    </Dialog>
  )
}

function SubmissionGallery() {
  const [offset, setOffset] = useState(0)
  const [selecting, setSelecting] = useState(false)
  const [checked, setChecked] = useState<Set<string>>(new Set())
  const [confirming, setConfirming] = useState<string[] | null>(null)
  const page = useQuery({
    queryKey: ['submissions', 'list', offset],
    queryFn: () => fetchSubmissions(PAGE_SIZE, offset),
    placeholderData: keepPreviousData,
  })
  const remove = useDeleteMany(() => {
    setConfirming(null)
    setChecked(new Set())
  })

  function toggle(id: string) {
    setChecked((current) => {
      const next = new Set(current)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  function stopSelecting() {
    setSelecting(false)
    setChecked(new Set())
  }

  if (page.isPending) return <p className={styles.notice}>Loading submissions…</p>
  if (page.isError) {
    return (
      <p role="alert" className={styles.error}>
        The submissions could not be loaded.
      </p>
    )
  }

  const { items, total } = page.data
  const lastPageOffset = Math.max(0, Math.floor((total - 1) / PAGE_SIZE) * PAGE_SIZE)
  if (offset > lastPageOffset) {
    setOffset(lastPageOffset)
    return <p className={styles.notice}>Loading submissions…</p>
  }
  const lastShown = Math.min(offset + items.length, total)

  return (
    <>
      <div className={styles.toolbar}>
        {selecting ? (
          <>
            <button
              type="button"
              className={styles.danger}
              disabled={checked.size === 0}
              onClick={() => setConfirming([...checked])}
            >
              Delete selected ({checked.size})
            </button>
            <button type="button" className={styles.button} onClick={stopSelecting}>
              Cancel selection
            </button>
          </>
        ) : (
          <button type="button" className={styles.button} onClick={() => setSelecting(true)} disabled={total === 0}>
            Select
          </button>
        )}
        {total > 0 && (
          <span className={styles.hint}>
            Showing {offset + 1}–{lastShown} of {total}
          </span>
        )}
      </div>

      {total === 0 ? (
        <p className={styles.notice}>No submissions yet.</p>
      ) : (
        <ul className={styles.grid} aria-label="Submissions">
          {items.map((submission) => (
            <li key={submission.id} className={styles.card}>
              {selecting && (
                <input
                  type="checkbox"
                  className={styles.check}
                  checked={checked.has(submission.id)}
                  onChange={() => toggle(submission.id)}
                  aria-label={`Select ${describe(submission)}`}
                />
              )}
              {selecting ? (
                <button
                  type="button"
                  className={styles.thumbButton}
                  onClick={() => toggle(submission.id)}
                  aria-label={`Toggle ${describe(submission)}`}
                >
                  <img className={styles.thumb} src={submission.thumbUrl} alt="" loading="lazy" />
                </button>
              ) : (
                <LightboxImage src={submission.fullUrl} thumbSrc={submission.thumbUrl} alt={describe(submission)} />
              )}
              <div className={styles.cardBody}>
                <span className={styles.submitter}>{submission.submitter.username}</span>
                {submission.description && <span className={styles.description}>{submission.description}</span>}
              </div>
              {!selecting && (
                <div className={styles.actions}>
                  <a
                    className={styles.button}
                    href={submission.downloadUrl}
                    download={submission.originalFilename}
                    aria-label={`Download original of ${describe(submission)}`}
                  >
                    Download
                  </a>
                  <button
                    type="button"
                    className={styles.button}
                    onClick={() => setConfirming([submission.id])}
                    aria-label={`Delete ${describe(submission)}`}
                  >
                    Delete
                  </button>
                </div>
              )}
            </li>
          ))}
        </ul>
      )}

      {total > PAGE_SIZE && (
        <nav className={styles.pager} aria-label="Pages">
          <button
            type="button"
            className={styles.button}
            disabled={offset === 0}
            onClick={() => setOffset(Math.max(0, offset - PAGE_SIZE))}
          >
            Previous
          </button>
          <button
            type="button"
            className={styles.button}
            disabled={offset + PAGE_SIZE >= total}
            onClick={() => setOffset(offset + PAGE_SIZE)}
          >
            Next
          </button>
        </nav>
      )}

      {confirming && (
        <ConfirmDelete
          count={confirming.length}
          pending={remove.isPending}
          error={remove.error}
          onConfirm={() => remove.mutate(confirming)}
          onCancel={() => {
            setConfirming(null)
            remove.reset()
          }}
        />
      )}
    </>
  )
}

export function OfficerSubmissions() {
  return (
    <OfficerOnly>
      <section className={styles.page}>
        <h1 className={styles.title}>Review submissions</h1>
        <div className={styles.links}>
          <Link to="/submissions">
            Back to the submission form
          </Link>
        </div>
        <SubmissionGallery />
      </section>
    </OfficerOnly>
  )
}
