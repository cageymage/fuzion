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
          This policy covers the Fuzion guild site, forever.fuziongaming.gg. It collects only what
          it needs to run the guild:
        </p>
        <ul>
          <li>
            Application fields: your name, character name, class, role, expected playstyle,
            availability, Discord handle and notes.
          </li>
          <li>
            Discord identity: when you log in with Discord, your Discord ID and username, plus a
            session cookie that keeps you logged in and a short-lived cookie that remembers which
            page you were on so we can send you back to it after you log in, deleted when login
            finishes.
          </li>
          <li>
            Discord server check: when you log in, we also ask Discord which servers you are in,
            only to record whether you are in the guild&apos;s Discord server. We store that yes or
            no, not the list, and use it to decide who can answer guild surveys.
          </li>
          <li>
            Battle.net link: if you choose to link Battle.net on your dashboard, your Battle.net ID
            and BattleTag. While linking, we also read the list of World of Warcraft characters on
            that account once, to record which guild roster characters are yours. We keep only that
            ownership, not the list or Battle.net&apos;s access token. Unlinking deletes both and
            removes you as the owner of those characters.
          </li>
          <li>
            Survey responses: your answers, the guild name you optionally enter, and your Discord
            username at the time you respond. Officers can see these per person, and the public
            results show totals only.
          </li>
          <li>Images uploaded by officers, such as news post images.</li>
          <li>
            Art and screenshot submissions: when a guild member submits images, we keep each exact
            original file, a small preview, its file name, the optional description, who submitted
            it, and when they accepted the license and which version of its wording. Originals keep
            any data the file carries, such as EXIF details and, for some phone photos, the GPS
            location where it was taken. Only officers can see submissions; they are never public.
            Officers can delete a submission at any time, which removes the original and the
            preview.
          </li>
        </ul>
        <p>Other services are involved in running the site:</p>
        <ul>
          <li>Cloudflare Turnstile checks that application submissions come from a person.</li>
          <li>Twitch and YouTube provide live status, thumbnails and embedded streams and videos.</li>
          <li>
            Blizzard Battle.net signs you in when you link your account and tells us which
            characters are on it.
          </li>
          <li>Render hosts the site and its database.</li>
        </ul>
        <p>
          We do not sell your data. To have your application or other data removed, ask an officer
          on Discord.
        </p>
        <p>
          fuziongaming.gg and other fuziongaming.gg sites are run by the same people and have their
          own notices. Logging in with Discord on one of them does not log you in on another.
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
        <p>
          When you submit art or screenshots, you confirm that you created each image or bought the
          rights to it (game screenshots are fine), and you grant the guild an unrestricted license
          to use it on this site and in related social media posts. You keep your copyright.
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
          <li>Embedded videos, live status and channel avatars: YouTube, a Google LLC service.</li>
          <li>Bot protection: Cloudflare, Inc.</li>
          <li>Hosting: Render Services, Inc.</li>
          <li>Home page artwork: generated with Midjourney.</li>
          <li>News card background artwork: generated with Midjourney.</li>
          <li>About page artwork: generated with Midjourney.</li>
          <li>
            GitHub and the GitHub logo: trademarks of GitHub, Inc., shown only to link to this
            site&apos;s source code. Icon shape from Simple Icons (CC0).
          </li>
        </ul>
      </section>
    </article>
  )
}
