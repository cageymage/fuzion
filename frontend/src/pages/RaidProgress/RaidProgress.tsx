import { Link } from 'react-router-dom'
import { useCurrentUser } from '../../hooks/useCurrentUser'
import { formatRaidDay } from '../../lib/format'
import type { RaidProgressBoss } from '../../types/raidProgress'
import styles from './RaidProgress.module.css'
import { useRaidProgressData, useRaidTiersData } from './useRaidProgressData'

interface BossListProps {
  bosses: RaidProgressBoss[]
}

function BossList({ bosses }: BossListProps) {
  return (
    <ul className={styles.bossList}>
      {bosses.map((boss) => (
        <li
          key={boss.id}
          className={boss.killedAt ? `${styles.boss} ${styles.bossKilled}` : styles.boss}
        >
          <span
            className={boss.killedAt ? `${styles.marker} ${styles.markerKilled}` : styles.marker}
            aria-hidden="true"
          >
            {boss.killedAt ? '✓' : '—'}
          </span>
          <span className={styles.bossInfo}>
            <span className={styles.bossName}>{boss.name}</span>
            {boss.killedAt && <span className={styles.killedDate}>{formatRaidDay(boss.killedAt)}</span>}
          </span>
        </li>
      ))}
    </ul>
  )
}

interface TierCardProps {
  name: string
  killed: number
  total: number
  bosses: RaidProgressBoss[]
  muted?: boolean
}

function TierCard({ name, killed, total, bosses, muted }: TierCardProps) {
  return (
    <div className={muted ? `card ${styles.card} ${styles.historyCard}` : `card ${styles.card}`}>
      <h3 className={styles.tierName}>{name}</h3>
      <p className={styles.summary}>
        {killed} / {total} {total === 1 ? 'boss' : 'bosses'} defeated
      </p>
      <progress className={styles.bar} value={killed} max={total} />
      <BossList bosses={bosses} />
    </div>
  )
}

export function RaidProgress() {
  const progress = useRaidProgressData()
  const tiers = useRaidTiersData()
  const currentUser = useCurrentUser()
  const pastTiers = (tiers.data ?? []).filter((tier) => !tier.isCurrent)

  return (
    <section className={styles.page}>
      <h1 className={styles.title}>Raid Progress</h1>
      {currentUser.data?.isOfficer && (
        <Link to="/officer/raid-progress" className={styles.manageLink}>
          Manage raid progress
        </Link>
      )}

      {progress.isPending && <p className={styles.noticeText}>Loading raid progress…</p>}
      {progress.isError && <p className={styles.noticeText}>Raid progress could not be loaded.</p>}
      {progress.isSuccess && progress.data.length === 0 && (
        <p className={styles.noticeText}>No tier set up yet.</p>
      )}

      {progress.isSuccess && progress.data.length > 0 && (
        <>
          <h2 className={styles.sectionTitle}>Current Raids</h2>
          <div className={styles.currentList}>
            {progress.data.map((tierProgress) => (
              <TierCard
                key={tierProgress.tier.name}
                name={tierProgress.tier.name}
                killed={tierProgress.killed}
                total={tierProgress.total}
                bosses={tierProgress.bosses}
              />
            ))}
          </div>
        </>
      )}

      {pastTiers.length > 0 && (
        <details className={styles.history}>
          <summary className={styles.historyTitle}>Raid History</summary>
          <div className={styles.historyList}>
            {pastTiers.map((tier) => (
              <TierCard
                key={tier.id}
                name={tier.name}
                killed={tier.bosses.filter((boss) => boss.killedAt).length}
                total={tier.bosses.length}
                bosses={tier.bosses}
                muted
              />
            ))}
          </div>
        </details>
      )}
    </section>
  )
}
