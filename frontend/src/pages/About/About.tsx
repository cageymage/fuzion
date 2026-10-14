import { Link } from 'react-router-dom'
import styles from './About.module.css'

export function About() {
  return (
    <article className={styles.page}>
      <div className="eyebrow">PvE · US · World of Warcraft: Forever</div>
      <h1 className={styles.title}>About Fuzion</h1>
      <p className={styles.lead}>
        Fuzion is a PvE guild built around raiding, dungeons and a community that sticks together
        between tiers.
      </p>
      <Link to="/applications" className={styles.apply}>
        Apply to Fuzion
      </Link>

      <section aria-labelledby="schedule-heading" className={styles.section}>
        <h2 id="schedule-heading" className={styles.sectionTitle}>
          Raid schedule
        </h2>
        <p>
          We raid on set nights each week. Upcoming raids and events are posted on the{' '}
          <Link to="/calendar">calendar</Link>.
        </p>
      </section>

      <section aria-labelledby="progression-heading" className={styles.section}>
        <h2 id="progression-heading" className={styles.sectionTitle}>
          Progression
        </h2>
        <p>
          We push current content steadily without burning people out. See where we stand on the{' '}
          <Link to="/raid-progress">raid progress</Link> page.
        </p>
      </section>

      <section aria-labelledby="culture-heading" className={styles.section}>
        <h2 id="culture-heading" className={styles.sectionTitle}>
          Culture
        </h2>
        <p>
          Members show up prepared, help each other improve and keep things friendly. Most of the
          guild lives in Discord, raid nights and off nights alike.
        </p>
      </section>

      <section aria-labelledby="recruiting-heading" className={styles.section}>
        <h2 id="recruiting-heading" className={styles.sectionTitle}>
          Who we are looking for
        </h2>
        <p>
          Players who are reliable on raid nights, open to feedback and want to be part of the
          guild, not only its loot.
        </p>
        <Link to="/applications" className={styles.apply}>
          Apply to Fuzion
        </Link>
      </section>

      <section aria-labelledby="credits-heading" className={styles.credits}>
        <h2 id="credits-heading" className={styles.creditsTitle}>
          Credits
        </h2>
        <p>Thanks to the services and people who keep this site running:</p>
        <ul>
          <li>Hosting: Render.</li>
          <li>Bot protection: Cloudflare Turnstile.</li>
          <li>Built with React, Vite and Go.</li>
          <li>Home page and news card artwork: generated with Midjourney.</li>
          <li>The officers and members who test the site and report what breaks.</li>
        </ul>
      </section>
    </article>
  )
}
