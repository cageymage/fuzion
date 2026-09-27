import styles from './StreamerAvatar.module.css'

interface StreamerAvatarProps {
  name: string
  avatarUrl: string | null
  size: 'small' | 'medium'
}

export function StreamerAvatar({ name, avatarUrl, size }: StreamerAvatarProps) {
  if (avatarUrl) {
    return <img className={`${styles.avatar} ${styles[size]}`} src={avatarUrl} alt="" />
  }

  return (
    <span className={`${styles.avatar} ${styles.initial} ${styles[size]}`} aria-hidden="true">
      {name.charAt(0).toUpperCase()}
    </span>
  )
}
