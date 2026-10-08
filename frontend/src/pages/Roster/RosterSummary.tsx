import styles from './Roster.module.css'
import type { RosterSummary as Summary } from './summarizeRoster'

function count(value: number, singular: string, plural: string) {
  return `${value} ${value === 1 ? singular : plural}`
}

export function RosterSummary({ summary }: { summary: Summary }) {
  return (
    <section className={styles.summary} aria-label="Roster summary">
      <p className={styles.summaryTotal}>{count(summary.total, 'character', 'characters')}</p>
      <p className={styles.summaryLine}>
        {count(summary.tanks, 'Tank', 'Tanks')} | {count(summary.healers, 'Healer', 'Healers')} |{' '}
        {summary.dps} DPS
      </p>
      <p className={styles.summaryLine}>
        Main {summary.mains} | {count(summary.alts, 'Alt', 'Alts')}
        {summary.averageLevel !== null && <> · Avg level {summary.averageLevel}</>}
      </p>
    </section>
  )
}
