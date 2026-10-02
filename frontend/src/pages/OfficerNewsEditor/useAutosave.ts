import { useMutation } from '@tanstack/react-query'
import { useEffect, useRef, useState } from 'react'
import { saveNewsPost } from '../../api/news'
import type { PostFields } from '../../types/news'

export const autosaveDelayMs = 2000

export type SaveStatus =
  | { state: 'idle' }
  | { state: 'saving' }
  | { state: 'saved'; at: Date }
  | { state: 'failed' }

function sameFields(a: PostFields, b: PostFields): boolean {
  return (
    a.title === b.title &&
    a.excerpt === b.excerpt &&
    a.category === b.category &&
    a.body === b.body &&
    a.pinned === b.pinned
  )
}

export function useAutosave(postId: string, fields: PostFields) {
  const [savedFields, setSavedFields] = useState(fields)
  const [status, setStatus] = useState<SaveStatus>({ state: 'idle' })
  const { mutateAsync } = useMutation({
    mutationFn: (toSave: PostFields) => saveNewsPost(postId, toSave),
  })
  const latestFields = useRef(fields)
  latestFields.current = fields

  const dirty = !sameFields(fields, savedFields)

  async function saveNow(): Promise<boolean> {
    const toSave = latestFields.current
    setStatus({ state: 'saving' })
    try {
      await mutateAsync(toSave)
    } catch {
      setStatus({ state: 'failed' })
      return false
    }
    setSavedFields(toSave)
    setStatus({ state: 'saved', at: new Date() })
    return true
  }

  const saveNowRef = useRef(saveNow)
  saveNowRef.current = saveNow

  useEffect(() => {
    if (!dirty) return
    const timer = setTimeout(() => void saveNowRef.current(), autosaveDelayMs)
    return () => clearTimeout(timer)
  }, [dirty, fields])

  return { status, dirty, saveNow }
}
