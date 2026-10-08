import { useMemo, useState } from 'react'
import { useCurrentUser } from '../../hooks/useCurrentUser'
import { classColor } from '../../lib/classColor'
import { primaryProfessionIcons } from '../../lib/professionIcon'
import { raceIcon } from '../../lib/raceIcon'
import { CharacterDialog } from './CharacterDialog'
import { DeleteCharacterDialog } from './DeleteCharacterDialog'
import { filterRoster, sortRoster, type RoleFilter, type RosterSort, type SortKey } from './filterRoster'
import type { Character } from '../../types/roster'
import styles from './Roster.module.css'
import { roleLabel, specLabel } from './rosterLabels'
import { RosterSummary } from './RosterSummary'
import { summarizeRoster } from './summarizeRoster'
import { useRosterData } from './useRosterData'

const roleFilters: { value: RoleFilter; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'tank', label: 'Tank' },
  { value: 'healer', label: 'Healer' },
  { value: 'dps', label: 'DPS' },
]

type DialogState =
  { kind: 'add' } | { kind: 'edit'; character: Character } | { kind: 'delete'; character: Character }

interface SortHeaderProps {
  label: string
  sortKey: SortKey
  sort: RosterSort | null
  onSort: (key: SortKey) => void
}

function SortHeader({ label, sortKey, sort, onSort }: SortHeaderProps) {
  const direction = sort?.key === sortKey ? sort.direction : null
  const ariaSort = direction === 'asc' ? 'ascending' : direction === 'desc' ? 'descending' : 'none'
  return (
    <th scope="col" aria-sort={ariaSort}>
      <button type="button" className={styles.sortButton} onClick={() => onSort(sortKey)}>
        {label}
        {direction && <span aria-hidden="true">{direction === 'asc' ? ' ▲' : ' ▼'}</span>}
      </button>
    </th>
  )
}

function PencilIcon() {
  return (
    <svg
      width="16"
      height="16"
      viewBox="0 0 16 16"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.4"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M2.5 13.5 3 10l7.5-7.5 3 3L6 13z" />
      <path d="M9 4l3 3" />
    </svg>
  )
}

function TrashIcon() {
  return (
    <svg
      width="16"
      height="16"
      viewBox="0 0 16 16"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.4"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M2.5 4h11M6 4V2.5h4V4M4 4l.6 9h6.8L12 4M6.5 6.5v4M9.5 6.5v4" />
    </svg>
  )
}

// Stands in for races Blizzard has no icon for, so every race stays one icon wide.
function GenericRaceBadge({ race }: { race: string }) {
  return (
    <svg
      className={styles.raceIcon}
      width="24"
      height="24"
      viewBox="0 0 24 24"
      fill="none"
      stroke="var(--gold)"
      strokeWidth="1.5"
      strokeLinecap="round"
      role="img"
      aria-label={race}
    >
      <title>{race}</title>
      <path d="M4 8h10a2.5 2.5 0 1 0-2.5-2.5" />
      <path d="M4 12.5h14a2.5 2.5 0 1 1-2.5 2.5" />
      <path d="M4 17h7a2 2 0 1 1-2 2" />
    </svg>
  )
}

function RaceCell({ race }: { race: string | null }) {
  if (race === null) return null
  const iconUrl = raceIcon(race)
  if (iconUrl === null) return <GenericRaceBadge race={race} />
  return <img className={styles.raceIcon} src={iconUrl} alt={race} title={race} width={24} height={24} />
}

