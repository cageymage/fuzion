import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { useCurrentUser } from '../../hooks/useCurrentUser'
import { classColor } from '../../lib/classColor'
import { filterRoster, type RoleFilter } from './filterRoster'
import styles from './Roster.module.css'
import { roleLabel, specLabel } from './rosterLabels'
import { useRosterData } from './useRosterData'

const roleFilters: { value: RoleFilter; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'tank', label: 'Tank' },
  { value: 'healer', label: 'Healer' },
  { value: 'dps', label: 'DPS' },
]

export function Roster() {
  const roster = useRosterData()
  const currentUser = useCurrentUser()
  const [role, setRole] = useState<RoleFilter>('all')
  const [className, setClassName] = useState('all')
  const [showAlts, setShowAlts] = useState(false)

  const classes = useMemo(
    () => Array.from(new Set((roster.data ?? []).map((character) => character.class))).sort(),
    [roster.data],
  )

  const visibleCharacters = filterRoster(roster.data ?? [], { role, className, showAlts })

  return (
    <section className={styles.page}>
      <h1 className={styles.title}>Roster</h1>
      {currentUser.data?.isOfficer && (
        <Link to="/officer/roster" className={styles.manageLink}>
          Manage roster
        </Link>
      )}

      <div className={styles.filters}>
        <div className={styles.roleFilters}>
          {roleFilters.map(({ value, label }) => (
            <button
              key={value}
              type="button"
              className={styles.filter}
              aria-pressed={role === value}
              onClick={() => setRole(value)}
            >
              {label}
            </button>
          ))}
        </div>

        <label className={styles.classField} htmlFor="roster-class">
          Class
          <select
            id="roster-class"
            className={styles.classSelect}
            value={className}
            onChange={(event) => setClassName(event.target.value)}
          >
            <option value="all">All classes</option>
            {classes.map((value) => (
              <option key={value} value={value}>
                {value}
              </option>
            ))}
          </select>
        </label>

        <label className={styles.altsField} htmlFor="roster-show-alts">
          <input
            id="roster-show-alts"
            type="checkbox"
            className={styles.altsCheckbox}
            checked={showAlts}
            onChange={(event) => setShowAlts(event.target.checked)}
          />
          Show alts
        </label>
      </div>

      {roster.isPending && <p className={styles.noticeText}>Loading the roster…</p>}
      {roster.isError && <p className={styles.noticeText}>The roster could not be loaded.</p>}

      {roster.isSuccess && (
        <>
          <p className={styles.count}>{visibleCharacters.length} characters</p>
          {visibleCharacters.length === 0 ? (
            <p className={styles.noticeText}>No characters match.</p>
          ) : (
            <table className={styles.table}>
              <thead>
                <tr>
                  <th scope="col">Name</th>
                  <th scope="col">Class</th>
                  <th scope="col">Spec</th>
                  <th scope="col">Role</th>
                  <th scope="col">Main/Alt</th>
                  <th scope="col">Raid Team</th>
                </tr>
              </thead>
              <tbody>
                {visibleCharacters.map((character) => (
                  <tr key={character.id}>
                    <td>{character.name}</td>
                    <td style={{ color: classColor(character.class) }}>{character.class}</td>
                    <td>{specLabel(character)}</td>
                    <td>{roleLabel(character)}</td>
                    <td>{character.isMain ? 'Main' : 'Alt'}</td>
                    <td>{character.raidTeam ?? '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </>
      )}
    </section>
  )
}
