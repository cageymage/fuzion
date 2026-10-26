import { useState } from 'react'
import { Dialog } from '../../components/Dialog/Dialog'
import { useVoidSurveyResponse } from '../../hooks/useSurveyAdmin'
import { formatDeadline } from '../../lib/format'
import type { OfficerSurveyResponse, SurveyQuestion } from '../../types/surveys'
import styles from './OfficerSurveyManage.module.css'

const maxReasonLength = 500

interface ResponsesTableProps {
  surveyId: string
  questions: SurveyQuestion[]
  responses: OfficerSurveyResponse[]
}

export function ResponsesTable({ surveyId, questions, responses }: ResponsesTableProps) {
  const [onlyInServer, setOnlyInServer] = useState(false)
  const [voiding, setVoiding] = useState<OfficerSurveyResponse | null>(null)
  const [reason, setReason] = useState('')
  const voidResponse = useVoidSurveyResponse(surveyId)

  const voidedCount = responses.filter((response) => response.voidedAt !== null).length
  const shown = onlyInServer ? responses.filter((response) => response.inDiscordServer) : responses

  const closeDialog = () => {
    setVoiding(null)
    setReason('')
    voidResponse.reset()
  }

  const confirmVoid = () => {
    if (!voiding) return
    voidResponse.mutate({ responseId: voiding.id, voided: true, reason: reason.trim() }, { onSuccess: closeDialog })
  }

  if (responses.length === 0) {
    return <p className={styles.notice}>No responses yet.</p>
  }

  return (
    <div>
      <p className={styles.summary}>
        {responses.length} {responses.length === 1 ? 'response' : 'responses'}, {voidedCount} voided
      </p>
      <label className={styles.filter}>
        <input type="checkbox" checked={onlyInServer} onChange={(event) => setOnlyInServer(event.target.checked)} />
        Only people in our Discord server
      </label>
      <div className={styles.tableWrap}>
        <table className={styles.table}>
          <thead>
            <tr>
              <th scope="col">Discord name</th>
              <th scope="col">Guild name</th>
              <th scope="col">In server</th>
              <th scope="col">Submitted</th>
              <th scope="col">Answers</th>
              <th scope="col">Status</th>
              <th scope="col">
                <span className={styles.visuallyHidden}>Actions</span>
              </th>
            </tr>
          </thead>
          <tbody>
            {shown.map((response) => {
              const voided = response.voidedAt !== null
              return (
                <tr key={response.id} className={voided ? styles.voidedRow : undefined}>
                  <td>{response.discordUsername}</td>
                  <td>{response.guildName || 'Not given'}</td>
                  <td>{response.inDiscordServer ? 'Yes' : 'No'}</td>
                  <td>{formatDeadline(response.submittedAt)}</td>
                  <td>
                    <ul className={styles.answers}>
                      {questions.map((question) => {
                        const answer = response.answers.find((a) => a.questionId === question.id)
                        return answer ? (
                          <li key={question.id}>
                            <strong>{question.prompt}</strong>: {answer.values.join(', ')}
                          </li>
                        ) : null
                      })}
                    </ul>
                  </td>
                  <td>
                    {voided
                      ? `Voided by ${response.voidedBy ?? 'an officer'}: ${response.voidReason || 'no reason given'}`
                      : 'Counted'}
                  </td>
                  <td>
                    {voided ? (
                      <button
                        type="button"
                        className={styles.smallButton}
                        aria-label={`Restore ${response.discordUsername}`}
                        disabled={voidResponse.isPending}
                        onClick={() => voidResponse.mutate({ responseId: response.id, voided: false })}
                      >
                        Restore
                      </button>
                    ) : (
                      <button
                        type="button"
                        className={styles.smallButton}
                        aria-label={`Void ${response.discordUsername}`}
                        onClick={() => setVoiding(response)}
                      >
                        Void
                      </button>
                    )}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
      {voidResponse.isError && !voiding && (
        <p role="alert" className={styles.error}>
          {voidResponse.error.message}
        </p>
      )}
      {voiding && (
        <Dialog title={`Void ${voiding.discordUsername}'s response`} onClose={closeDialog}>
          <label className={styles.field}>
            Reason (optional)
            <textarea
              rows={3}
              maxLength={maxReasonLength}
              value={reason}
              onChange={(event) => setReason(event.target.value)}
            />
          </label>
          {voidResponse.isError && (
            <p role="alert" className={styles.error}>
              {voidResponse.error.message}
            </p>
          )}
          <div className={styles.dialogActions}>
            <button type="button" className={styles.smallButton} onClick={closeDialog}>
              Cancel
            </button>
            <button type="button" className={styles.button} disabled={voidResponse.isPending} onClick={confirmVoid}>
              Void response
            </button>
          </div>
        </Dialog>
      )}
    </div>
  )
}
