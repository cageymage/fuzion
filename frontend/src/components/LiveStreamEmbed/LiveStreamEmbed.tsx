import { formatViewerCount } from '../../lib/format'
import { twitchChannelFromUrl } from '../../lib/twitch'
import type { Stream } from '../../types/streams'
import cardStyles from '../LiveStreamCard/LiveStreamCard.module.css'
import { LiveStreamCard } from '../LiveStreamCard/LiveStreamCard'
import { StreamerAvatar } from '../StreamerAvatar/StreamerAvatar'
import styles from './LiveStreamEmbed.module.css'

interface LiveStreamEmbedProps {
  stream: Stream
}

function playerUrl(channel: string): string {
  const params = new URLSearchParams({
    channel,
    // Twitch refuses to load the player unless parent matches the serving domain.
    parent: window.location.hostname,
    muted: 'true',
    autoplay: 'true',
  })
  return `https://player.twitch.tv/?${params}`
}

export function LiveStreamEmbed({ stream }: LiveStreamEmbedProps) {
  const channel = twitchChannelFromUrl(stream.channelUrl)
  if (!channel) return <LiveStreamCard stream={stream} />

  return (
    <div className={`card ${cardStyles.card}`}>
      <div className={styles.player}>
        <iframe
          className={styles.frame}
          src={playerUrl(channel)}
          title={`${stream.streamerName} live stream`}
          allow="autoplay; fullscreen"
          allowFullScreen
        />
      </div>
      <div className={cardStyles.streamer}>
        <StreamerAvatar name={stream.streamerName} avatarUrl={stream.avatarUrl} size="small" />
        <span>{stream.streamerName} is live</span>
        <span className={styles.viewerCount}>{formatViewerCount(stream.viewerCount)}</span>
      </div>
      {stream.title && (
        <p className={cardStyles.title} title={stream.title}>
          {stream.title}
        </p>
      )}
      <div className={cardStyles.game}>{stream.gameName}</div>
      <a className={cardStyles.watchLink} href={stream.channelUrl} target="_blank" rel="noreferrer">
        Watch on Twitch →
      </a>
    </div>
  )
}
