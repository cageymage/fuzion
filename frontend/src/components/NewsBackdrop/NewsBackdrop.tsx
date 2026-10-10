import guildNewsAvif from '../../assets/news/guild-news.avif'
import guildNewsWebp from '../../assets/news/guild-news.webp'
import patchNotesAvif from '../../assets/news/patch-notes.avif'
import patchNotesWebp from '../../assets/news/patch-notes.webp'
import raidProgressAvif from '../../assets/news/raid-progress.avif'
import raidProgressWebp from '../../assets/news/raid-progress.webp'
import recruitmentAvif from '../../assets/news/recruitment.avif'
import recruitmentWebp from '../../assets/news/recruitment.webp'
import type { NewsCategory } from '../../types/news'
import styles from './NewsBackdrop.module.css'

const categoryArt: Record<NewsCategory, { avif: string; webp: string }> = {
  'raid-progress': { avif: raidProgressAvif, webp: raidProgressWebp },
  recruitment: { avif: recruitmentAvif, webp: recruitmentWebp },
  'guild-news': { avif: guildNewsAvif, webp: guildNewsWebp },
  'patch-notes': { avif: patchNotesAvif, webp: patchNotesWebp },
}

interface NewsBackdropProps {
  category: NewsCategory
  imageUrl?: string | null
}

export function NewsBackdrop({ category, imageUrl }: NewsBackdropProps) {
  const art = categoryArt[category]

  return (
    <div className={`${styles.backdrop} ${styles[category]}`} aria-hidden="true">
      {imageUrl ? (
        <img className={styles.image} src={imageUrl} alt="" loading="lazy" />
      ) : (
        <picture>
          <source type="image/avif" srcSet={art.avif} />
          <img className={styles.image} src={art.webp} alt="" loading="lazy" />
        </picture>
      )}
    </div>
  )
}
