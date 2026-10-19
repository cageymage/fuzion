import { Link } from 'react-router-dom'
import styles from './NotFound.module.css'

export function NotFound() {
  return (
    <section className={styles.page}>
      <h1 className={styles.title}>Page not found</h1>
      <p>
        <Link to="/">Back to Fuzion Gaming</Link>
      </p>
    </section>
  )
}
