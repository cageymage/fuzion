import type { Survey, SurveyOptionPreset, SurveyQuestion, SurveyQuestionInput, SurveyQuestionType } from '../../types/surveys'

export type OptionSource = SurveyOptionPreset | 'custom'

export interface QuestionDraft {
  key: string
  prompt: string
  type: SurveyQuestionType
  required: boolean
  source: OptionSource
  customOptions: string
}

export interface SurveyDraft {
  title: string
  description: string
  closesAt: string
  questions: QuestionDraft[]
}

export const optionSources: { value: OptionSource; label: string }[] = [
  { value: 'wow-classes', label: 'Classic classes' },
  { value: 'wow-roles', label: 'Roles' },
  { value: 'weekdays', label: 'Weekdays' },
  { value: 'custom', label: 'Custom options' },
]

let nextKey = 0

export function newQuestionDraft(): QuestionDraft {
  nextKey += 1
  return { key: `draft-${nextKey}`, prompt: '', type: 'single', required: true, source: 'custom', customOptions: '' }
}

export function emptySurveyDraft(): SurveyDraft {
  return { title: '', description: '', closesAt: '', questions: [newQuestionDraft()] }
}

export function draftFromSurvey(survey: Survey): SurveyDraft {
  return {
    title: survey.title,
    description: survey.description,
    closesAt: toLocalInput(survey.closesAt),
    questions: survey.questions.map(draftFromQuestion),
  }
}

function draftFromQuestion(question: SurveyQuestion): QuestionDraft {
  nextKey += 1
  return {
    key: `draft-${nextKey}`,
    prompt: question.prompt,
    type: question.type,
    required: question.required,
    source: 'custom',
    customOptions: question.options.join('\n'),
  }
}

export function toQuestionInputs(questions: QuestionDraft[]): SurveyQuestionInput[] {
  return questions.map((question) => {
    const base = { prompt: question.prompt.trim(), type: question.type, required: question.required }
    if (question.source !== 'custom') {
      return { ...base, preset: question.source }
    }
    return { ...base, options: question.customOptions.split('\n').map((line) => line.trim()).filter(Boolean) }
  })
}

// <input type="datetime-local"> works in local time without a zone, so it is converted at the edges.
export function toLocalInput(isoDate: string): string {
  const date = new Date(isoDate)
  const pad = (value: number) => String(value).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

export function fromLocalInput(localValue: string): string {
  return new Date(localValue).toISOString()
}
