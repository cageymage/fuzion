import { formatViewerCount } from '../../lib/format'
import { streamPlatformNames, type Stream } from '../../types/streams'
import { PlatformIcon } from '../PlatformIcon/PlatformIcon'
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
      </div>
      <div className={styles.streamer}>
        {stream.avatarUrl && <img className={styles.avatar} src={stream.avatarUrl} alt="" />}
        <span>{stream.streamerName} is live</span>
      </div>
      {stream.title && (
        <p className={styles.title} title={stream.title}>
          {stream.title}
        </p>
      )}
      <a className={styles.watchLink} href={stream.channelUrl} target="_blank" rel="noreferrer">
        <PlatformIcon platform={stream.platform} size={18} />
        Watch {stream.streamerName} on {streamPlatformNames[stream.platform]} →
      </a>
    </div>
  )
}
