import { formatViewerCount } from '../../lib/format'
import type { LiveStream } from '../../types/streams'
import styles from './LiveStreamCard.module.css'

interface LiveStreamCardProps {
  stream: LiveStream
}

export function LiveStreamCard({ stream }: LiveStreamCardProps) {
  return (
    <div className={`card ${styles.card}`}>
      <div className={styles.preview}>
        {stream.thumbnailUrl && (
          <img className={styles.previewImage} src={stream.thumbnailUrl} alt="" />
        )}
        <span className={styles.liveBadge}>LIVE</span>
        <span className={styles.viewerCount}>{formatViewerCount(stream.viewerCount)}</span>
        <span className={styles.playIcon} aria-hidden="true">
          <svg width="32" height="32" viewBox="0 0 24 24">
            <path d="M4 3 L4 21 L21 12 Z" fill="rgba(241,230,214,0.85)" />
          </svg>
        </span>
      </div>
      <div className={styles.streamer}>{stream.streamerName} is live</div>
      <div className={styles.game}>{stream.gameName}</div>
      <a className={styles.watchLink} href={stream.channelUrl} target="_blank" rel="noreferrer">
        Watch on Twitch →
      </a>
    </div>
  )
}
