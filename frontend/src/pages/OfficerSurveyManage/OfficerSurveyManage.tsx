import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { surveyExportUrl } from '../../api/surveys'
import { Dialog } from '../../components/Dialog/Dialog'
import { OfficerOnly } from '../../components/OfficerOnly/OfficerOnly'
import { SurveyAddOption } from '../../components/SurveyAddOption/SurveyAddOption'
import { SurveyEditorForm } from '../../components/SurveyEditorForm/SurveyEditorForm'
import {
  draftFromSurvey,
  fromLocalInput,
  toQuestionInputs,
  type SurveyDraft,
} from '../../components/SurveyEditorForm/surveyDraft'
import { useToast } from '../../components/Toast/Toast'
import { useAddSurveyOption, useSurveyResponses, useUpdateSurvey } from '../../hooks/useSurveyAdmin'
import { useSurvey } from '../Survey/useSurvey'
import { LiveResults } from './LiveResults'
import styles from './OfficerSurveyManage.module.css'
import { ResponsesTable } from './ResponsesTable'

function ManageSurvey() {
  const { id = '' } = useParams()
  const survey = useSurvey(id)
  const responses = useSurveyResponses(id, { refresh: survey.data?.status === 'open' })
  const update = useUpdateSurvey(id)
  const addOption = useAddSurveyOption(id)
  const { showToast } = useToast()
  const [confirmingClose, setConfirmingClose] = useState(false)
  const [missingDeadline, setMissingDeadline] = useState(false)

  if (survey.isPending) {
    return (
      <section className={styles.page}>
        <p className={styles.notice}>Loading survey…</p>
      </section>
    )
  }
  if (survey.isError) {
    return (
      <section className={styles.page}>
        <h1 className={styles.title}>Manage Survey</h1>
        <p role="alert" className={styles.error}>
          The survey could not be loaded.
        </p>
        <Link to="/officer/surveys" className={styles.link}>
          Back to Manage Surveys
        </Link>
      </section>
    )
  }

  const { data } = survey
  const questionsLocked = !responses.isSuccess || responses.data.length > 0

  const handleSave = (draft: SurveyDraft) => {
    if (draft.closesAt === '') {
      setMissingDeadline(true)
      return
    }
    setMissingDeadline(false)
    update.mutate(
      {
        title: draft.title,
        description: draft.description,
        closesAt: fromLocalInput(draft.closesAt),
        ...(questionsLocked ? {} : { questions: toQuestionInputs(draft.questions) }),
      },
      { onSuccess: () => showToast('Survey saved') },
    )
  }

  const handleAddOption = async (questionId: string, option: string) => {
    await addOption.mutateAsync({ questionId, option })
    showToast('Option added')
  }

  const closeNow = () => {
    update.mutate(
      { closesAt: new Date().toISOString() },
      {
        onSuccess: () => showToast('Survey closed'),
        onSettled: () => setConfirmingClose(false),
      },
    )
  }

  const saveError = missingDeadline ? 'Pick when the survey closes' : (update.error?.message ?? null)

  return (
    <section className={styles.page}>
      <h1 className={styles.title}>{data.title}</h1>
      <div className={styles.links}>
        <Link to="/officer/surveys" className={styles.link}>
          Back to Manage Surveys
        </Link>
        <Link to={`/surveys/${data.id}`} className={styles.link}>
          View public page
        </Link>
        <a href={surveyExportUrl(data.id)} className={styles.link} download>
          Download CSV
        </a>
        {data.status === 'open' && (
          <button type="button" className={styles.smallButton} onClick={() => setConfirmingClose(true)}>
            Close now
          </button>
        )}
      </div>

      {data.status === 'open' && <LiveResults surveyId={data.id} />}

      <h2 className={styles.heading}>Settings</h2>
      {/* Keyed on the deadline so closing the survey resets the form; a stale future deadline would reopen it on the next save. */}
      <SurveyEditorForm
        key={`${data.id}-${data.closesAt}`}
        initial={draftFromSurvey(data)}
        questionsLocked={questionsLocked}
        submitLabel="Save changes"
        pending={update.isPending && !confirmingClose}
        error={saveError}
        onSubmit={handleSave}
      />

      {questionsLocked && data.status === 'open' && (
        <>
          <h2 className={styles.heading}>Add an option</h2>
          <SurveyAddOption
            questions={data.questions}
            pending={addOption.isPending}
            error={addOption.error?.message ?? null}
            onSubmit={handleAddOption}
          />
        </>
      )}

      <h2 className={styles.heading}>Responses</h2>
      {responses.isPending && <p className={styles.notice}>Loading responses…</p>}
      {responses.isError && (
        <p role="alert" className={styles.error}>
          The responses could not be loaded.
        </p>
      )}
      {responses.isSuccess && (
        <ResponsesTable surveyId={data.id} questions={data.questions} responses={responses.data} />
      )}

      {confirmingClose && (
        <Dialog title="Close this survey now?" onClose={() => setConfirmingClose(false)}>
          <p>Results become public straight away and nobody can respond or edit afterwards.</p>
          <div className={styles.dialogActions}>
            <button type="button" className={styles.smallButton} onClick={() => setConfirmingClose(false)}>
              Cancel
            </button>
            <button type="button" className={styles.button} disabled={update.isPending} onClick={closeNow}>
              Close survey
            </button>
          </div>
        </Dialog>
      )}
    </section>
  )
}

export function OfficerSurveyManage() {
  return (
    <OfficerOnly>
      <ManageSurvey />
    </OfficerOnly>
  )
}
