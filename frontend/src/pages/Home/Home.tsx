import { Link } from 'react-router-dom'
import { FeaturedNewsCard } from '../../components/FeaturedNewsCard/FeaturedNewsCard'
import { LiveStreamCard } from '../../components/LiveStreamCard/LiveStreamCard'
import { NewsCard } from '../../components/NewsCard/NewsCard'
import { NextRaidCard } from '../../components/NextRaidCard/NextRaidCard'
import styles from './Home.module.css'
import { useHomeData } from './useHomeData'

export function Home() {
  const { news, nextRaid, liveStreams } = useHomeData()
  const [featuredPost, ...remainingPosts] = news.data ?? []

  return (
    <>
      <section className={styles.hero}>
        <div>
          <div className="eyebrow">Emberreach · US</div>
          <h1 className={styles.title}>Fuzion</h1>
          <div className={styles.tagline}>Raiding · Mythic+ · PvP · Crafting</div>
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
            <h2 className={styles.sectionTitle}>Live Now</h2>
          </header>
          {liveStreams.isPending && <p className={styles.noticeText}>Checking who is live…</p>}
          {liveStreams.isError && (
            <p className={styles.noticeText}>Live streams could not be loaded.</p>
          )}
          {liveStreams.data?.map((stream) => (
            <LiveStreamCard key={stream.id} stream={stream} />
          ))}
          {liveStreams.isSuccess && liveStreams.data.length === 0 && (
            <p className={styles.noticeText}>Nobody is streaming right now.</p>
          )}
        </section>
      </div>
    </>
  )
}
