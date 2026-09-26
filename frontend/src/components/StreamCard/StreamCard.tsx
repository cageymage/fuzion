import { formatViewerCount } from '../../lib/format'
import type { Stream } from '../../types/streams'
import styles from './StreamCard.module.css'

interface StreamCardProps {
  stream: Stream
}

export function StreamCard({ stream }: StreamCardProps) {
  return (
    <a className={`card ${styles.card}`} href={stream.channelUrl} target="_blank" rel="noreferrer">
      {stream.isLive && stream.thumbnailUrl && (
        <img className={styles.thumbnail} src={stream.thumbnailUrl} alt="" />
      )}
      <div className={styles.header}>
        <span className={styles.name}>{stream.streamerName}</span>
        {stream.isLive && <span className={styles.liveBadge}>LIVE</span>}
      </div>
      {stream.isLive && (
        <div className={styles.details}>
          <span>{stream.gameName}</span>
          <span>{formatViewerCount(stream.viewerCount)}</span>
        </div>
      )}
    </a>
  )
}