export function Roster() {
  const roster = useRosterData()
  const currentUser = useCurrentUser()
  const [role, setRole] = useState<RoleFilter>('all')
  const [className, setClassName] = useState('all')
  const [showAlts, setShowAlts] = useState(false)
  const [search, setSearch] = useState('')
  const [sort, setSort] = useState<RosterSort | null>(null)
  const [dialog, setDialog] = useState<DialogState | null>(null)
  const closeDialog = () => setDialog(null)
  const canManage = currentUser.data?.isOfficer === true

  // Ascending, then descending, then back to the roster's default order.
  function toggleSort(key: SortKey) {
    if (sort?.key !== key) setSort({ key, direction: 'asc' })
    else if (sort.direction === 'asc') setSort({ key, direction: 'desc' })
    else setSort(null)
  }

  const classes = useMemo(
    () => Array.from(new Set((roster.data ?? []).map((character) => character.class))).sort(),
    [roster.data],
  )

  const visibleCharacters = sortRoster(
    filterRoster(roster.data ?? [], { role, className, showAlts, search }),
    sort,
  )

  return (
    <section className={styles.page}>
      <h1 className={styles.title}>Roster</h1>
      {canManage && (
        <div className={styles.toolbar}>
          <button type="button" className={styles.button} onClick={() => setDialog({ kind: 'add' })}>
            Add character
          </button>
        </div>
      )}

      {roster.isSuccess && <RosterSummary summary={summarizeRoster(visibleCharacters)} />}

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

        <label className={styles.classField} htmlFor="roster-search">
          Name
          <input
            id="roster-search"
            type="search"
            className={styles.searchInput}
            placeholder="Search name…"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
          />
        </label>

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
          {visibleCharacters.length === 0 ? (
            <p className={styles.noticeText}>No characters match.</p>
          ) : (
            <div className={styles.tableScroll} role="region" aria-label="Roster table" tabIndex={0}>
              <table className={styles.table}>
                <thead>
                  <tr>
                    <SortHeader label="Race" sortKey="race" sort={sort} onSort={toggleSort} />
                    <SortHeader label="Name" sortKey="name" sort={sort} onSort={toggleSort} />
                    <th scope="col">Secondary name</th>
                    <SortHeader label="Level" sortKey="level" sort={sort} onSort={toggleSort} />
                    <SortHeader label="Class" sortKey="class" sort={sort} onSort={toggleSort} />
                    <th scope="col">Spec</th>
                    <SortHeader label="Role" sortKey="role" sort={sort} onSort={toggleSort} />
                    <SortHeader label="Raid Team" sortKey="raidTeam" sort={sort} onSort={toggleSort} />
                    <th scope="col">Main/Alt</th>
                    <th scope="col">Professions</th>
                    {canManage && (
                      <th scope="col">
                        <span className={styles.visuallyHidden}>Actions</span>
                      </th>
                    )}
                  </tr>
                </thead>
                <tbody>
                  {visibleCharacters.map((character) => (
                    <tr key={character.id}>
                      <td>
                        <RaceCell race={character.race} />
                      </td>
                      <td>{character.name}</td>
                      <td>{character.secondaryName}</td>
                      <td>{character.level}</td>
                      <td style={{ color: classColor(character.class) }}>{character.class}</td>
                      <td>{specLabel(character)}</td>
                      <td>{roleLabel(character)}</td>
                      <td>{character.raidTeam ?? '—'}</td>
                      <td>{character.isMain ? 'Main' : 'Alt'}</td>
                      <td>
                        <span className={styles.professionIcons}>
                          {primaryProfessionIcons(character.professions).map(({ profession, iconUrl }) => (
                            <img
                              key={profession}
                              className={styles.professionIcon}
                              src={iconUrl}
                              alt={profession}
                              title={profession}
                              width={24}
                              height={24}
                            />
                          ))}
                        </span>
                      </td>
                      {canManage && (
                        <td>
                          <div className={styles.rowActions}>
                            <button
                              type="button"
                              className={styles.smallButton}
                              title={`Edit ${character.name}`}
                              aria-label={`Edit ${character.name}`}
                              onClick={() => setDialog({ kind: 'edit', character })}
                            >
                              <PencilIcon />
                            </button>
                            <button
                              type="button"
                              className={`${styles.smallButton} ${styles.dangerButton}`}
                              title={`Delete ${character.name}`}
                              aria-label={`Delete ${character.name}`}
                              onClick={() => setDialog({ kind: 'delete', character })}
                            >
                              <TrashIcon />
                            </button>
                          </div>
                        </td>
                      )}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}

      {dialog?.kind === 'add' && <CharacterDialog onClose={closeDialog} />}
      {dialog?.kind === 'edit' && <CharacterDialog character={dialog.character} onClose={closeDialog} />}
      {dialog?.kind === 'delete' && (
        <DeleteCharacterDialog character={dialog.character} onClose={closeDialog} />
      )}
    </section>
  )
}
