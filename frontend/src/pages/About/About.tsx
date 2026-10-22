import { Link } from 'react-router-dom'
import heroDesktopAvif from '../../assets/about-hero-desktop.avif'
import heroDesktopJpg from '../../assets/about-hero-desktop.jpg'
import heroDesktopWebp from '../../assets/about-hero-desktop.webp'
import heroMobileAvif from '../../assets/about-hero-mobile.avif'
import heroMobileJpg from '../../assets/about-hero-mobile.jpg'
import heroMobileWebp from '../../assets/about-hero-mobile.webp'
import styles from './About.module.css'

const heroFetchPriority = { fetchpriority: 'high' }

function GroupIcon() {
  return (
    <svg
      className={styles.statIcon}
      width="24"
      height="24"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <circle cx="9" cy="8" r="3" />
      <path d="M3 19c0-3.3 2.7-5 6-5s6 1.7 6 5" />
      <circle cx="17" cy="9" r="2.2" />
      <path d="M16 14.2c2.9 0 5 1.4 5 4.3" />
    </svg>
  )
}

function LinkedIcon() {
  return (
    <svg
      className={styles.statIcon}
      width="24"
      height="24"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      aria-hidden="true"
    >
      <circle cx="9" cy="12" r="5" />
      <circle cx="15" cy="12" r="5" />
    </svg>
  )
}

function CalendarIcon() {
  return (
    <svg
      className={styles.statIcon}
      width="24"
      height="24"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <rect x="4" y="5" width="16" height="15" rx="2" />
      <path d="M4 10h16M8 3v4M16 3v4" />
    </svg>
  )
}

export function About() {
  return (
    <article className={styles.page}>
      <header className={styles.hero}>
        <div className={styles.heroText}>
          <div className="eyebrow">PvE · US · World of Warcraft: Forever</div>
          <h1 className={styles.title}>About Fuzion</h1>
          <p className={styles.tagline}>Serious about clearing. Not about ourselves.</p>
          <p className={styles.lead}>
            A PvE guild that takes the raid seriously and everything else lightly.
          </p>
          <Link to="/applications" className={styles.apply}>
            Apply to Fuzion
          </Link>
        </div>
        <picture className={styles.heroImage}>
          <source media="(max-width: 720px)" type="image/avif" srcSet={heroMobileAvif} />
          <source media="(max-width: 720px)" type="image/webp" srcSet={heroMobileWebp} />
          <source media="(max-width: 720px)" type="image/jpeg" srcSet={heroMobileJpg} />
          <source type="image/avif" srcSet={heroDesktopAvif} />
          <source type="image/webp" srcSet={heroDesktopWebp} />
          <img
            src={heroDesktopJpg}
            width={1000}
            height={500}
            alt="A group of adventurers silhouetted in a snowy canyon, facing a glowing amber gateway in an ancient fortress."
            {...heroFetchPriority}
          />
        </picture>
      </header>

      <ul className={styles.stats} aria-label="Guild at a glance">
        <li className={styles.stat}>
          <GroupIcon />
          <span className={styles.statValue}>20</span>
          <span>person core team</span>
        </li>
        <li className={styles.stat}>
          <LinkedIcon />
          <span className={styles.statValue}>40</span>
          <span>with partner guilds</span>
        </li>
        <li className={`${styles.stat} ${styles.statOpen}`}>
          <CalendarIcon />
          <span className={styles.statValue}>TBA</span>
          <span>raid nights, set with the roster</span>
        </li>
      </ul>

      <section aria-labelledby="founder-heading" className={styles.section}>
        <h2 id="founder-heading" className={styles.sectionTitle}>
          Who&apos;s behind Fuzion
        </h2>
        <p>
          Fuzion was founded by{' '}
          <a href="https://fuziongaming.gg/" target="_blank" rel="noopener noreferrer">
            Centrifuze
          </a>
          , a streamer who plays WoW (Season of Discovery and retail), OSRS, The Division 2, Hitman
          and more, and hosts the CCW Pro Wrestling Supershow on Twitch. Fuzion has raided retail
          together since 2024, and Forever is the next chapter. Raids here run just serious enough
          to get things done, with the jokes kept in.
        </p>
        <p>
          Want to see how we play? Catch raid nights and more on the{' '}
          <Link to="/streams">streams page</Link>.
        </p>
      </section>

      <section aria-labelledby="culture-heading" className={styles.section}>
        <h2 id="culture-heading" className={styles.sectionTitle}>
          Culture
        </h2>
        <p>
          Serious about the raid, relaxed about everything else. We play because it is fun, and we
          would rather you enjoy the game than treat it like a second job. Most of the guild lives
          in Discord, raid nights and off nights alike.
        </p>
      </section>

      <section aria-labelledby="faq-heading" className={styles.section}>
        <h2 id="faq-heading" className={styles.sectionTitle}>
          Questions
        </h2>
        <h3 className={styles.question}>Which server will you be on?</h3>
        <p>
          The PvE server. There is only one PvE megaserver, so we will update this once we know
          whether it gets its own realm name.
        </p>
        <h3 className={styles.question}>When do you raid?</h3>
        <p>
          Raid nights are not set yet. We will pick them with the roster and post them on the{' '}
          <Link to="/calendar">calendar</Link>.
        </p>
        <h3 className={styles.question}>How do I join?</h3>
        <p>
          Send in an application and an officer will get back to you.
        </p>
      </section>

      <section aria-labelledby="recruiting-heading" className={styles.section}>
        <h2 id="recruiting-heading" className={styles.sectionTitle}>
          Who we are looking for
        </h2>
        <p>
          We are building the founding roster now: enough players for a 20-person raid team, ready
          to team up with other guilds for 40-person raids. Be easy to play with, open to feedback
          and here for the guild, not only its loot.
        </p>
        <Link to="/applications" className={styles.apply}>
          Apply to Fuzion
        </Link>
      </section>
    </article>
  )
}
