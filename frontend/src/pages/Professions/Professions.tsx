import { useState } from 'react'
import { classColor } from '../../lib/classColor'
import { groupProfessions } from './groupProfessions'
import styles from './Professions.module.css'
import { useProfessionsData } from './useProfessionsData'

export function Professions() {
  const professions = useProfessionsData()
  const [query, setQuery] = useState('')

  const groups = groupProfessions(professions.data ?? [], query)
  const trimmedQuery = query.trim()

  return (
    <section className={styles.page}>
      <h1 className={styles.title}>Professions</h1>

      <label className={styles.searchField} htmlFor="professions-search">
        Search
        <input
          id="professions-search"
          type="search"
          className={styles.search}
          placeholder="Profession or character"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
        />
      </label>

      {professions.isPending && <p className={styles.noticeText}>Loading professions…</p>}
      {professions.isError && <p className={styles.noticeText}>Professions could not be loaded.</p>}

      {professions.isSuccess && professions.data.length === 0 && (
        <p className={styles.noticeText}>Nobody has recorded a profession yet.</p>
      )}
      {professions.isSuccess && professions.data.length > 0 && groups.length === 0 && (
        <p className={styles.noticeText}>No matches for "{trimmedQuery}".</p>
      )}

      {groups.map((group) => {
        const headingId = `profession-${group.profession.replace(/\s+/g, '-')}`
        return (
          <section key={group.profession} className={styles.group} aria-labelledby={headingId}>
            <h2 id={headingId} className={styles.groupTitle}>
              {group.profession}
            </h2>
            <ul className={styles.entries}>
              {group.entries.map((entry) => (
                <li key={entry.id} className={styles.entry}>
                  <span>
                    <span className={styles.name} style={{ color: classColor(entry.character.class) }}>
                      {entry.character.name}
                    </span>{' '}
                    <span className={styles.secondaryName}>{entry.character.secondaryName}</span>
                  </span>
                  <span className={styles.skill}>{entry.skillLevel}</span>
                </li>
              ))}
            </ul>
          </section>
        )
      })}
    </section>
  )
}
