import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { deleteNewsPost, editablePostKey } from '../../api/news'
import { useToast } from '../Toast/Toast'
import styles from './DeleteNewsPost.module.css'

interface DeleteNewsPostProps {
  postId: string
  onDeleted: () => void
}

export function DeleteNewsPost({ postId, onDeleted }: DeleteNewsPostProps) {
  const queryClient = useQueryClient()
  const { showToast } = useToast()
  const [confirming, setConfirming] = useState(false)
  const remove = useMutation({
    mutationFn: () => deleteNewsPost(postId),
    onSuccess: () => {
      showToast('Post deleted')
      void queryClient.invalidateQueries({ queryKey: ['officer-news', 'drafts'] })
      void queryClient.invalidateQueries({ queryKey: ['news'] })
      onDeleted()
      queryClient.removeQueries({ queryKey: editablePostKey(postId) })
      queryClient.removeQueries({ queryKey: ['news', 'post', postId] })
    },
  })

  return (
    <>
      {!confirming && (
        <button type="button" className={`${styles.button} ${styles.danger}`} onClick={() => setConfirming(true)}>
          Delete
        </button>
      )}
      {confirming && (
        <span className={styles.confirm}>
          <span>Delete this post? This cannot be undone.</span>
          <button
            type="button"
            className={`${styles.button} ${styles.danger}`}
            disabled={remove.isPending}
            onClick={() => remove.mutate()}
          >
            Confirm delete
          </button>
          <button type="button" className={styles.button} onClick={() => setConfirming(false)}>
            Cancel delete
          </button>
        </span>
      )}
      {remove.isError && (
        <p role="alert" className={styles.error}>
          {remove.error.message}
        </p>
      )}
    </>
  )
}
