import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Link, NavLink } from 'react-router-dom'
import { loginUrl, logout } from '../../api/auth'
import { navItems } from '../../app/navigation'
import { currentUserQueryKey, useCurrentUser } from '../../hooks/useCurrentUser'
import styles from './AppHeader.module.css'

export function AppHeader() {
  return (
    <header className={styles.header}>
      <Link to="/" className={styles.brand}>
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" aria-hidden="true">
          <path d="M12 2 L20 12 L12 22 L4 12 Z" stroke="var(--gold)" strokeWidth="1.5" />
          <path d="M12 2 L20 12 L12 22 L4 12 Z" fill="var(--gold)" fillOpacity="0.15" />
        </svg>
        <span className={styles.wordmark}>FUZION</span>
      </Link>
      <nav className={styles.nav} aria-label="Main">
        {navItems.map((item) => (
          <NavLink
            key={item.path}
            to={item.path}
            end={item.path === '/'}
            className={({ isActive }) =>
              isActive ? `${styles.navLink} ${styles.active}` : styles.navLink
            }
          >
            {item.label}
          </NavLink>
        ))}
      </nav>
      <AccountMenu />
    </header>
  )
}

function AccountMenu() {
  const currentUser = useCurrentUser()
  const queryClient = useQueryClient()
  const logoutMutation = useMutation({
    mutationFn: logout,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: currentUserQueryKey }),
  })

  if (currentUser.isPending) {
    return <div className={styles.account} aria-hidden="true" />
  }

  if (!currentUser.data) {
    // A plain anchor, not a router Link: the server has to answer this one so it can redirect to Discord.
    return (
      <div className={styles.account}>
        <a href={loginUrl} className={styles.loginLink}>
          Log in with Discord
        </a>
      </div>
    )
  }

  const user = currentUser.data
  return (
    <div className={styles.account}>
      {user.avatarUrl ? (
        <img src={user.avatarUrl} alt="" className={styles.avatar} />
      ) : (
        <span className={styles.avatarFallback} aria-hidden="true">
          {user.username.slice(0, 1).toUpperCase()}
        </span>
      )}
      <span className={styles.username}>{user.username}</span>
      <button
        type="button"
        className={styles.logoutButton}
        onClick={() => logoutMutation.mutate()}
        disabled={logoutMutation.isPending}
      >
        Log out
      </button>
    </div>
  )
}
