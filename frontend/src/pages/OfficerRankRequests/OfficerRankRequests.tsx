import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { ApiError } from '../../api/client'
import { useLoginUrl } from '../../hooks/useLoginUrl'
import type { RankRequest, RankRequestDecision, RankRequestDirection } from '../../types/rankRequests'
import styles from './OfficerRankRequests.module.css'
import { useRankRequestList, useResolveRankRequest } from './useOfficerRankRequests'

const maxNoteLength = 500

const directionLabels: Record<RankRequestDirection, string> = {
  to_alt: 'Main to alt rank',
  to_main: 'Alt to main rank',
}

function formatDate(isoDate: string): string {
  return new Date(isoDate).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
}

function fullName(request: RankRequest): string {
  return `${request.characterName} ${request.characterSecondaryName}`.trim()
}

function RequestSummary({ request }: { request: RankRequest }) {
  return (
    <dl className={styles.detailList}>
      <dt>Character</dt>
      <dd>
        {fullName(request)} ({request.characterClass})
      </dd>
      <dt>Request</dt>
      <dd>{directionLabels[request.direction]}</dd>
      <dt>Member</dt>
      <dd>{request.requestedBy}</dd>
      <dt>Asked</dt>
      <dd>{formatDate(request.createdAt)}</dd>
      <dt>Note</dt>
      <dd className={styles.notes}>{request.note || 'None'}</dd>
    </dl>
  )
}

interface OpenRequestProps {
  request: RankRequest
  onResolved: (decision: RankRequestDecision) => void
}

function OpenRequest({ request, onResolved }: OpenRequestProps) {
  const resolve = useResolveRankRequest()
  const [note, setNote] = useState('')

  function decide(event: FormEvent, status: RankRequestDecision) {
    event.preventDefault()
    resolve.mutate({ id: request.id, status, note: note.trim() }, { onSuccess: () => onResolved(status) })
  }

  return (
    <li className={`card ${styles.row}`} aria-label={`Rank request for ${fullName(request)}`}>
      <RequestSummary request={request} />
      <form className={styles.resolveForm}>
        <label className={styles.field}>
          <span>Note to the member (optional)</span>
          <textarea
            rows={2}
            maxLength={maxNoteLength}
            value={note}
            onChange={(event) => setNote(event.target.value)}
          />
        </label>
        <div className={styles.actions}>
          <button
            type="submit"
            className={styles.button}
            disabled={resolve.isPending}
            onClick={(event) => decide(event, 'done')}
          >
            Done
          </button>
          <button
            type="submit"
            className={`${styles.button} ${styles.dangerButton}`}
            disabled={resolve.isPending}
            onClick={(event) => decide(event, 'declined')}
          >
            Decline
          </button>
        </div>
        {resolve.isError && (
          <p role="alert" className={styles.error}>
            {resolve.error.message}
          </p>
        )}
      </form>
    </li>
  )
}

function resolution(request: RankRequest): string {
  const outcome = request.status === 'done' ? 'Done' : 'Declined'
  const by = request.resolvedBy ?? 'the roster sync'
  const when = request.resolvedAt ? ` on ${formatDate(request.resolvedAt)}` : ''
  const note = request.resolutionNote ? `: ${request.resolutionNote}` : ''
  return `${outcome} by ${by}${when}${note}`
}

function AccessDenied({ anonymous }: { anonymous: boolean }) {
  const loginUrl = useLoginUrl()
  return (
    <div className={styles.noticeText}>
      <p>Only officers can handle rank requests.</p>
      {anonymous && (
        // A plain anchor, not a router Link: the server has to answer this one so it can redirect to Discord.
        <a href={loginUrl}>Log in with Discord</a>
      )}
    </div>
  )
}

export function OfficerRankRequests() {
  const requests = useRankRequestList()
  const [lastResolved, setLastResolved] = useState<string | null>(null)

  if (requests.error instanceof ApiError && [401, 403].includes(requests.error.status)) {
    return (
      <section className={styles.page}>
        <h1 className={styles.title}>Rank Requests</h1>
        <AccessDenied anonymous={requests.error.status === 401} />
      </section>
    )
  }

  const open = requests.data?.filter((request) => request.status === 'open') ?? []
  const resolved = requests.data?.filter((request) => request.status !== 'open') ?? []

  return (
    <section className={styles.page}>
      <h1 className={styles.title}>Rank Requests</h1>
      <Link to="/dashboard" className={styles.backLink}>
        Back to Dashboard
      </Link>
      <p className={styles.noticeText}>
        Change the rank in game, then mark the request done. The site picks up the new rank on the
        next roster sync, which also closes requests it sees carried out.
      </p>

      {lastResolved && (
        <p role="status" className={styles.noticeText}>
          {lastResolved}
        </p>
      )}
      {requests.isPending && <p className={styles.noticeText}>Loading rank requests…</p>}
      {requests.isError && <p className={styles.noticeText}>Rank requests could not be loaded.</p>}

      {requests.isSuccess && (
        <>
          <h2 className={styles.subtitle}>Open</h2>
          {open.length === 0 ? (
            <p className={styles.noticeText}>No open rank requests.</p>
          ) : (
            <ul className={styles.list} aria-label="Open rank requests">
              {open.map((request) => (
                <OpenRequest
                  key={request.id}
                  request={request}
                  onResolved={(decision) =>
                    setLastResolved(
                      `The request for ${fullName(request)} was ${decision === 'done' ? 'marked done' : 'declined'}.`,
                    )
                  }
                />
              ))}
            </ul>
          )}

          <h2 className={styles.subtitle}>Recently resolved</h2>
          {resolved.length === 0 ? (
            <p className={styles.noticeText}>Nothing resolved yet.</p>
          ) : (
            <ul className={styles.list} aria-label="Resolved rank requests">
              {resolved.map((request) => (
                <li key={request.id} className={`card ${styles.row}`}>
                  <RequestSummary request={request} />
                  <p className={`${styles.resolution} ${styles[request.status]}`}>{resolution(request)}</p>
                </li>
              ))}
            </ul>
          )}
        </>
      )}
    </section>
  )
}
