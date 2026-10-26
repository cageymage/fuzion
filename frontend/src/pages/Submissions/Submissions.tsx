import { useMutation, useQuery } from '@tanstack/react-query'
import { useEffect, useRef, useState, type DragEvent, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import {
  ALLOWED_TYPES,
  MAX_BATCH_BYTES,
  MAX_BATCH_FILES,
  MAX_DESCRIPTION,
  MAX_FILE_BYTES,
  fetchSubmissionStatus,
  submitImages,
} from '../../api/submissions'
import { useCurrentUser } from '../../hooks/useCurrentUser'
import { useLoginUrl } from '../../hooks/useLoginUrl'
import type { SubmissionFileResult } from '../../types/submissions'
import styles from './Submissions.module.css'

interface Selected {
  key: string
  file: File
  previewUrl: string
  description: string
}

const mib = (bytes: number) => `${Math.round(bytes / (1024 * 1024))} MiB`

// Checks a new file against the server's limits, so the member hears no before uploading.
function problemWith(file: File): string | null {
  if (!ALLOWED_TYPES.includes(file.type)) return `${file.name}: must be a PNG, JPEG, WebP or GIF image`
  if (file.size > MAX_FILE_BYTES) return `${file.name}: must be at most ${mib(MAX_FILE_BYTES)}`
  return null
}

function SubmissionForm() {
  const inputRef = useRef<HTMLInputElement>(null)
  const nextKey = useRef(0)
  const [selected, setSelected] = useState<Selected[]>([])
  const [problems, setProblems] = useState<string[]>([])
  const [attested, setAttested] = useState(false)
  const [dragging, setDragging] = useState(false)
  const [results, setResults] = useState<SubmissionFileResult[]>([])
  const submit = useMutation({
    mutationFn: (items: Selected[]) =>
      submitImages(items.map(({ file, description }) => ({ file, description: description.trim() }))),
    onSuccess: (fileResults, items) => {
      setResults(fileResults)
      // Keep only the files that failed, so the member can fix and resend just those.
      const kept = items.filter((_, i) => !fileResults[i]?.submission)
      for (const item of items) {
        if (!kept.includes(item)) URL.revokeObjectURL(item.previewUrl)
      }
      setSelected(kept)
    },
  })

  // Frees the previews still selected when the page goes away.
  const selectedRef = useRef(selected)
  selectedRef.current = selected
  useEffect(() => () => selectedRef.current.forEach(({ previewUrl }) => URL.revokeObjectURL(previewUrl)), [])

  function addFiles(files: FileList | File[]) {
    const newProblems: string[] = []
    const accepted: Selected[] = []
    for (const file of Array.from(files)) {
      const problem = problemWith(file)
      if (problem) {
        newProblems.push(problem)
        continue
      }
      if (selected.length + accepted.length >= MAX_BATCH_FILES) {
        newProblems.push(`You can add at most ${MAX_BATCH_FILES} images per submission.`)
        break
      }
      accepted.push({ key: String(nextKey.current++), file, previewUrl: URL.createObjectURL(file), description: '' })
    }
    setProblems(newProblems)
    setResults([])
    setSelected((current) => [...current, ...accepted])
  }

  function remove(key: string) {
    setSelected((current) => {
      const item = current.find((candidate) => candidate.key === key)
      if (item) URL.revokeObjectURL(item.previewUrl)
      return current.filter((candidate) => candidate.key !== key)
    })
  }

  function describe(key: string, description: string) {
    setSelected((current) => current.map((item) => (item.key === key ? { ...item, description } : item)))
  }

  function onDrop(event: DragEvent) {
    event.preventDefault()
    setDragging(false)
    addFiles(event.dataTransfer.files)
  }

  const totalBytes = selected.reduce((sum, { file }) => sum + file.size, 0)
  const tooLarge = totalBytes > MAX_BATCH_BYTES

  function onSubmit(event: FormEvent) {
    event.preventDefault()
    if (selected.length === 0 || !attested || tooLarge) return
    submit.mutate(selected)
  }

  return (
    <form className={styles.form} onSubmit={onSubmit} aria-label="Submit art or screenshots">
      <div
        className={`${styles.dropzone} ${dragging ? styles.dragging : ''}`}
        onDragOver={(event) => {
          event.preventDefault()
          setDragging(true)
        }}
        onDragLeave={() => setDragging(false)}
        onDrop={onDrop}
        data-testid="dropzone"
      >
        <p>Drag images here, or</p>
        <button type="button" className={styles.button} onClick={() => inputRef.current?.click()}>
          Choose images
        </button>
        <input
          ref={inputRef}
          className={styles.fileInput}
          type="file"
          accept={ALLOWED_TYPES.join(',')}
          multiple
          aria-label="Images to submit"
          tabIndex={-1}
          onChange={(event) => {
            if (event.target.files) addFiles(event.target.files)
            event.target.value = ''
          }}
        />
        <p className={styles.hint}>
          PNG, JPEG, WebP or GIF. Up to {MAX_BATCH_FILES} images and {mib(MAX_BATCH_BYTES)} at a time,{' '}
          {mib(MAX_FILE_BYTES)} each.
        </p>
      </div>

      {problems.length > 0 && (
        <ul role="alert" className={styles.problems}>
          {problems.map((problem) => (
            <li key={problem}>{problem}</li>
          ))}
        </ul>
      )}

      {selected.length > 0 && (
        <ul className={styles.selected} aria-label="Selected images">
          {selected.map((item) => (
            <li key={item.key} className={styles.item}>
              <img className={styles.preview} src={item.previewUrl} alt={`Preview of ${item.file.name}`} />
              <div className={styles.itemBody}>
                <span className={styles.filename}>{item.file.name}</span>
                <textarea
                  className={styles.description}
                  maxLength={MAX_DESCRIPTION}
                  value={item.description}
                  onChange={(event) => describe(item.key, event.target.value)}
                  placeholder="Optional description"
                  aria-label={`Description for ${item.file.name}`}
                />
              </div>
              <button
                type="button"
                className={styles.button}
                onClick={() => remove(item.key)}
                aria-label={`Remove ${item.file.name}`}
              >
                Remove
              </button>
            </li>
          ))}
        </ul>
      )}
      {tooLarge && (
        <p role="alert" className={styles.error}>
          These images add up to more than {mib(MAX_BATCH_BYTES)}. Remove some and submit them separately.
        </p>
      )}

      <section aria-labelledby="license-heading" className={styles.license}>
        <h2 id="license-heading" className={styles.licenseTitle}>
          License
        </h2>
        <p>
          By submitting, you grant the Fuzion guild an unrestricted license to use these images on
          the guild site and in any related social media posts. You keep your copyright.
        </p>
        <label className={styles.attestation}>
          <input type="checkbox" checked={attested} onChange={(event) => setAttested(event.target.checked)} />
          I created these images or bought the rights to them, and I grant the license above. Game
          screenshots are fine.
        </label>
      </section>

      <button
        type="submit"
        className={styles.submit}
        disabled={selected.length === 0 || !attested || tooLarge || submit.isPending}
      >
        {submit.isPending ? 'Submitting…' : 'Submit'}
      </button>

      {submit.isError && (
        <p role="alert" className={styles.error}>
          {submit.error.message}
        </p>
      )}
      {results.length > 0 && (
        <ul className={styles.results} aria-label="Submission results">
          {results.map((result, i) => (
            <li key={`${result.filename}-${i}`} className={result.submission ? styles.success : styles.error}>
              {result.submission ? `${result.filename}: submitted` : `${result.filename}: ${result.error}`}
            </li>
          ))}
        </ul>
      )}
    </form>
  )
}

export function Submissions() {
  const currentUser = useCurrentUser()
  const loginUrl = useLoginUrl()
  const status = useQuery({ queryKey: ['submissions', 'status'], queryFn: fetchSubmissionStatus })

  let body
  if (currentUser.isPending || status.isPending) {
    body = <p className={styles.notice}>Loading…</p>
  } else if (status.isError) {
    body = (
      <p role="alert" className={styles.error}>
        Submissions could not be loaded.
      </p>
    )
  } else if (!status.data.open) {
    body = <p className={styles.notice}>Submissions are temporarily closed. Please try again later.</p>
  } else if (!currentUser.data) {
    body = (
      <p>
        <a href={loginUrl}>Log in with Discord</a> to submit art or screenshots.
      </p>
    )
  } else if (!currentUser.data.isGuildMember) {
    body = <p className={styles.notice}>Only members of the guild&apos;s Discord server can submit images.</p>
  } else {
    body = <SubmissionForm />
  }

  return (
    <article className={styles.page}>
      <h1 className={styles.title}>Submit art or screenshots</h1>
      {currentUser.data?.isOfficer && (
        <Link to="/officer/submissions" className={styles.manageLink}>
          Review submissions
        </Link>
      )}
      <p className={styles.intro}>
        Share your art or screenshots with the guild. Officers may use them for news posts, the site
        and the guild&apos;s social media.
      </p>
      {body}
    </article>
  )
}
