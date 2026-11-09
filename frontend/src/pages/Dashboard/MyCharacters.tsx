import { useQuery } from '@tanstack/react-query'
import { battlenetLinkUrl } from '../../api/auth'
import { fetchMyCharacters } from '../../api/roster'
import { classColor } from '../../lib/classColor'
import styles from './Dashboard.module.css'
import { myCharactersQueryKey } from './myCharactersQueryKey'
import { MyProfessions } from './MyProfessions'

export function MyCharacters() {
  const characters = useQuery({ queryKey: myCharactersQueryKey, queryFn: fetchMyCharacters })

  return (
    <section aria-labelledby="characters-heading" className={styles.section}>
      <h2 id="characters-heading" className={styles.sectionTitle}>
        Your characters
      </h2>
      {characters.isPending && <p>Loading your characters…</p>}
      {characters.isError && (
        <p role="alert" className={styles.error}>
          Could not load your characters, try again later.
        </p>
      )}
      {characters.data?.length === 0 && (
        <p>
          None of the characters on your Battle.net account are on the guild roster yet. Once an
          officer adds them, <a href={battlenetLinkUrl}>link Battle.net again</a> to claim them.
        </p>
      )}
      {characters.data && characters.data.length > 0 && (
        <>
          <ul className={styles.characters} aria-label="Characters">
            {characters.data.map((character) => {
              const fullName = `${character.name} ${character.secondaryName}`.trim()
              return (
                <li key={character.id} className={styles.character}>
                  <div className={styles.characterHeader} data-testid="character-header">
                    <span>
                      <strong style={{ color: classColor(character.class) }}>{fullName}</strong> ·{' '}
                      <span style={{ color: classColor(character.class) }}>{character.class}</span>
                    </span>
                    {character.rank && <span className={styles.rankBadge}>{character.rank}</span>}
                  </div>
                  <MyProfessions character={character} fullName={fullName} />
                </li>
              )
            })}
          </ul>
          <p className={styles.hint}>
            Missing a character? <a href={battlenetLinkUrl}>Link Battle.net again</a> to pick up
            ones added to the roster since you linked. Main or alt follows your guild rank in game,
            so ask an officer to change it. Raid teams are set by officers.
          </p>
        </>
      )}
    </section>
  )
}
