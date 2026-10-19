import { foreverGuildUrl, twitchUrl, youTubeUrl } from '../../app/links'
import { LiveStreamCard } from '../../components/LiveStreamCard/LiveStreamCard'
import { PlatformIcon } from '../../components/PlatformIcon/PlatformIcon'
import { SuggestedVideoCard } from '../../components/SuggestedVideoCard/SuggestedVideoCard'
import styles from './Home.module.css'
import { useHomeData } from './useHomeData'

export function Home() {
  const { liveStreams, suggestedVideo, nobodyIsLive } = useHomeData()
  const vaultVideo = nobodyIsLive ? suggestedVideo.data : null

  return (
    <>
      <section className={styles.hero}>
        <h1 className={styles.title}>Fuzion Gaming</h1>
        <p className={styles.tagline}>
          A gaming community of guildmates, streamers and friends. Everything Fuzion starts here.
        </p>
      </section>

      <section aria-labelledby="guilds-heading" className={styles.section}>
        <h2 id="guilds-heading" className={styles.sectionTitle}>
          Our guilds
        </h2>
        <a className={`card ${styles.guildCard}`} href={foreverGuildUrl}>
          <span className="eyebrow">World of Warcraft: Forever · PvE · US</span>
          <span className={styles.guildName}>Fuzion</span>
          <span className={styles.guildBlurb}>
            Raiding, dungeons and community. Read the news, check the roster and apply to join.
          </span>
          <span className={styles.guildLink}>Visit forever.fuziongaming.gg →</span>
        </a>
      </section>

      <section aria-labelledby="find-us-heading" className={styles.section}>
        <h2 id="find-us-heading" className={styles.sectionTitle}>
          Find us
        </h2>
        <ul className={styles.socialLinks}>
          <li>
            <a className={styles.socialLink} href={twitchUrl} target="_blank" rel="noreferrer">
              <PlatformIcon platform="twitch" />
              Twitch
            </a>
          </li>
          <li>
            <a className={styles.socialLink} href={youTubeUrl} target="_blank" rel="noreferrer">
              <PlatformIcon platform="youtube" />
              YouTube
            </a>
          </li>
        </ul>
      </section>

      <section aria-labelledby="streams-heading" className={styles.section}>
        <h2 id="streams-heading" className={styles.sectionTitle}>
          {vaultVideo ? 'From the Vault' : 'Live Now'}
        </h2>
        {liveStreams.isPending && <p className={styles.noticeText}>Checking who is live…</p>}
        {liveStreams.isError && (
          <p className={styles.noticeText}>Live streams could not be loaded.</p>
        )}
        {liveStreams.isSuccess && liveStreams.data.length > 0 && (
          <div className={styles.streamGrid}>
            {liveStreams.data.map((stream) => (
              <LiveStreamCard key={stream.id} stream={stream} />
            ))}
          </div>
        )}
        {vaultVideo && (
          <div className={styles.video}>
            <SuggestedVideoCard video={vaultVideo} />
          </div>
        )}
        {nobodyIsLive && (suggestedVideo.isError || suggestedVideo.data === null) && (
          <p className={styles.noticeText}>Nobody is streaming right now.</p>
        )}
      </section>
    </>
  )
}
