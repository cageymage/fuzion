import { formatViewerCount } from '../../lib/format'
import type { Stream } from '../../types/streams'
import { StreamerAvatar } from '../StreamerAvatar/StreamerAvatar'
import styles from './StreamCard.module.css'

interface StreamCardProps {
  stream: Stream
}

export function StreamCard({ stream }: StreamCardProps) {
  return (
    <a className={`card ${styles.card}`} href={stream.channelUrl} target="_blank" rel="noreferrer">
      {stream.thumbnailUrl && <img className={styles.thumbnail} src={stream.thumbnailUrl} alt="" />}
      <div className={styles.header}>
        <span className={styles.streamer}>
          <StreamerAvatar name={stream.streamerName} avatarUrl={stream.avatarUrl} size="medium" />
          <span className={styles.name}>{stream.streamerName}</span>
        </span>
        <span className={styles.liveBadge}>LIVE</span>
      </div>
      {stream.title && (
        <p className={styles.title} title={stream.title}>
          {stream.title}
        </p>
      )}
      <div className={styles.details}>
        <span>{stream.gameName}</span>
        <span>{formatViewerCount(stream.viewerCount)}</span>
      </div>
    </a>
  )
}
