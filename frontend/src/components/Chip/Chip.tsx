import type { ReactNode } from 'react'
import styles from './Chip.module.css'

export type ChipTone = 'gold' | 'epic' | 'legendary'

interface ChipProps {
  tone: ChipTone
  children: ReactNode
}

export function Chip({ tone, children }: ChipProps) {
  return <span className={`${styles.chip} ${styles[tone]}`}>{children}</span>
}
