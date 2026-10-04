import { useEffect, useState } from 'react'
import { createPortal } from 'react-dom'
import styles from './LightboxImage.module.css'

interface LightboxImageProps {
  src?: string
  alt?: string
}

const uploadedImagePath = /^\/api\/images\/[0-9a-f-]{36}$/i

export function LightboxImage({ src, alt = '' }: LightboxImageProps) {
  const [open, setOpen] = useState(false)
  const thumbnailSrc = src && uploadedImagePath.test(src) ? `${src}/thumb` : src

  useEffect(() => {
    if (!open) return
    function closeOnEscape(event: KeyboardEvent) {
      if (event.key === 'Escape') setOpen(false)
    }
    document.addEventListener('keydown', closeOnEscape)
    return () => document.removeEventListener('keydown', closeOnEscape)
  }, [open])

  return (
    <>
      <button
        type="button"
        className={styles.thumbnailButton}
        aria-label={alt ? `Open image: ${alt}` : 'Open image'}
        onClick={() => setOpen(true)}
      >
        <img className={styles.thumbnail} src={thumbnailSrc} alt={alt} />
      </button>
      {open &&
        // A portal, because Markdown puts images inside <p>, which cannot contain a <div>.
        createPortal(
          <div
            className={styles.backdrop}
            role="dialog"
            aria-modal="true"
            aria-label="Image preview"
            onClick={() => setOpen(false)}
          >
            <img className={styles.fullImage} src={src} alt={alt} />
          </div>,
          document.body,
        )}
    </>
  )
}
