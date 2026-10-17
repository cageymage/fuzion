import styles from './Charter.module.css'

export function Charter() {
  return (
    <article className={styles.page}>
      <h1 className={styles.title}>Guild charter</h1>
      <p className={styles.intro}>
        This is a draft for the guild leaders to review, so details such as the raid schedule and
        the loot system may change. Nothing here is meant to surprise you later, which is why we
        write it down. Applying means you have read it.
      </p>

      <section aria-labelledby="rules-heading" className={styles.section}>
        <h2 id="rules-heading" className={styles.sectionTitle}>
          Guild rules
        </h2>
        <p>Play by the game&apos;s rules. In Fuzion that means none of the following:</p>
        <ul>
          <li>Cheating, botting or using exploits.</li>
          <li>Buying gold or selling gold or items for real money.</li>
          <li>Sharing or selling accounts.</li>
          <li>Running or joining GDKP raids.</li>
        </ul>
      </section>

      <section aria-labelledby="conduct-heading" className={styles.section}>
        <h2 id="conduct-heading" className={styles.sectionTitle}>
          How we treat people
        </h2>
        <p>
          Be civil to guildmates and to people outside the guild. Joking around is part of the
          fun, and adult conversation and humor are to be expected here. Read the room, be mindful
          of others, and if something bothers you, talk to an officer.
        </p>
      </section>

      <section aria-labelledby="raids-heading" className={styles.section}>
        <h2 id="raids-heading" className={styles.sectionTitle}>
          Raids
        </h2>
        <ul>
          <li>We raid 10, 20 and 40 player content.</li>
          <li>The raid schedule is not set yet. It will be announced on Discord.</li>
          <li>
            Come prepared with the basics where you can. Potions and enchants are encouraged.
          </li>
          <li>
            We do not expect full knowledge of every encounter, the best gear, or every buff. Ask
            questions and learn as you go.
          </li>
          <li>
            Our goal is to clear content while it is current. We are not racing for world firsts.
          </li>
        </ul>
      </section>

      <section aria-labelledby="loot-heading" className={styles.section}>
        <h2 id="loot-heading" className={styles.sectionTitle}>
          Loot
        </h2>
        <p>
          We expect to use the in-game roll system. This is not final, and if it changes, officers
          will announce it before it applies.
        </p>
        <ul>
          <li>Main spec rolls come before off spec rolls.</li>
          <li>
            If an item is only a minor upgrade for you but a big one for someone else, think
            before you roll. If you are unsure, ask in raid chat.
          </li>
          <li>
            If a roll clearly leaves someone short-changed, officers may step in to sort it out.
          </li>
        </ul>
      </section>
    </article>
  )
}
