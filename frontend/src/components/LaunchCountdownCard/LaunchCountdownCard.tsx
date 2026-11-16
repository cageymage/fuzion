import { useEffect, useState } from 'react'
import { splitCountdown } from '../../lib/format'
import styles from './LaunchCountdownCard.module.css'

export const LAUNCH_AT = '2026-11-04T15:00:00-08:00'

const LIVE_MESSAGE_MS = 48 * 3_600_000

interface LaunchCountdownCardProps {
  launchAt: string
}

const twoDigits = (value: number) => String(value).padStart(2, '0')

export function LaunchCountdownCard({ launchAt }: LaunchCountdownCardProps) {
  const [now, setNow] = useState(() => Date.now())

  useEffect(() => {
    const ticker = setInterval(() => setNow(Date.now()), 1_000)
    return () => clearInterval(ticker)
  }, [])

  const msUntilLaunch = new Date(launchAt).getTime() - now

  if (msUntilLaunch <= -LIVE_MESSAGE_MS) {
    return null
  }

  if (msUntilLaunch <= 0) {
    return (
      <div className={`card ${styles.card}`}>
        <div className="eyebrow eyebrow-muted">Launch Day</div>
        <div className={`display ${styles.live}`}>WoW Forever is live</div>
      </div>
    )
  }

  const { days, hours, minutes, seconds } = splitCountdown(msUntilLaunch)
  const segments = [
    { value: String(days), unit: 'days' },
    { value: twoDigits(hours), unit: 'hours' },
    { value: twoDigits(minutes), unit: 'minutes' },
    { value: twoDigits(seconds), unit: 'seconds' },
  ]

  return (
    <div className={`card ${styles.card}`}>
      <div className="eyebrow eyebrow-muted">WoW Forever launches in</div>
      <div className={styles.segments}>
        {segments.map(({ value, unit }) => (
          <div key={unit} className={styles.segment}>
            <span className={`display ${styles.value}`}>{value}</span>
            <span className={styles.unit}>{unit}</span>
          </div>
        ))}
      </div>
    </div>
  )
}
