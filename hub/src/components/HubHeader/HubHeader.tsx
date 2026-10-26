import { Link } from 'react-router-dom'
import logo48 from '../../assets/logo-48.webp'
import logo96 from '../../assets/logo-96.webp'
import logo144 from '../../assets/logo-144.webp'
import styles from './HubHeader.module.css'

export function HubHeader() {
  return (
    <header className={styles.header}>
      <Link to="/" className={styles.brand}>
        <img
          className={styles.logo}
          src={logo48}
          srcSet={`${logo48} 1x, ${logo96} 2x, ${logo144} 3x`}
          width={48}
          height={48}
          alt=""
        />
        <span className={styles.wordmark}>FUZION GAMING</span>
      </Link>
    </header>
  )
}
