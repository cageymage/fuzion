import { Link } from 'react-router-dom'
import styles from './AppFooter.module.css'

export function AppFooter() {
  return (
    <footer className={styles.footer}>
      <span>Fuzion · PvE · World of Warcraft: Forever</span>
      <nav aria-label="Footer" className={styles.nav}>
        <Link className={styles.link} to="/about">
          About
        </Link>
        <Link className={styles.link} to="/charter">
          Charter
        </Link>
        <Link className={styles.link} to="/legal">
          Legal
        </Link>
      </nav>
    </footer>
  )
}
