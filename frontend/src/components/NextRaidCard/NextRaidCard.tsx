import { useEffect, useState } from 'react'
import { formatCountdown } from '../../lib/format'
import type { Raid } from '../../types/raids'
import styles from './NextRaidCard.module.css'

interface NextRaidCardProps {
  raid: Raid
}

export function NextRaidCard({ raid }: NextRaidCardProps) {
  const [now, setNow] = useState(() => Date.now())

  useEffect(() => {
    const ticker = setInterval(() => setNow(Date.now()), 30_000)
    return () => clearInterval(ticker)
  }, [])

  const startsAt = new Date(raid.startsAt).getTime()

  return (
    <div className={`card ${styles.card}`}>
      <div className="eyebrow eyebrow-muted">Next Raid</div>
      <div className={`display ${styles.countdown}`}>{formatCountdown(startsAt - now)}</div>
      <div className={styles.detail}>
        {raid.difficulty} · {raid.instanceName}
      </div>
      <div className={styles.progress}>{raid.progressSummary}</div>
    </div>
  )
}
