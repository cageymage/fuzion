import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { OfficerOnly } from '../../components/OfficerOnly/OfficerOnly'
import { SurveyEditorForm } from '../../components/SurveyEditorForm/SurveyEditorForm'
import {
  emptySurveyDraft,
  fromLocalInput,
  toQuestionInputs,
  type SurveyDraft,
} from '../../components/SurveyEditorForm/surveyDraft'
import { useCreateSurvey } from '../../hooks/useSurveyAdmin'
import styles from './OfficerSurveyNew.module.css'

function NewSurveyForm() {
  const navigate = useNavigate()
  const create = useCreateSurvey()
  const [initial] = useState(emptySurveyDraft)
  const [missingDeadline, setMissingDeadline] = useState(false)

  const handleSubmit = (draft: SurveyDraft) => {
    if (draft.closesAt === '') {
      setMissingDeadline(true)
      return
    }
    setMissingDeadline(false)
    create.mutate(
      {
        title: draft.title,
        description: draft.description,
        closesAt: fromLocalInput(draft.closesAt),
        questions: toQuestionInputs(draft.questions),
      },
      { onSuccess: (survey) => navigate(`/officer/surveys/${survey.id}`) },
    )
  }

  const error = missingDeadline ? 'Pick when the survey closes' : (create.error?.message ?? null)

  return (
    <section className={styles.page}>
      <h1 className={styles.title}>New Survey</h1>
      <Link to="/officer/surveys" className={styles.backLink}>
        Back to Manage Surveys
      </Link>
      <SurveyEditorForm
        initial={initial}
        questionsLocked={false}
        submitLabel="Create survey"
        pending={create.isPending}
        error={error}
        onSubmit={handleSubmit}
      />
    </section>
  )
}

export function OfficerSurveyNew() {
  return (
    <OfficerOnly>
      <NewSurveyForm />
    </OfficerOnly>
  )
}
