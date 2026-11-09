import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, type FormEvent } from 'react'
import { cancelRankRequest, createRankRequest } from '../../api/rankRequests'
import type { RankRequest, RankRequestDirection } from '../../types/rankRequests'
import type { Character } from '../../types/roster'
import styles from './Dashboard.module.css'
import { myRankRequestsQueryKey, rankRequestsQueryKey } from './myCharactersQueryKey'

const maxNoteLength = 500

const moveLabels: Record<RankRequestDirection, string> = {
  to_alt: 'move to an alt rank',
  to_main: 'move to a main rank',
}

function statusMessage(request: RankRequest): string {
  const move = moveLabels[request.direction]
  switch (request.status) {
    case 'open':
      return `You asked for a ${move}. An officer will change it in game.`
    case 'done':
      return `Your request for a ${move} is done.`
    case 'declined':
      return request.resolutionNote
        ? `Your request for a ${move} was declined: ${request.resolutionNote}`
        : `Your request for a ${move} was declined.`
  }
}

interface RequestFormProps {
  character: Character
  fullName: string
  onDone: () => void
}

function RequestForm({ character, fullName, onDone }: RequestFormProps) {
  const queryClient = useQueryClient()
  const [note, setNote] = useState('')
  const send = useMutation({
    mutationFn: () => createRankRequest(character.id, note),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: myRankRequestsQueryKey }),
        queryClient.invalidateQueries({ queryKey: rankRequestsQueryKey }),
      ])
      onDone()
    },
  })

  function submit(event: FormEvent) {
    event.preventDefault()
    send.mutate()
  }

  return (
    <form className={styles.requestForm} onSubmit={submit} aria-label={`Rank request for ${fullName}`}>
      <label className={styles.specField}>
        <span>Note for officers (optional)</span>
        <textarea
          className={styles.textarea}
          rows={2}
          maxLength={maxNoteLength}
          value={note}
          onChange={(event) => setNote(event.target.value)}
        />
      </label>
      <div className={styles.formActions}>
        <button type="submit" className={styles.button} disabled={send.isPending}>
          Send request
        </button>
        <button type="button" className={styles.button} onClick={onDone} disabled={send.isPending}>
          Cancel
        </button>
      </div>
      {send.isError && (
        <p role="alert" className={styles.error}>
          {send.error.message}
        </p>
      )}
    </form>
  )
}

interface RankRequestControlProps {
  character: Character
  fullName: string
  request: RankRequest | undefined
}

export function RankRequestControl({ character, fullName, request }: RankRequestControlProps) {
  const queryClient = useQueryClient()
  const [asking, setAsking] = useState(false)
  const cancel = useMutation({
    mutationFn: (id: string) => cancelRankRequest(id),
    onSuccess: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: myRankRequestsQueryKey }),
        queryClient.invalidateQueries({ queryKey: rankRequestsQueryKey }),
      ]),
  })
  const askLabel = character.isMain ? 'Request alt rank' : 'Request main rank'

  return (
    <div className={styles.rankRequest}>
      {request && <p className={styles.hint}>{statusMessage(request)}</p>}
      {request?.status === 'open' ? (
        <button
          type="button"
          className={styles.button}
          onClick={() => cancel.mutate(request.id)}
          disabled={cancel.isPending}
          aria-label={`Cancel rank request for ${fullName}`}
        >
          Cancel request
        </button>
      ) : asking ? (
        <RequestForm character={character} fullName={fullName} onDone={() => setAsking(false)} />
      ) : (
        <button
          type="button"
          className={styles.button}
          onClick={() => setAsking(true)}
          aria-label={`${askLabel} for ${fullName}`}
        >
          {askLabel}
        </button>
      )}
      {cancel.isError && (
        <p role="alert" className={styles.error}>
          {cancel.error.message}
        </p>
      )}
    </div>
  )
}
