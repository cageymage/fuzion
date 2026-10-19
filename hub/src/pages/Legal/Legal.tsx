import { foreverGuildUrl } from '../../app/links'
import styles from './Legal.module.css'

export function Legal() {
  return (
    <article className={styles.page}>
      <h1 className={styles.title}>Legal</h1>
      <p className={styles.intro}>
        This text is boilerplate written for the guild to review. It is not legal advice.
      </p>

      <section aria-labelledby="privacy-heading" className={styles.section}>
        <h2 id="privacy-heading" className={styles.sectionTitle}>
          Privacy policy
        </h2>
        <p>
          This policy covers fuziongaming.gg. The site has no login, forms or uploads, so it
          collects no personal data itself and sets no cookies of its own.
        </p>
        <p>Other services are involved in running the site:</p>
        <ul>
          <li>Render hosts the site.</li>
          <li>
            Twitch and YouTube provide live status, thumbnails and channel avatars. Videos embedded
            from YouTube load from YouTube, which may set its own cookies.
          </li>
          <li>Google Fonts serves the site&apos;s fonts.</li>
        </ul>
        <p>
          Other fuziongaming.gg sites are run by the same people and have their own notices. For
          what the guild site collects, such as applications and Discord logins, see the{' '}
          <a href={`${foreverGuildUrl}/legal`}>forever.fuziongaming.gg legal page</a>.
        </p>
      </section>

      <section aria-labelledby="copyright-heading" className={styles.section}>
        <h2 id="copyright-heading" className={styles.sectionTitle}>
          Copyright and content
        </h2>
        <p>
          Fuzion names and branding belong to Fuzion. Streams and videos shown here remain the
          property of their creators. If you believe something on this site infringes your rights,
          ask an officer on Discord and it will be reviewed and removed where appropriate.
        </p>
      </section>

      <section aria-labelledby="blizzard-heading" className={styles.section}>
        <h2 id="blizzard-heading" className={styles.sectionTitle}>
          Blizzard and World of Warcraft
        </h2>
        <p>
          This site is a fan project run by a gaming community. It is not affiliated with or
          endorsed by Blizzard Entertainment, Inc. World of Warcraft and Blizzard Entertainment are
          trademarks or registered trademarks of Blizzard Entertainment, Inc. in the United States
          and other countries.
        </p>
      </section>

      <section aria-labelledby="attributions-heading" className={styles.section}>
        <h2 id="attributions-heading" className={styles.sectionTitle}>
          Attributions
        </h2>
        <ul>
          <li>Live status, channel avatars and thumbnails: Twitch Interactive, Inc.</li>
          <li>Embedded videos, live status and channel avatars: YouTube, a Google LLC service.</li>
          <li>
            Twitch and YouTube logos: trademarks of Twitch Interactive, Inc. and Google LLC, shown
            only to link to Fuzion&apos;s channels. Icon shapes from Simple Icons (CC0).
          </li>
          <li>
            GitHub and the GitHub logo: trademarks of GitHub, Inc., shown only to link to this
            site&apos;s source code. Icon shape from Simple Icons (CC0).
          </li>
          <li>Fonts: Google Fonts, a Google LLC service.</li>
          <li>Hosting: Render Services, Inc.</li>
        </ul>
      </section>
    </article>
  )
}
