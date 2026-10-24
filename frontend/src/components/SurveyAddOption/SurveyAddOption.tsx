import { useState, type FormEvent } from 'react'
import type { SurveyQuestion } from '../../types/surveys'
import styles from './SurveyAddOption.module.css'

interface SurveyAddOptionProps {
  questions: SurveyQuestion[]
  pending: boolean
  error: string | null
  onSubmit: (questionId: string, option: string) => Promise<void>
}

export function SurveyAddOption({ questions, pending, error, onSubmit }: SurveyAddOptionProps) {
  const [questionId, setQuestionId] = useState(questions[0]?.id ?? '')
  const [option, setOption] = useState('')

  const handleSubmit = async (event: FormEvent) => {
    event.preventDefault()
    try {
      await onSubmit(questionId, option.trim())
      setOption('')
    } catch {
      // The rejection reaches the officer through the error prop; the typed option stays for a retry.
    }
  }

  return (
    <form className={styles.form} onSubmit={handleSubmit} noValidate>
      <p className={styles.notice}>
        Questions are locked once people have responded, but you can still add a missing option. Existing answers are
        not affected.
      </p>
      <div className={styles.row}>
        <label className={styles.field}>
          Question
          <select value={questionId} onChange={(event) => setQuestionId(event.target.value)}>
            {questions.map((question) => (
              <option key={question.id} value={question.id}>
                {question.prompt}
              </option>
            ))}
          </select>
        </label>
        <label className={styles.field}>
          New option
          <input value={option} onChange={(event) => setOption(event.target.value)} />
        </label>
        <button type="submit" className={styles.button} disabled={pending || option.trim() === ''}>
          {pending ? 'Adding…' : 'Add option'}
        </button>
      </div>
      {error && (
        <p role="alert" className={styles.error}>
          {error}
        </p>
      )}
    </form>
  )
}
