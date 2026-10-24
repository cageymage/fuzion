import type { SurveyResults as SurveyResultsData } from '../../types/surveys'
import styles from './Survey.module.css'
import { useSurveyResults } from './useSurvey'

interface SurveyResultsProps {
  surveyId: string
}

function percent(count: number, turnout: number): number {
  return turnout === 0 ? 0 : Math.round((count / turnout) * 100)
}

export function SurveyResults({ surveyId }: SurveyResultsProps) {
  const results = useSurveyResults(surveyId)

  if (results.isPending) {
    return <p className={styles.notice}>Loading results…</p>
  }
  if (results.isError) {
    return (
      <p role="alert" className={styles.error}>
        The results could not be loaded.
      </p>
    )
  }

  return <SurveyResultsView results={results.data} />
}

export function SurveyResultsView({ results }: { results: SurveyResultsData }) {
  const { turnout, questions } = results
  return (
    <section aria-label="Results">
      <p className={styles.turnout}>
        {turnout} {turnout === 1 ? 'response' : 'responses'}
      </p>
      {questions.map((question) => (
        <section key={question.id} aria-labelledby={`results-${question.id}`} className={styles.result}>
          <h2 id={`results-${question.id}`} className={styles.questionTitle}>
            {question.prompt}
          </h2>
          <ul className={styles.bars}>
            {question.options.map((option) => (
              <li key={option.label} className={styles.bar}>
                <span className={styles.barLabel}>{option.label}</span>
                <span className={styles.track} aria-hidden="true">
                  <span className={styles.fill} style={{ width: `${percent(option.count, turnout)}%` }} />
                </span>
                <span className={styles.barValue}>
                  {option.count} ({percent(option.count, turnout)}%)
                </span>
              </li>
            ))}
          </ul>
        </section>
      ))}
    </section>
  )
}
