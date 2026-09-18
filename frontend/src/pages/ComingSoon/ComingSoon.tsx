import styles from './ComingSoon.module.css'

interface ComingSoonProps {
  title: string
}

export function ComingSoon({ title }: ComingSoonProps) {
  return (
    <section className={styles.section}>
      <div className="eyebrow">Fuzion</div>
      <h1 className={styles.title}>{title}</h1>
      <p className={styles.note}>This page has not been built yet.</p>
    </section>
  )
}
