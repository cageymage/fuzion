import { useQuery } from '@tanstack/react-query'
import { fetchStreams } from '../../api/streams'
import { StreamCard } from '../../components/StreamCard/StreamCard'
import styles from './Streams.module.css'

export function Streams() {
  const streams = useQuery({ queryKey: ['streams'], queryFn: fetchStreams })

  return (
    <section className={styles.page}>
      <h1 className={styles.title}>Streams</h1>
      {streams.isPending && <p className={styles.noticeText}>Loading streams…</p>}
      {streams.isError && <p className={styles.noticeText}>Streams could not be loaded.</p>}
      {streams.isSuccess && streams.data.length === 0 && (
        <p className={styles.noticeText}>No one has added a stream channel yet.</p>
      )}
      {streams.isSuccess && streams.data.length > 0 && (
        <div className={styles.grid}>
          {streams.data.map((stream) => (
            <StreamCard key={stream.id} stream={stream} />
          ))}
        </div>
      )}
    </section>
  )
}
