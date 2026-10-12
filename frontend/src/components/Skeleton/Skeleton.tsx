import type { ReactNode } from 'react'
import styles from './Skeleton.module.css'

type SkeletonProps = {
  height: number | string
  className?: string
}

export function Skeleton({ height, className }: SkeletonProps) {
  return (
    <div
      data-skeleton
      aria-hidden="true"
      className={className ? `${styles.skeleton} ${className}` : styles.skeleton}
      style={{ height }}
    />
  )
}

type LoadingPlaceholderProps = {
  label: string
  children?: ReactNode
}

export function LoadingPlaceholder({ label, children }: LoadingPlaceholderProps) {
  return (
    <div role="status">
      <span className={styles.label}>{label}</span>
      {children}
    </div>
  )
}
