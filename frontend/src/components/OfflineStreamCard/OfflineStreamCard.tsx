import type { Stream } from '../../types/streams'
import { StreamerAvatar } from '../StreamerAvatar/StreamerAvatar'
import styles from './OfflineStreamCard.module.css'

interface OfflineStreamCardProps {
  stream: Stream
}

export function OfflineStreamCard({ stream }: OfflineStreamCardProps) {
  return (
    <a className={`card ${styles.card}`} href={stream.channelUrl} target="_blank" rel="noreferrer">
      <StreamerAvatar name={stream.streamerName} avatarUrl={stream.avatarUrl} size="medium" />
      <span className={styles.name}>{stream.streamerName}</span>
    </a>
  )
}
