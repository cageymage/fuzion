import { useMemo } from 'react'
import { Link } from 'react-router-dom'
import { FeaturedNewsCard } from '../../components/FeaturedNewsCard/FeaturedNewsCard'
import { LiveStreamEmbed } from '../../components/LiveStreamEmbed/LiveStreamEmbed'
import { NewsCard } from '../../components/NewsCard/NewsCard'
import { NextRaidCard } from '../../components/NextRaidCard/NextRaidCard'
import { SuggestedVideoCard } from '../../components/SuggestedVideoCard/SuggestedVideoCard'
import styles from './Home.module.css'
import { useHomeData } from './useHomeData'

export function Home() {
  const { news, nextRaid, liveStreams, suggestedVideo } = useHomeData()
  const nobodyIsLive = liveStreams.isSuccess && liveStreams.data.length === 0
  const vaultVideo = nobodyIsLive ? suggestedVideo.data : null
  const [featuredPost, ...remainingPosts] = news.data ?? []
  const featuredStream = useMemo(
    () => liveStreams.data?.[Math.floor(Math.random() * liveStreams.data.length)],
    [liveStreams.data],
  )

  return (
    <>
      <section className={styles.hero}>
        <div>
          <div className="eyebrow">Emberreach · US</div>
          <h1 className={styles.title}>Fuzion</h1>
          <div className={styles.tagline}>Raiding · Dungeons · Community</div>
        </div>
        {nextRaid.data ? (
          <NextRaidCard raid={nextRaid.data} />
        ) : (
          <div className={`card ${styles.notice}`}>
            <div className="eyebrow eyebrow-muted">Next Raid</div>
            <p className={styles.noticeText}>
              {nextRaid.isPending ? 'Checking the raid schedule…' : 'No raid scheduled yet.'}
            </p>
          </div>
        )}
      </section>

      <div className={styles.divider} />

      <div className={styles.grid}>
        <section>
          <header className={styles.sectionHeader}>
            <h2 className={styles.sectionTitle}>Latest News</h2>
            <Link to="/news" className={styles.viewAll}>
              View all →
            </Link>
          </header>
          {news.isPending && <p className={styles.noticeText}>Loading the latest news…</p>}
          {news.isError && <p className={styles.noticeText}>Latest news could not be loaded.</p>}
          {featuredPost && <FeaturedNewsCard post={featuredPost} />}
          {remainingPosts.map((post) => (
            <NewsCard key={post.id} post={post} />
          ))}
          {news.isSuccess && news.data.length === 0 && (
            <p className={styles.noticeText}>No news posted yet.</p>
          )}
        </section>

        <section>
          <header className={styles.sectionHeader}>
            <h2 className={styles.sectionTitle}>{vaultVideo ? 'From the Vault' : 'Live Now'}</h2>
          </header>
          {liveStreams.isPending && <p className={styles.noticeText}>Checking who is live…</p>}
          {liveStreams.isError && (
            <p className={styles.noticeText}>Live streams could not be loaded.</p>
          )}
          {featuredStream && <LiveStreamEmbed key={featuredStream.id} stream={featuredStream} />}
          {vaultVideo && <SuggestedVideoCard video={vaultVideo} />}
          {nobodyIsLive && (suggestedVideo.isError || suggestedVideo.data === null) && (
            <p className={styles.noticeText}>Nobody is streaming right now.</p>
          )}
        </section>
      </div>
    </>
  )
}
