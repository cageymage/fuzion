import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { ApiError } from '../../api/client'
import { useToast } from '../../components/Toast/Toast'
import type { Survey, SurveyQuestion } from '../../types/surveys'
import styles from './Survey.module.css'
import { useSubmitSurveyResponse } from './useSurvey'

const maxGuildNameLength = 64
const genericError = 'Something went wrong, try again or ping an officer on Discord'

interface SurveyFormProps {
  survey: Survey
}

export function SurveyForm({ survey }: SurveyFormProps) {
  const [guildName, setGuildName] = useState(survey.myResponse?.guildName ?? '')
  const [selected, setSelected] = useState<Record<string, string[]>>(() =>
    Object.fromEntries((survey.myResponse?.answers ?? []).map((a) => [a.questionId, a.values])),
  )
  const [unanswered, setUnanswered] = useState<string[]>([])
  const submission = useSubmitSurveyResponse(survey.id)
  const { showToast } = useToast()

  const choose = (question: SurveyQuestion, option: string) =>
    setSelected((current) => {
      const chosen = current[question.id] ?? []
      if (question.type === 'single') {
        return { ...current, [question.id]: [option] }
      }
      const next = chosen.includes(option) ? chosen.filter((o) => o !== option) : [...chosen, option]
      return { ...current, [question.id]: next }
    })

  const clear = (question: SurveyQuestion) =>
    setSelected((current) => ({ ...current, [question.id]: [] }))

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault()
    const missing = survey.questions
      .filter((q) => q.required && (selected[q.id] ?? []).length === 0)
      .map((q) => q.prompt)
    if (missing.length > 0) {
      setUnanswered(missing)
      return
    }
    setUnanswered([])
    const answers = survey.questions
      .filter((q) => (selected[q.id] ?? []).length > 0)
      .map((q) => ({ questionId: q.id, values: q.options.filter((o) => selected[q.id].includes(o)) }))
    submission.mutate(
      { guildName: guildName.trim(), answers },
      {
        onSuccess: () => showToast('Response saved'),
        onError: (error) => {
          if (error instanceof ApiError && error.status === 409) {
            showToast('This survey has closed, here are the results')
          }
        },
      },
    )
  }

  let serverError: string | null = null
  if (submission.error instanceof ApiError && submission.error.status === 400) {
    serverError = submission.error.message
  } else if (submission.isError && !(submission.error instanceof ApiError && submission.error.status === 409)) {
    serverError = genericError
  }

  return (
    <form className={styles.form} onSubmit={handleSubmit} noValidate>
      <label className={styles.field}>
        Name you use in the guild (optional)
        <input
          maxLength={maxGuildNameLength}
          value={guildName}
          onChange={(event) => setGuildName(event.target.value)}
        />
      </label>
      {survey.questions.map((question) => (
        <fieldset key={question.id} className={styles.question}>
          <legend>
            {question.prompt}
            {question.required ? ' *' : ''}
          </legend>
          <div className={styles.options}>
            {question.options.map((option) => (
              <label key={option} className={styles.chip}>
                <input
                  type={question.type === 'single' ? 'radio' : 'checkbox'}
                  name={question.id}
                  value={option}
                  checked={(selected[question.id] ?? []).includes(option)}
                  onChange={() => choose(question, option)}
                />
                {option}
              </label>
            ))}
          </div>
          {question.type === 'single' && !question.required && (selected[question.id] ?? []).length > 0 && (
            <button
              type="button"
              className={styles.clear}
              aria-label={`Clear ${question.prompt}`}
              onClick={() => clear(question)}
            >
              Clear
            </button>
          )}
        </fieldset>
      ))}
      <p className={styles.notice}>
        Officers can see your answers and your Discord name. See how we handle your details on the{' '}
        <Link to="/legal">Legal page</Link>.
      </p>
      {unanswered.length > 0 && (
        <div role="alert" className={styles.error}>
          <p className={styles.errorLead}>Please answer:</p>
          <ul className={styles.errorList}>
            {unanswered.map((prompt) => (
              <li key={prompt}>{prompt}</li>
            ))}
          </ul>
        </div>
      )}
      {serverError && (
        <p role="alert" className={styles.error}>
          {serverError}
        </p>
      )}
      <button type="submit" className={styles.button} disabled={submission.isPending}>
        {submission.isPending ? 'Saving…' : survey.myResponse ? 'Update response' : 'Submit response'}
      </button>
    </form>
  )
}
