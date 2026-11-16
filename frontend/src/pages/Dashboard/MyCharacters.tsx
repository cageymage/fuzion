import { useQuery } from '@tanstack/react-query'
import { battlenetLinkUrl } from '../../api/auth'
import { fetchMyRankRequests, myRankRequestsQueryKey } from '../../api/rankRequests'
import { fetchMyCharacters } from '../../api/roster'
import { classColor } from '../../lib/classColor'
import { CharacterSpecs } from './CharacterSpecs'
import styles from './Dashboard.module.css'
import { myCharactersQueryKey } from './myCharactersQueryKey'
import { RankRequestControl } from './RankRequestControl'

export function MyCharacters() {
  const characters = useQuery({ queryKey: myCharactersQueryKey, queryFn: fetchMyCharacters })
  const rankRequests = useQuery({ queryKey: myRankRequestsQueryKey, queryFn: fetchMyRankRequests })

  return (
    <section aria-labelledby="characters-heading" className={styles.section}>
      <div className={styles.sectionHeader}>
        <h2 id="characters-heading" className={styles.sectionTitle}>
          Your characters
        </h2>
        {/* A plain anchor, not a router Link: the server redirects to Battle.net. */}
        <a href={battlenetLinkUrl} className={styles.linkButton}>
          Refresh characters
        </a>
      </div>
      <p className={styles.hint}>
        Refreshing picks up characters added to the guild roster since you linked Battle.net.
      </p>
      {characters.isPending && <p>Loading your characters…</p>}
      {characters.isError && (
        <p role="alert" className={styles.error}>
          Could not load your characters, try again later.
        </p>
      )}
      {characters.data?.length === 0 && (
        <p>
          None of the characters on your Battle.net account are on the guild roster yet. Once they
          are, refresh your characters to claim them.
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
                  <CharacterSpecs character={character} fullName={fullName} />
                  {character.professions.length > 0 && (
                    <p className={styles.hint}>
                      {character.professions
                        .map(({ profession, skillLevel }) => `${profession} ${skillLevel}`)
                        .join(' · ')}
                    </p>
                  )}
                  {rankRequests.isSuccess && (
                    <RankRequestControl
                      character={character}
                      fullName={fullName}
                      request={rankRequests.data.find((request) => request.characterId === character.id)}
                    />
                  )}
                </li>
              )
            })}
          </ul>
          {rankRequests.isError && (
            <p role="alert" className={styles.error}>
              Could not load your rank requests, try again later.
            </p>
          )}
          <p className={styles.hint}>
            Main or alt follows your guild rank in game, so ask an officer with a rank request to
            change it. Professions come from Battle.net and raid teams are set by officers.
          </p>
        </>
      )}
    </section>
  )
}
