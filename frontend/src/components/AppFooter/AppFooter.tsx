import styles from './AppFooter.module.css'

export function AppFooter() {
  return (
    <footer className={styles.footer}>
      Fuzion · Emberreach · World of Warcraft: Forever
      <span className={styles.credit}>Profession icons © Blizzard Entertainment</span>
    </footer>
  )
}
