import { formatViewerCount } from '../../lib/format'
import type { Stream } from '../../types/streams'
import { StreamerAvatar } from '../StreamerAvatar/StreamerAvatar'
import styles from './LiveStreamCard.module.css'

interface LiveStreamCardProps {
  stream: Stream
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
      <div className={styles.streamer}>
        <StreamerAvatar name={stream.streamerName} avatarUrl={stream.avatarUrl} size="small" />
        <span>{stream.streamerName} is live</span>
      </div>
      {stream.title && (
        <p className={styles.title} title={stream.title}>
          {stream.title}
        </p>
      )}
      <a className={styles.watchLink} href={stream.channelUrl} target="_blank" rel="noreferrer">
        Watch on Twitch →
      </a>
    </div>
  )
}
