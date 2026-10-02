import type { ReactNode } from 'react'
import { Navigate } from 'react-router-dom'
import { useCurrentUser } from '../../hooks/useCurrentUser'
import styles from './OfficerOnly.module.css'

interface OfficerOnlyProps {
  children: ReactNode
}

export function OfficerOnly({ children }: OfficerOnlyProps) {
  const currentUser = useCurrentUser()

  if (currentUser.isPending) {
    return <p className={styles.notice}>Checking your access…</p>
  }
  if (currentUser.isError) {
    return <p className={styles.notice}>Your access could not be checked.</p>
  }
  if (!currentUser.data?.isOfficer) {
    return <Navigate to="/" replace />
  }
  return <>{children}</>
}
