import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { battlenetLinkUrl } from '../../api/auth'
import { fetchMyCharacters, updateMyCharacter } from '../../api/roster'
import styles from './Dashboard.module.css'

export const myCharactersQueryKey = ['me', 'characters'] as const

export function MyCharacters() {
  const queryClient = useQueryClient()
  const characters = useQuery({ queryKey: myCharactersQueryKey, queryFn: fetchMyCharacters })
  const makeMain = useMutation({
    mutationFn: (id: string) => updateMyCharacter(id, { isMain: true }),
    onSuccess: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: myCharactersQueryKey }),
        queryClient.invalidateQueries({ queryKey: ['roster'] }),
      ]),
  })

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
          <ul className={styles.characters}>
            {characters.data.map((character) => {
              const fullName = `${character.name} ${character.secondaryName}`.trim()
              return (
                <li key={character.id} className={styles.character}>
                  <span>
                    <strong>{fullName}</strong> · {character.class}
                  </span>
                  {character.isMain ? (
                    <span className={styles.mainBadge}>Main</span>
                  ) : (
                    <button
                      type="button"
                      className={styles.button}
                      aria-label={`Make ${fullName} your main`}
                      onClick={() => makeMain.mutate(character.id)}
                      disabled={makeMain.isPending}
                    >
                      Make main
                    </button>
                  )}
                </li>
              )
            })}
          </ul>
          {makeMain.isError && (
            <p role="alert" className={styles.error}>
              Could not change your main, try again.
            </p>
          )}
          <p className={styles.hint}>
            Missing a character? <a href={battlenetLinkUrl}>Link Battle.net again</a> to pick up
            ones added to the roster since you linked. Raid teams are set by officers.
          </p>
        </>
      )}
    </section>
  )
}
