import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import styles from './Toast.module.css'

const toastDurationMs = 4000

interface ToastContextValue {
  showToast: (message: string) => void
}

// Without a provider (isolated component tests) showing a toast does nothing.
const ToastContext = createContext<ToastContextValue>({ showToast: () => {} })

export function useToast(): ToastContextValue {
  return useContext(ToastContext)
}

interface ToastProviderProps {
  children: ReactNode
}

export function ToastProvider({ children }: ToastProviderProps) {
  const [toast, setToast] = useState<{ message: string } | null>(null)
  const showToast = useCallback((message: string) => setToast({ message }), [])
  const value = useMemo(() => ({ showToast }), [showToast])

  useEffect(() => {
    if (!toast) return
    const timer = setTimeout(() => setToast(null), toastDurationMs)
    return () => clearTimeout(timer)
  }, [toast])

  return (
    <ToastContext.Provider value={value}>
      {children}
      {toast && (
        <div role="status" className={`card ${styles.toast}`}>
          <span>{toast.message}</span>
          <button
            type="button"
            className={styles.dismiss}
            aria-label="Dismiss notification"
            onClick={() => setToast(null)}
          >
            ✕
          </button>
        </div>
      )}
    </ToastContext.Provider>
  )
}
