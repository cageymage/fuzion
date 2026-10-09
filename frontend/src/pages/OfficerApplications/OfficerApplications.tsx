import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { loginUrl } from '../../api/auth'
import { ApiError } from '../../api/client'
import type {
  Application,
  ApplicationStatus,
  ApplicationStatusFilter,
  ReviewDecision,
} from '../../types/applications'
import styles from './OfficerApplications.module.css'
import { useApplicationList, useReviewApplication } from './useOfficerApplications'

const maxReviewNoteLength = 2000

const filters: { value: ApplicationStatusFilter; label: string }[] = [
  { value: 'pending', label: 'Pending' },
  { value: 'accepted', label: 'Accepted' },
  { value: 'declined', label: 'Declined' },
  { value: 'all', label: 'All' },
]

const statusLabels: Record<ApplicationStatus, string> = {
  pending: 'Pending',
  accepted: 'Accepted',
  declined: 'Declined',
}

const roleLabels: Record<Application['role'], string> = { tank: 'Tank', healer: 'Healer', dps: 'DPS' }

const emptyMessages: Record<ApplicationStatusFilter, string> = {
  pending: 'No pending applications.',
  accepted: 'No accepted applications.',
  declined: 'No declined applications.',
  all: 'No applications yet.',
}

function formatSubmitted(isoDate: string): string {
  return new Date(isoDate).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
}

interface ReviewFormProps {
  application: Application
  onReviewed: (decision: ReviewDecision) => void
}

function ReviewForm({ application, onReviewed }: ReviewFormProps) {
  const review = useReviewApplication()
  const [reviewNote, setReviewNote] = useState(application.reviewNote)

  function decide(event: FormEvent, status: ReviewDecision) {
    event.preventDefault()
    review.mutate(
      { id: application.id, status, reviewNote: reviewNote.trim() },
      { onSuccess: () => onReviewed(status) },
    )
  }

  return (
    <form className={styles.reviewForm}>
      <label className={styles.field}>
        <span>Review note (optional)</span>
        <textarea
          rows={3}
          maxLength={maxReviewNoteLength}
          value={reviewNote}
          onChange={(event) => setReviewNote(event.target.value)}
        />
      </label>
      <div className={styles.actions}>
        <button
          type="submit"
          className={styles.button}
          disabled={review.isPending}
          onClick={(event) => decide(event, 'accepted')}
        >
          Accept
        </button>
        <button
          type="submit"
          className={`${styles.button} ${styles.dangerButton}`}
          disabled={review.isPending}
          onClick={(event) => decide(event, 'declined')}
        >
          Decline
        </button>
      </div>
      {review.isError && (
        <p role="alert" className={styles.error}>
          {review.error.message}
        </p>
      )}
    </form>
  )
}

interface ApplicationRowProps {
  application: Application
  expanded: boolean
  onToggle: () => void
  onReviewed: (decision: ReviewDecision) => void
}

function ApplicationRow({ application, expanded, onToggle, onReviewed }: ApplicationRowProps) {
  const detailsId = `application-details-${application.id}`

  return (
    <li className={`card ${styles.row}`}>
      <button
        type="button"
        className={styles.rowHeader}
        aria-expanded={expanded}
        aria-controls={detailsId}
        onClick={onToggle}
      >
        <span className={styles.applicant}>{application.applicantName}</span>
        <span>{application.characterName}</span>
        <span>
          {application.class} · {roleLabels[application.role]}
        </span>
        <span className={styles.muted}>{formatSubmitted(application.submittedAt)}</span>
        <span className={`${styles.status} ${styles[application.status]}`}>
          {statusLabels[application.status]}
        </span>
      </button>
      {expanded && (
        <div id={detailsId} className={styles.details}>
          <dl className={styles.detailList}>
            <dt>Availability</dt>
            <dd>{application.availability}</dd>
            <dt>Discord</dt>
            <dd>{application.discordHandle}</dd>
            <dt>Notes</dt>
            <dd className={styles.notes}>{application.notes || 'None'}</dd>
          </dl>
          <ReviewForm application={application} onReviewed={onReviewed} />
        </div>
      )}
    </li>
  )
}

function AccessDenied({ anonymous }: { anonymous: boolean }) {
  return (
    <div className={styles.noticeText}>
      <p>Only officers can review applications.</p>
      {anonymous && (
        // A plain anchor, not a router Link: the server has to answer this one so it can redirect to Discord.
        <a href={loginUrl}>Log in with Discord</a>
      )}
    </div>
  )
}

export function OfficerApplications() {
  const [filter, setFilter] = useState<ApplicationStatusFilter>('pending')
  const [expandedId, setExpandedId] = useState<string | null>(null)
  const [lastReview, setLastReview] = useState<string | null>(null)
  const applications = useApplicationList(filter)

  if (applications.error instanceof ApiError && [401, 403].includes(applications.error.status)) {
    return (
      <section className={styles.page}>
        <h1 className={styles.title}>Review Applications</h1>
        <AccessDenied anonymous={applications.error.status === 401} />
      </section>
    )
  }

  return (
    <section className={styles.page}>
      <h1 className={styles.title}>Review Applications</h1>
      <Link to="/applications" className={styles.backLink}>
        Back to Applications
      </Link>

      <label className={styles.filter}>
        <span>Status</span>
        <select
          value={filter}
          onChange={(event) => {
            setFilter(event.target.value as ApplicationStatusFilter)
            setExpandedId(null)
          }}
        >
          {filters.map(({ value, label }) => (
            <option key={value} value={value}>
              {label}
            </option>
          ))}
        </select>
      </label>

      {lastReview && (
        <p role="status" className={styles.noticeText}>
          {lastReview}
        </p>
      )}
      {applications.isPending && <p className={styles.noticeText}>Loading applications…</p>}
      {applications.isError && <p className={styles.noticeText}>Applications could not be loaded.</p>}
      {applications.isSuccess && applications.data.length === 0 && (
        <p className={styles.noticeText}>{emptyMessages[filter]}</p>
      )}
      {applications.isSuccess && (
        <ul className={styles.list}>
          {applications.data.map((application) => (
            <ApplicationRow
              key={application.id}
              application={application}
              expanded={expandedId === application.id}
              onToggle={() => setExpandedId(expandedId === application.id ? null : application.id)}
              onReviewed={(decision) => {
                setExpandedId(null)
                setLastReview(`${application.characterName} was ${decision}.`)
              }}
            />
          ))}
        </ul>
      )}
    </section>
  )
}
