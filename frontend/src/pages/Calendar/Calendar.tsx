import { useQuery } from '@tanstack/react-query'
import { fetchUpcomingRaids } from '../../api/raids'
import { formatRaidDay, formatRaidTime } from '../../lib/format'
import type { Raid } from '../../types/raids'
import styles from './Calendar.module.css'

function groupByDay(raids: Raid[]): { day: string; raids: Raid[] }[] {
  const days: { day: string; raids: Raid[] }[] = []
  for (const raid of raids) {
    const day = formatRaidDay(raid.startsAt)
    const lastDay = days[days.length - 1]
    if (lastDay?.day === day) {
      lastDay.raids.push(raid)
    } else {
      days.push({ day, raids: [raid] })
    }
  }
  return days
}

export function Calendar() {
  const raids = useQuery({ queryKey: ['raids'], queryFn: fetchUpcomingRaids })

  return (
    <div className={styles.page}>
      <h1 className={styles.title}>Calendar &amp; Events</h1>
      {raids.isPending && <p className={styles.notice}>Loading the raid schedule…</p>}
      {raids.isError && <p className={styles.notice}>The raid schedule could not be loaded.</p>}
      {raids.isSuccess && raids.data.length === 0 && (
        <p className={styles.notice}>No raids scheduled.</p>
      )}
      {raids.data &&
        groupByDay(raids.data).map(({ day, raids: dayRaids }) => (
          <section key={day} className={styles.day}>
            <h2 className={styles.dayHeading}>{day}</h2>
            <ul className={styles.raidList}>
              {dayRaids.map((raid) => (
                <li key={raid.id} className={`card ${styles.raid}`}>
                  <time className={styles.time} dateTime={raid.startsAt}>
                    {formatRaidTime(raid.startsAt)}
                  </time>
                  <div>
                    <div className={styles.raidName}>
                      {raid.difficulty} · {raid.instanceName}
                    </div>
                    <div className={styles.progress}>{raid.progressSummary}</div>
                  </div>
                </li>
              ))}
            </ul>
          </section>
        ))}
    </div>
  )
}
