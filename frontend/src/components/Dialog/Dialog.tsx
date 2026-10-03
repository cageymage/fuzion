import { useEffect, useId, useRef, type ReactNode } from 'react'
import styles from './Dialog.module.css'

interface DialogProps {
  title: string
  onClose: () => void
  children: ReactNode
}

// Mount it to open it and unmount it to close it, so each opening starts from fresh state.
export function Dialog({ title, onClose, children }: DialogProps) {
  const ref = useRef<HTMLDialogElement>(null)
  const titleId = useId()

  useEffect(() => {
    ref.current?.showModal()
  }, [])

  return (
    <dialog ref={ref} className={styles.dialog} aria-labelledby={titleId} onClose={onClose}>
      <h2 id={titleId} className={styles.title}>
        {title}
      </h2>
      {children}
    </dialog>
  )
}
