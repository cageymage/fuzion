import { Link } from 'react-router-dom'
import styles from './HubFooter.module.css'

export function HubFooter() {
  return (
    <footer className={styles.footer}>
      <span>Fuzion Gaming</span>
      <nav aria-label="Footer" className={styles.nav}>
        <Link className={styles.link} to="/legal">
          Legal
        </Link>
      </nav>
    </footer>
  )
}
