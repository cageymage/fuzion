import { useQuery } from '@tanstack/react-query'
import { fetchStreams } from '../../api/streams'
import { OfflineStreamCard } from '../../components/OfflineStreamCard/OfflineStreamCard'
import { StreamCard } from '../../components/StreamCard/StreamCard'
import styles from './Streams.module.css'

export function Streams() {
  const streams = useQuery({ queryKey: ['streams'], queryFn: fetchStreams })

  const live = streams.data?.filter((stream) => stream.isLive) ?? []
  const offline = streams.data?.filter((stream) => !stream.isLive) ?? []

  return (
    <section className={styles.page}>
      <h1 className={styles.title}>Streams</h1>
      {streams.isPending && <p className={styles.noticeText}>Loading streams…</p>}
      {streams.isError && <p className={styles.noticeText}>Streams could not be loaded.</p>}
      {streams.isSuccess && streams.data.length === 0 && (
        <p className={styles.noticeText}>No one has added a stream channel yet.</p>
      )}
      {streams.isSuccess && live.length > 0 && (
        <section aria-labelledby="live-streams-heading">
          <h2 id="live-streams-heading" className={styles.sectionTitle}>
            Live now
          </h2>
          <div className={styles.grid}>
            {live.map((stream) => (
              <StreamCard key={stream.id} stream={stream} />
            ))}
          </div>
        </section>
      )}
      {streams.isSuccess && live.length === 0 && offline.length > 0 && (
        <p className={styles.noticeText}>Nobody is live right now.</p>
      )}
      {streams.isSuccess && offline.length > 0 && (
        <section aria-labelledby="offline-streams-heading" className={styles.offlineSection}>
          <h2 id="offline-streams-heading" className={styles.sectionTitle}>
            Offline
          </h2>
          <div className={styles.offlineGrid}>
            {offline.map((stream) => (
              <OfflineStreamCard key={stream.id} stream={stream} />
            ))}
          </div>
        </section>
      )}
    </section>
  )
}
