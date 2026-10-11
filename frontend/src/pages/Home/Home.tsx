import { useMemo } from 'react'
import { Link } from 'react-router-dom'
import heroDesktopAvif from '../../assets/hero-desktop.avif'
import heroDesktopJpg from '../../assets/hero-desktop.jpg'
import heroDesktopWebp from '../../assets/hero-desktop.webp'
import heroMobileAvif from '../../assets/hero-mobile.avif'
import heroMobileJpg from '../../assets/hero-mobile.jpg'
import heroMobileWebp from '../../assets/hero-mobile.webp'
import { FeaturedNewsCard } from '../../components/FeaturedNewsCard/FeaturedNewsCard'
import { LiveStreamEmbed } from '../../components/LiveStreamEmbed/LiveStreamEmbed'
import { NewsCard } from '../../components/NewsCard/NewsCard'
import { NextRaidCard } from '../../components/NextRaidCard/NextRaidCard'
import { SuggestedVideoCard } from '../../components/SuggestedVideoCard/SuggestedVideoCard'
import { LoadingPlaceholder, Skeleton } from '../../components/Skeleton/Skeleton'
import styles from './Home.module.css'
import { useHomeData } from './useHomeData'

// React 18 warns on the camelCase fetchPriority prop; the lowercase attribute is the supported spelling until React 19.
const heroFetchPriority = { fetchpriority: 'high' }

const FEATURED_NEWS_HEIGHT = 260
const NEWS_CARD_HEIGHT = 120

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
        <picture className={styles.heroImage}>
          <source media="(max-width: 720px)" type="image/avif" srcSet={heroMobileAvif} />
          <source media="(max-width: 720px)" type="image/webp" srcSet={heroMobileWebp} />
          <source media="(max-width: 720px)" type="image/jpeg" srcSet={heroMobileJpg} />
          <source type="image/avif" srcSet={heroDesktopAvif} />
          <source type="image/webp" srcSet={heroDesktopWebp} />
          <img src={heroDesktopJpg} width={2400} height={900} alt="" {...heroFetchPriority} />
        </picture>
        <div className={styles.heroText}>
          <div className="eyebrow">PvE · US</div>
          <h1 className={styles.title}>Fuzion</h1>
          <div className={styles.tagline}>Raiding · Dungeons · Community</div>
        </div>
        {nextRaid.data ? (
          <NextRaidCard raid={nextRaid.data} />
        ) : (
          <div className={`card ${styles.notice}`}>
            <div className="eyebrow eyebrow-muted">Next Raid</div>
            {nextRaid.isPending ? (
              <LoadingPlaceholder label="Checking the raid schedule…">
                <Skeleton height={128} className={styles.noticeSkeleton} />
              </LoadingPlaceholder>
            ) : (
              <p className={styles.noticeText}>No raid scheduled yet.</p>
            )}
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
          {news.isPending && (
            <LoadingPlaceholder label="Loading the latest news…">
              <Skeleton height={FEATURED_NEWS_HEIGHT} className={styles.newsSkeleton} />
              <Skeleton height={NEWS_CARD_HEIGHT} className={styles.newsSkeleton} />
              <Skeleton height={NEWS_CARD_HEIGHT} className={styles.newsSkeleton} />
            </LoadingPlaceholder>
          )}
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
          {liveStreams.isPending && (
            <LoadingPlaceholder label="Checking who is live…">
              <Skeleton height="auto" className={styles.videoSkeleton} />
            </LoadingPlaceholder>
          )}
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
