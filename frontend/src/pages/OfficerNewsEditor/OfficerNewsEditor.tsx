import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import MDEditor, {
  commands,
  TextAreaCommandOrchestrator,
  type ICommand,
  type TextAreaTextApi,
} from '@uiw/react-md-editor/nohighlight'
import '@uiw/react-md-editor/markdown-editor.css'
import { useEffect, useRef, useState, type ClipboardEvent, type DragEvent } from 'react'
import { Link, useBlocker, useNavigate, useParams } from 'react-router-dom'
import { uploadImage } from '../../api/images'
import { editablePostKey, fetchEditablePost, publishNewsPost } from '../../api/news'
import { LightboxImage } from '../../components/LightboxImage/LightboxImage'
import { OfficerOnly } from '../../components/OfficerOnly/OfficerOnly'
import { useToast } from '../../components/Toast/Toast'
import { newsCategoryMeta } from '../../lib/newsCategory'
import type { EditablePost, NewsCategory, PostFields } from '../../types/news'
import styles from './OfficerNewsEditor.module.css'
import { useAutosave, type SaveStatus } from './useAutosave'

const categories: NewsCategory[] = ['guild-news', 'raid-progress', 'recruitment', 'patch-notes']

interface ImageCommandOptions {
  uploading: boolean
  chooseFile: (api: TextAreaTextApi) => void
  enterUrl: (api: TextAreaTextApi) => void
}

function imageCommand({ uploading, chooseFile, enterUrl }: ImageCommandOptions): ICommand {
  return {
    name: 'image',
    keyCommand: 'group',
    groupName: 'image',
    buttonProps: { 'aria-label': 'Insert image', title: 'Insert image', disabled: uploading },
    icon: <span aria-hidden="true">🖼</span>,
    children: [
      {
        name: 'upload-image',
        keyCommand: 'upload-image',
        buttonProps: { 'aria-label': 'Upload image', title: 'Upload image' },
        icon: <span>Upload</span>,
        execute: (_state, api) => chooseFile(api),
      },
      {
        name: 'image-from-url',
        keyCommand: 'image-from-url',
        buttonProps: { 'aria-label': 'From URL', title: 'From URL' },
        icon: <span>From URL</span>,
        execute: (_state, api) => enterUrl(api),
      },
    ],
  }
}

function firstImageFile(files: FileList | undefined): File | undefined {
  return Array.from(files ?? []).find((file) => file.type.startsWith('image/'))
}

function statusText(status: SaveStatus): string {
  switch (status.state) {
    case 'saving':
      return 'Saving…'
    case 'saved':
      return `Saved ${status.at.toLocaleTimeString([], { hour12: false })}`
    case 'failed':
      return 'Save failed'
    case 'idle':
      return ''
  }
}

interface PostEditorProps {
  post: EditablePost
}

