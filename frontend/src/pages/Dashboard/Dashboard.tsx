import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useSearchParams } from 'react-router-dom'
import { battlenetLinkUrl, loginUrl, unlinkBattlenet } from '../../api/auth'
import { currentUserQueryKey, useCurrentUser } from '../../hooks/useCurrentUser'
import styles from './Dashboard.module.css'

export function Dashboard() {
  const currentUser = useCurrentUser()
  const queryClient = useQueryClient()
  const [searchParams] = useSearchParams()
  const unlink = useMutation({
    mutationFn: unlinkBattlenet,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: currentUserQueryKey }),
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
    </article>
  )
}
