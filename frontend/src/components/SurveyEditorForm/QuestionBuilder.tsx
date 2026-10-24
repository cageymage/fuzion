import type { SurveyQuestionType } from '../../types/surveys'
import styles from './SurveyEditorForm.module.css'
import { newQuestionDraft, optionSources, type OptionSource, type QuestionDraft } from './surveyDraft'

interface QuestionBuilderProps {
  questions: QuestionDraft[]
  onChange: (questions: QuestionDraft[]) => void
}

export function QuestionBuilder({ questions, onChange }: QuestionBuilderProps) {
  const update = (index: number, changes: Partial<QuestionDraft>) =>
    onChange(questions.map((question, i) => (i === index ? { ...question, ...changes } : question)))

  const move = (index: number, offset: -1 | 1) => {
    const target = index + offset
    const reordered = [...questions]
    ;[reordered[index], reordered[target]] = [reordered[target], reordered[index]]
    onChange(reordered)
  }

  return (
    <div className={styles.builder}>
      {questions.map((question, index) => {
        const number = index + 1
        return (
          <fieldset key={question.key} className={`card ${styles.questionCard}`}>
            <legend>Question {number}</legend>
            <label className={styles.field}>
              Prompt for question {number}
              <input value={question.prompt} onChange={(event) => update(index, { prompt: event.target.value })} />
            </label>
            <div className={styles.row}>
              <label className={styles.field}>
                Type of question {number}
                <select
                  value={question.type}
                  onChange={(event) => update(index, { type: event.target.value as SurveyQuestionType })}
                >
                  <option value="single">Single choice</option>
                  <option value="multi">Multiple choice</option>
                </select>
              </label>
              <label className={styles.field}>
                Options for question {number}
                <select
                  value={question.source}
                  onChange={(event) => update(index, { source: event.target.value as OptionSource })}
                >
                  {optionSources.map(({ value, label }) => (
                    <option key={value} value={value}>
                      {label}
                    </option>
                  ))}
                </select>
              </label>
              <label className={styles.checkbox}>
                <input
                  type="checkbox"
                  checked={question.required}
                  onChange={(event) => update(index, { required: event.target.checked })}
                />
                Question {number} is required
              </label>
            </div>
            {question.source === 'custom' && (
              <label className={styles.field}>
                Custom options for question {number}, one per line
                <textarea
                  rows={4}
                  value={question.customOptions}
                  onChange={(event) => update(index, { customOptions: event.target.value })}
                />
              </label>
            )}
            <div className={styles.questionActions}>
              <button type="button" className={styles.smallButton} disabled={index === 0} onClick={() => move(index, -1)}>
                Move question {number} up
              </button>
              <button
                type="button"
                className={styles.smallButton}
                disabled={index === questions.length - 1}
                onClick={() => move(index, 1)}
              >
                Move question {number} down
              </button>
              <button
                type="button"
                className={styles.smallButton}
                disabled={questions.length === 1}
                onClick={() => onChange(questions.filter((_, i) => i !== index))}
              >
                Remove question {number}
              </button>
            </div>
          </fieldset>
        )
      })}
      <button type="button" className={styles.smallButton} onClick={() => onChange([...questions, newQuestionDraft()])}>
        Add question
      </button>
    </div>
  )
}