function PostEditor({ post }: PostEditorProps) {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const { showToast } = useToast()
  const [fields, setFields] = useState<PostFields>({
    title: post.title,
    excerpt: post.excerpt,
    category: post.category,
    body: post.body,
    pinned: post.pinned,
  })
  const [publishedAt, setPublishedAt] = useState(post.publishedAt)
  const [confirmingPublish, setConfirmingPublish] = useState(false)
  const { status, dirty, saveNow } = useAutosave(post.id, fields)
  const publish = useMutation({
    mutationFn: async () => {
      if (dirty && !(await saveNow())) {
        throw new Error('Save the post before publishing.')
      }
      return publishNewsPost(post.id)
    },
    onSuccess: (published) => {
      setPublishedAt(published.publishedAt)
      setConfirmingPublish(false)
      queryClient.setQueryData(editablePostKey(post.id), published)
      void queryClient.invalidateQueries({ queryKey: ['officer-news', 'drafts'] })
      void queryClient.invalidateQueries({ queryKey: ['news'] })
      showToast('Post published')
      navigate(`/news/${published.id}`)
    },
  })

  const blocker = useBlocker(dirty)

  const [imageError, setImageError] = useState<string | null>(null)
  const [draggingImage, setDraggingImage] = useState(false)
  const fileInput = useRef<HTMLInputElement>(null)
  const pendingApi = useRef<TextAreaTextApi | null>(null)
  const upload = useMutation({ mutationFn: uploadImage })

  function uploadInto(file: File, api: TextAreaTextApi) {
    setImageError(null)
    upload.mutate(file, {
      onSuccess: (image) => api.replaceSelection(`![](${image.url})`),
      onError: (error) => setImageError(error.message),
    })
  }

  function enterImageUrl(api: TextAreaTextApi) {
    const url = window.prompt('Image URL')?.trim()
    if (!url) return
    if (!url.startsWith('https://')) {
      setImageError('Image URL must start with https://')
      return
    }
    setImageError(null)
    api.replaceSelection(`![](${url})`)
  }

  function uploadPastedImage(event: ClipboardEvent<HTMLTextAreaElement>) {
    const file = firstImageFile(event.clipboardData.files)
    if (!file) return
    event.preventDefault()
    uploadInto(file, new TextAreaCommandOrchestrator(event.currentTarget).textApi)
  }

  function allowImageDrop(event: DragEvent<HTMLDivElement>) {
    if (!event.dataTransfer.types.includes('Files')) return
    event.preventDefault()
    setDraggingImage(true)
  }

  function dropImage(event: DragEvent<HTMLDivElement>) {
    const file = firstImageFile(event.dataTransfer.files)
    if (!file) return
    event.preventDefault()
    setDraggingImage(false)
    const textarea = event.currentTarget.querySelector('textarea')
    if (textarea) uploadInto(file, new TextAreaCommandOrchestrator(textarea).textApi)
  }

  const toolbarCommands = commands
    .getCommands()
    .map((command) =>
      command.keyCommand === 'image'
        ? imageCommand({
            uploading: upload.isPending,
            chooseFile: (api) => {
              pendingApi.current = api
              fileInput.current?.click()
            },
            enterUrl: enterImageUrl,
          })
        : command,
    )

  // beforeunload covers closing the tab; useBlocker covers in-app route changes.
  useEffect(() => {
    if (!dirty) return
    function warn(event: BeforeUnloadEvent) {
      event.preventDefault()
      event.returnValue = ''
    }
    window.addEventListener('beforeunload', warn)
    return () => window.removeEventListener('beforeunload', warn)
  }, [dirty])

  function change(update: Partial<PostFields>) {
    setFields((current) => ({ ...current, ...update }))
  }

  return (
    <section className={styles.page}>
      <h1 className={styles.title}>Edit post</h1>
      <Link to="/officer/news" className={styles.backLink}>
        Back to all posts
      </Link>

      <div className={styles.form}>
        <label className={styles.field}>
          <span>Title</span>
          <input value={fields.title} onChange={(event) => change({ title: event.target.value })} />
        </label>
        <label className={styles.field}>
          <span>Excerpt</span>
          <textarea
            rows={2}
            value={fields.excerpt}
            onChange={(event) => change({ excerpt: event.target.value })}
          />
        </label>
        <label className={styles.field}>
          <span>Category</span>
          <select
            value={fields.category}
            onChange={(event) => change({ category: event.target.value as NewsCategory })}
          >
            {categories.map((category) => (
              <option key={category} value={category}>
                {newsCategoryMeta(category).label}
              </option>
            ))}
          </select>
        </label>
        <label className={styles.checkbox}>
          <input
            type="checkbox"
            checked={fields.pinned}
            onChange={(event) => change({ pinned: event.target.checked })}
          />
          <span>Pinned</span>
        </label>

        <div
          className={`${styles.editor} ${draggingImage ? styles.dropTarget : ''}`}
          data-color-mode="dark"
          data-testid="body-editor"
          onDragOver={allowImageDrop}
          onDragLeave={() => setDraggingImage(false)}
          onDrop={dropImage}
        >
          <MDEditor
            value={fields.body}
            onChange={(value) => change({ body: value ?? '' })}
            commands={toolbarCommands}
            preview="live"
            previewOptions={{ components: { img: LightboxImage } }}
            height={420}
            textareaProps={{
              'aria-label': 'Body',
              onPaste: uploadPastedImage,
            }}
          />
        </div>
        <input
          ref={fileInput}
          type="file"
          accept="image/png,image/jpeg,image/webp,image/gif"
          aria-label="Image file"
          hidden
          onChange={(event) => {
            const file = event.target.files?.[0]
            event.target.value = ''
            if (file && pendingApi.current) uploadInto(file, pendingApi.current)
          }}
        />
        {upload.isPending && <p role="status">Uploading image…</p>}
        {imageError && (
          <p role="alert" className={styles.error}>
            {imageError}
          </p>
        )}

        <div className={styles.actions}>
          <p role="status" className={styles.status}>
            {statusText(status)}
          </p>
          {publishedAt !== null && <span className={styles.published}>Published</span>}
          {publishedAt === null && !confirmingPublish && (
            <button type="button" className={styles.button} onClick={() => setConfirmingPublish(true)}>
              Publish
            </button>
          )}
          {publishedAt === null && confirmingPublish && (
            <span className={styles.confirm}>
              <span>Publish this post to the site?</span>
              <button
                type="button"
                className={styles.button}
                disabled={publish.isPending}
                onClick={() => publish.mutate()}
              >
                Confirm publish
              </button>
              <button type="button" className={styles.button} onClick={() => setConfirmingPublish(false)}>
                Cancel
              </button>
            </span>
          )}
        </div>
        {blocker.state === 'blocked' && (
          <div className={styles.backdrop}>
            <div
              role="alertdialog"
              aria-modal="true"
              aria-label="Unsaved changes"
              className={`card ${styles.leaveDialog}`}
              onKeyDown={(event) => {
                if (event.key === 'Escape') blocker.reset()
              }}
            >
              <h2 className={styles.dialogTitle}>Unsaved changes</h2>
              <p className={styles.dialogText}>You have unsaved changes. Leave this page anyway?</p>
              <div className={styles.dialogActions}>
                <button
                  type="button"
                  className={styles.button}
                  autoFocus
                  onClick={() => blocker.reset()}
                >
                  Stay on this page
                </button>
                <button
                  type="button"
                  className={`${styles.button} ${styles.dangerButton}`}
                  onClick={() => blocker.proceed()}
                >
                  Leave without saving
                </button>
              </div>
            </div>
          </div>
        )}
        {publish.isError && (
          <p role="alert" className={styles.error}>
            {publish.error.message}
          </p>
        )}
      </div>
    </section>
  )
}

function EditorLoader() {
  const { id = '' } = useParams()
  const post = useQuery({
    queryKey: editablePostKey(id),
    queryFn: () => fetchEditablePost(id),
    staleTime: Infinity,
  })

  if (post.isPending) {
    return <p className={styles.notice}>Loading post…</p>
  }
  if (post.isError) {
    return <p className={styles.notice}>Post could not be loaded.</p>
  }
  return <PostEditor key={post.data.id} post={post.data} />
}

export function OfficerNewsEditor() {
  return (
    <OfficerOnly>
      <EditorLoader />
    </OfficerOnly>
  )
}
