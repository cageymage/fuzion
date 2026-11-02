import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useSearchParams } from 'react-router-dom'
import { battlenetLinkUrl, unlinkBattlenet } from '../../api/auth'
import { currentUserQueryKey, useCurrentUser } from '../../hooks/useCurrentUser'
import { useLoginUrl } from '../../hooks/useLoginUrl'
import styles from './Dashboard.module.css'
import { MyCharacters } from './MyCharacters'
import { myCharactersQueryKey } from './myCharactersQueryKey'

export function Dashboard() {
  const currentUser = useCurrentUser()
  const loginUrl = useLoginUrl()
  const queryClient = useQueryClient()
  const [searchParams] = useSearchParams()
  const unlink = useMutation({
    mutationFn: unlinkBattlenet,
    onSuccess: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: currentUserQueryKey }),
        queryClient.invalidateQueries({ queryKey: myCharactersQueryKey }),
      ]),
  })

  if (currentUser.isPending) {
    return null
  }

  const user = currentUser.data
  if (!user) {
    return (
      <article className={styles.page}>
        <h1 className={styles.title}>Dashboard</h1>
        <p>
          <a href={loginUrl}>Log in with Discord</a> to manage your linked accounts.
        </p>
      </article>
    )
  }

  return (
    <article className={styles.page}>
      <h1 className={styles.title}>Dashboard</h1>

      <section aria-labelledby="discord-heading" className={styles.section}>
        <h2 id="discord-heading" className={styles.sectionTitle}>
          Discord
        </h2>
        <p>
          Logged in as <strong>{user.username}</strong>.
        </p>
      </section>

      <section aria-labelledby="battlenet-heading" className={styles.section}>
        <h2 id="battlenet-heading" className={styles.sectionTitle}>
          Battle.net
        </h2>
        {searchParams.get('battlenet') === 'already-linked' && (
          <p role="alert" className={styles.error}>
            That Battle.net account is already linked to another member. Ask an officer on Discord
            if it should be moved.
          </p>
        )}
        {searchParams.get('battlenet') === 'characters-unavailable' && (
          <p role="status" className={styles.hint}>
            Battle.net is linked, but its World of Warcraft characters could not be read, so none
            were claimed and the characters you already had are unchanged. Try linking again later.
          </p>
        )}
        {user.battlenetLinked ? (
          <>
            <p>
              Linked as <strong>{user.battletag}</strong>.
            </p>
            <button
              type="button"
              className={styles.button}
              onClick={() => unlink.mutate()}
              disabled={unlink.isPending}
            >
              Unlink Battle.net
            </button>
            {unlink.isError && (
              <p role="alert" className={styles.error}>
                Could not unlink Battle.net, try again.
              </p>
            )}
          </>
        ) : (
          <>
            <p>Link your Battle.net account to connect your World of Warcraft characters.</p>
            {/* A plain anchor, not a router Link: the server redirects to Battle.net. */}
            <a href={battlenetLinkUrl} className={styles.linkButton}>
              Link Battle.net
            </a>
          </>
        )}
      </section>

      {user.battlenetLinked && <MyCharacters />}
    </article>
  )
}
