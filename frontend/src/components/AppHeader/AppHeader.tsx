import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useId, useRef, useState, type KeyboardEvent } from 'react'
import { Link, NavLink } from 'react-router-dom'
import { loginUrl, logout } from '../../api/auth'
import { navItems } from '../../app/navigation'
import logo144 from '../../assets/logo-144.webp'
import logo48 from '../../assets/logo-48.webp'
import logo96 from '../../assets/logo-96.webp'
import { currentUserQueryKey, useCurrentUser } from '../../hooks/useCurrentUser'
import styles from './AppHeader.module.css'

export function AppHeader() {
  const [menuOpen, setMenuOpen] = useState(false)
  const menuButtonRef = useRef<HTMLButtonElement>(null)
  const navId = useId()

  function closeMenuOnEscape(event: KeyboardEvent) {
    if (event.key === 'Escape' && menuOpen) {
      setMenuOpen(false)
      menuButtonRef.current?.focus()
    }
  }

  return (
    <header className={styles.header} onKeyDown={closeMenuOnEscape}>
      <Link to="/" className={styles.brand}>
        <img
          className={styles.logo}
          src={logo48}
          srcSet={`${logo48} 1x, ${logo96} 2x, ${logo144} 3x`}
          width={48}
          height={48}
          alt=""
        />
        <span className={styles.wordmark}>FUZION</span>
      </Link>
      <button
        ref={menuButtonRef}
        type="button"
        className={styles.menuButton}
        aria-expanded={menuOpen}
        aria-controls={navId}
        onClick={() => setMenuOpen((open) => !open)}
      >
        Menu
      </button>
      <nav
        id={navId}
        className={menuOpen ? `${styles.nav} ${styles.navOpen}` : styles.nav}
        aria-label="Main"
      >
        {navItems.map((item) => (
          <NavLink
            key={item.path}
            to={item.path}
            end={item.path === '/'}
            onClick={() => setMenuOpen(false)}
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
      <Link to="/dashboard" className={styles.username}>
        {user.username}
      </Link>
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
