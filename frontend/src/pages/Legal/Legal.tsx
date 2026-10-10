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
        <p>The Fuzion site collects only what it needs to run the guild:</p>
        <ul>
          <li>
            Application fields: your name, character name, class, role, availability, Discord
            handle and notes.
          </li>
          <li>
            Discord identity: when you log in with Discord, your Discord ID and username, plus a
            session cookie that keeps you logged in.
          </li>
          <li>
            Battle.net link: if you choose to link Battle.net on your dashboard, your Battle.net ID
            and BattleTag. Unlinking deletes both.
          </li>
          <li>Images uploaded by officers, such as news post images.</li>
        </ul>
        <p>Other services are involved in running the site:</p>
        <ul>
          <li>Cloudflare Turnstile checks that application submissions come from a person.</li>
          <li>Twitch and YouTube provide live status, thumbnails and embedded streams and videos.</li>
          <li>Blizzard Battle.net signs you in when you link your account.</li>
          <li>Render hosts the site and its database.</li>
        </ul>
        <p>
          We do not sell your data. To have your application or other data removed, ask an officer
          on Discord.
        </p>
      </section>

      <section aria-labelledby="copyright-heading" className={styles.section}>
        <h2 id="copyright-heading" className={styles.sectionTitle}>
          Copyright and content
        </h2>
        <p>
          Images and text posted by officers and members remain the property of their authors, who
          confirm they have the right to share them. Guild news, names and branding belong to
          Fuzion. If you believe something on this site infringes your rights, ask an officer on
          Discord and it will be reviewed and removed where appropriate.
        </p>
      </section>

      <section aria-labelledby="blizzard-heading" className={styles.section}>
        <h2 id="blizzard-heading" className={styles.sectionTitle}>
          Blizzard and World of Warcraft
        </h2>
        <p>
          This site is a fan project run by a guild. It is not affiliated with or endorsed by
          Blizzard Entertainment, Inc. World of Warcraft and Blizzard Entertainment are trademarks
          or registered trademarks of Blizzard Entertainment, Inc. in the United States and other
          countries. Game names, images and other game content remain the property of Blizzard
          Entertainment, Inc.
        </p>
      </section>

      <section aria-labelledby="attributions-heading" className={styles.section}>
        <h2 id="attributions-heading" className={styles.sectionTitle}>
          Attributions
        </h2>
        <ul>
          <li>World of Warcraft game content and class names: Blizzard Entertainment, Inc.</li>
          <li>Live status, channel avatars and embedded streams: Twitch Interactive, Inc.</li>
          <li>Embedded videos: YouTube, a Google LLC service.</li>
          <li>Bot protection: Cloudflare, Inc.</li>
          <li>Hosting: Render Services, Inc.</li>
          <li>Home page artwork: generated with Midjourney.</li>
        </ul>
      </section>
    </article>
  )
}
