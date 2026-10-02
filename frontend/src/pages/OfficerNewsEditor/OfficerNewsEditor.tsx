import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import MDEditor, { commands, type ICommand } from '@uiw/react-md-editor/nohighlight'
import '@uiw/react-md-editor/markdown-editor.css'
import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { fetchEditablePost, publishNewsPost } from '../../api/news'
import { OfficerOnly } from '../../components/OfficerOnly/OfficerOnly'
import { newsCategoryMeta } from '../../lib/newsCategory'
import type { EditablePost, NewsCategory, PostFields } from '../../types/news'
import styles from './OfficerNewsEditor.module.css'
import { useAutosave, type SaveStatus } from './useAutosave'

const categories: NewsCategory[] = ['guild-news', 'raid-progress', 'recruitment', 'patch-notes']

// Interim until image upload (#7): ask for a URL instead of opening a file picker.
const insertImage: ICommand = {
  name: 'image',
  keyCommand: 'image',
  buttonProps: { 'aria-label': 'Insert image', title: 'Insert image' },
  icon: <span aria-hidden="true">🖼</span>,
  execute: (_state, api) => {
    const url = window.prompt('Image URL')?.trim()
    if (url) api.replaceSelection(`![](${url})`)
  },
}

const toolbarCommands = commands
  .getCommands()
  .map((command) => (command.keyCommand === 'image' ? insertImage : command))

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
    onSuccess: async (published) => {
      setPublishedAt(published.publishedAt)
      setConfirmingPublish(false)
      await queryClient.invalidateQueries({ queryKey: ['officer-news', 'drafts'] })
      await queryClient.invalidateQueries({ queryKey: ['news'] })
    },
  })

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

        <div className={styles.editor} data-color-mode="dark">
          <MDEditor
            value={fields.body}
            onChange={(value) => change({ body: value ?? '' })}
            commands={toolbarCommands}
            preview="live"
            height={420}
            textareaProps={{ 'aria-label': 'Body' }}
          />
        </div>

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
    queryKey: ['officer-news', 'post', id],
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
