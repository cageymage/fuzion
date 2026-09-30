import type { SuggestedVideo } from '../../types/streams'
import cardStyles from '../LiveStreamCard/LiveStreamCard.module.css'
import styles from './SuggestedVideoCard.module.css'

interface SuggestedVideoCardProps {
  video: SuggestedVideo
}

export function SuggestedVideoCard({ video }: SuggestedVideoCardProps) {
  return (
    <div className={`card ${cardStyles.card}`}>
      <div className={styles.player}>
        <iframe
          className={styles.frame}
          src={`https://www.youtube.com/embed/${encodeURIComponent(video.id)}`}
          title={video.title}
          allow="fullscreen"
          allowFullScreen
        />
      </div>
      <p className={`${cardStyles.title} ${styles.title}`} title={video.title}>
        {video.title}
      </p>
      <a
        className={cardStyles.watchLink}
        href={`https://www.youtube.com/watch?v=${encodeURIComponent(video.id)}`}
        target="_blank"
        rel="noreferrer"
      >
        Watch on YouTube →
      </a>
    </div>
  )
}
