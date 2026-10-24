import { useState, type FormEvent } from 'react'
import { QuestionBuilder } from './QuestionBuilder'
import styles from './SurveyEditorForm.module.css'
import type { SurveyDraft } from './surveyDraft'

interface SurveyEditorFormProps {
  initial: SurveyDraft
  questionsLocked: boolean
  submitLabel: string
  pending: boolean
  error: string | null
  onSubmit: (draft: SurveyDraft) => void
}

export function SurveyEditorForm({
  initial,
  questionsLocked,
  submitLabel,
  pending,
  error,
  onSubmit,
}: SurveyEditorFormProps) {
  const [draft, setDraft] = useState(initial)

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault()
    onSubmit(draft)
  }

  return (
    <form className={styles.form} onSubmit={handleSubmit} noValidate>
      <label className={styles.field}>
        Title
        <input value={draft.title} onChange={(event) => setDraft({ ...draft, title: event.target.value })} />
      </label>
      <label className={styles.field}>
        Description
        <textarea
          rows={3}
          value={draft.description}
          onChange={(event) => setDraft({ ...draft, description: event.target.value })}
        />
      </label>
      <label className={styles.field}>
        Closes at
        <input
          type="datetime-local"
          value={draft.closesAt}
          onChange={(event) => setDraft({ ...draft, closesAt: event.target.value })}
        />
      </label>
      <h2 className={styles.heading}>Questions</h2>
      {questionsLocked ? (
        <>
          <p className={styles.notice}>Questions cannot change once people have responded.</p>
          <ol className={styles.lockedList}>
            {initial.questions.map((question) => (
              <li key={question.key}>
                <strong>{question.prompt}</strong> ({question.type === 'single' ? 'single choice' : 'multiple choice'}
                {question.required ? ', required' : ''}): {question.customOptions.split('\n').join(', ')}
              </li>
            ))}
          </ol>
        </>
      ) : (
        <QuestionBuilder questions={draft.questions} onChange={(questions) => setDraft({ ...draft, questions })} />
      )}
      {error && (
        <p role="alert" className={styles.error}>
          {error}
        </p>
      )}
      <button type="submit" className={styles.button} disabled={pending}>
        {pending ? 'Saving…' : submitLabel}
      </button>
    </form>
  )
}
