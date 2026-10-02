import { useMutation } from '@tanstack/react-query'
import { Turnstile, type TurnstileInstance } from '@marsidev/react-turnstile'
import { useRef, useState, type FormEvent } from 'react'
import { submitApplication } from '../../api/applications'
import { ApiError } from '../../api/client'
import { wowClasses } from '../../lib/wowClasses'
import type { ApplicationRole, SubmitApplicationRequest } from '../../types/applications'
import styles from './Applications.module.css'

const maxNotesLength = 2000
const turnstileSiteKey = import.meta.env.VITE_TURNSTILE_SITE_KEY
const genericError = 'Something went wrong, try again or ping an officer on Discord'

const roles: { value: ApplicationRole; label: string }[] = [
  { value: 'tank', label: 'Tank' },
  { value: 'healer', label: 'Healer' },
  { value: 'dps', label: 'DPS' },
]

const emptyForm: SubmitApplicationRequest = {
  applicantName: '',
  characterName: '',
  class: '',
  role: '',
  availability: '',
  discordHandle: '',
  notes: '',
  turnstileToken: '',
}

const requiredFields: { key: Exclude<keyof SubmitApplicationRequest, 'turnstileToken'>; label: string }[] = [
  { key: 'applicantName', label: 'Your name' },
  { key: 'characterName', label: 'Character name' },
  { key: 'class', label: 'Class' },
  { key: 'role', label: 'Role' },
  { key: 'availability', label: 'Availability' },
  { key: 'discordHandle', label: 'Discord handle' },
]

export function Applications() {
  const [form, setForm] = useState(emptyForm)
  const [validationError, setValidationError] = useState<string | null>(null)
  const turnstile = useRef<TurnstileInstance>(null)
  const submission = useMutation({ mutationFn: submitApplication })

  const update = (key: keyof SubmitApplicationRequest, value: string) =>
    setForm((current) => ({ ...current, [key]: value }))

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault()
    const missing = requiredFields.filter(({ key }) => form[key].trim() === '')
    if (missing.length > 0) {
      setValidationError(`Please fill in: ${missing.map(({ label }) => label).join(', ')}`)
      return
    }
    setValidationError(null)
    submission.mutate(form, {
      onError: () => {
        // Tokens are single-use, so a failed submit needs a fresh challenge.
        update('turnstileToken', '')
        turnstile.current?.reset()
      },
    })
  }

  if (submission.isSuccess) {
    return (
      <section className={styles.page}>
        <h1 className={styles.title}>Applications</h1>
        <p className={styles.thanks}>
          Thank you for applying! An officer will review your application and reach out on Discord.
        </p>
      </section>
    )
  }

  let serverError: string | null = null
  if (submission.error instanceof ApiError && submission.error.status === 400) {
    serverError = submission.error.message
  } else if (submission.isError) {
    serverError = genericError
  }
  const errorMessage = validationError ?? serverError

  return (
    <section className={styles.page}>
      <h1 className={styles.title}>Applications</h1>
      <form className={styles.form} onSubmit={handleSubmit} noValidate>
        {errorMessage && (
          <p role="alert" className={styles.error}>
            {errorMessage}
          </p>
        )}
        <label className={styles.field}>
          Your name *
          <input
            value={form.applicantName}
            onChange={(event) => update('applicantName', event.target.value)}
          />
        </label>
        <label className={styles.field}>
          Character name *
          <input
            value={form.characterName}
            onChange={(event) => update('characterName', event.target.value)}
          />
        </label>
        <label className={styles.field}>
          Class *
          <select value={form.class} onChange={(event) => update('class', event.target.value)}>
            <option value="">Select a class</option>
            {wowClasses.map((wowClass) => (
              <option key={wowClass} value={wowClass}>
                {wowClass}
              </option>
            ))}
          </select>
        </label>
        <fieldset className={styles.roles}>
          <legend>Role *</legend>
          {roles.map(({ value, label }) => (
            <label key={value} className={styles.radio}>
              <input
                type="radio"
                name="role"
                value={value}
                checked={form.role === value}
                onChange={() => update('role', value)}
              />
              {label}
            </label>
          ))}
        </fieldset>
        <label className={styles.field}>
          Availability *
          <input
            placeholder="Tue/Thu 8-11 EST"
            value={form.availability}
            onChange={(event) => update('availability', event.target.value)}
          />
        </label>
        <label className={styles.field}>
          Discord handle *
          <input
            value={form.discordHandle}
            onChange={(event) => update('discordHandle', event.target.value)}
          />
        </label>
        <label className={styles.field}>
          Notes (optional)
          <textarea
            rows={5}
            maxLength={maxNotesLength}
            value={form.notes}
            onChange={(event) => update('notes', event.target.value)}
          />
          <span className={styles.counter}>
            {form.notes.length} / {maxNotesLength}
          </span>
        </label>
        <Turnstile
          ref={turnstile}
          siteKey={turnstileSiteKey}
          onSuccess={(token) => update('turnstileToken', token)}
          onExpire={() => update('turnstileToken', '')}
          onError={() => update('turnstileToken', '')}
        />
        <button
          type="submit"
          className={styles.button}
          disabled={submission.isPending || form.turnstileToken === ''}
        >
          {submission.isPending ? 'Sending…' : 'Submit application'}
        </button>
      </form>
    </section>
  )
}
