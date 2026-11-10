import { useState } from 'react'
import { classColor } from '../../lib/classColor'
import type { ProfessionEntry } from '../../types/professions'
import { CharacterRecipes } from './CharacterRecipes'
import { CrafterResults } from './CrafterResults'
import { crafterKey, groupProfessions } from './groupProfessions'
import styles from './Professions.module.css'
import { useProfessionsData } from './useProfessionsData'
import { useRecipes } from './useRecipes'

const minRecipeSearchLength = 2

export function Professions() {
  const professions = useProfessionsData()
  const [query, setQuery] = useState('')
  const [openGroups, setOpenGroups] = useState<ReadonlySet<string>>(new Set())

  const trimmedQuery = query.trim()
  const searchingRecipes = trimmedQuery.length >= minRecipeSearchLength
  const recipes = useRecipes({ query: trimmedQuery, characterId: '' }, searchingRecipes)
  const matchedRecipes = searchingRecipes ? (recipes.data ?? []) : []
  const crafters = new Set(
    matchedRecipes.flatMap((recipe) =>
      recipe.characters.map((character) => crafterKey(recipe.profession, character.id)),
    ),
  )

  const groups = groupProfessions(professions.data ?? [], query, crafters)
  const waitingForRecipes = searchingRecipes && recipes.isPending
  const searching = trimmedQuery !== ''
  const professionNames = new Set((professions.data ?? []).map((entry) => entry.profession))
  const allOpen = professionNames.size > 0 && openGroups.size === professionNames.size

  function toggleGroup(profession: string) {
    setOpenGroups((current) => {
      const next = new Set(current)
      if (!next.delete(profession)) next.add(profession)
      return next
    })
  }

  function toggleAll() {
    setOpenGroups(allOpen ? new Set() : professionNames)
  }

  return (
    <section className={styles.page}>
      <h1 className={styles.title}>Professions</h1>

      <label className={styles.searchField} htmlFor="professions-search">
        Search
        <input
          id="professions-search"
          type="search"
          className={styles.search}
          placeholder="Profession, character or recipe"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
        />
      </label>

      {professions.isSuccess && professions.data.length > 0 && !searching && (
        <button type="button" className={styles.expandAll} onClick={toggleAll}>
          {allOpen ? 'Collapse all' : 'Expand all'}
        </button>
      )}

      {professions.isPending && <p className={styles.noticeText}>Loading professions…</p>}
      {professions.isError && <p className={styles.noticeText}>Professions could not be loaded.</p>}

      {searchingRecipes && recipes.isError && <p className={styles.noticeText}>Recipes could not be loaded.</p>}
      {matchedRecipes.length > 0 && <CrafterResults recipes={matchedRecipes} />}

      {professions.isSuccess && professions.data.length === 0 && (
        <p className={styles.noticeText}>Nobody has recorded a profession yet.</p>
      )}
      {professions.isSuccess &&
        professions.data.length > 0 &&
        groups.length === 0 &&
        !waitingForRecipes &&
        matchedRecipes.length === 0 && <p className={styles.noticeText}>No matches for "{trimmedQuery}".</p>}

      {groups.map((group) => {
        const headingId = `profession-${group.profession.replace(/\s+/g, '-')}`
        const open = searching || openGroups.has(group.profession)
        return (
          <section key={group.profession} className={styles.group} aria-labelledby={headingId}>
            <h2 id={headingId} className={styles.groupTitle}>
              <button
                type="button"
                className={styles.groupButton}
                aria-expanded={open}
                disabled={searching}
                onClick={() => toggleGroup(group.profession)}
              >
                <span className={styles.chevron} aria-hidden="true">
                  {open ? '▾' : '▸'}
                </span>
                {group.profession}
              </button>
            </h2>
            {open && (
              <ul className={styles.entries}>
                {group.entries.map((entry) => (
                  <CharacterEntry key={entry.id} entry={entry} />
                ))}
              </ul>
            )}
          </section>
        )
      })}
    </section>
  )
}

function CharacterEntry({ entry }: { entry: ProfessionEntry }) {
  const [expanded, setExpanded] = useState(false)
  const color = classColor(entry.character.class)

  return (
    <li className={styles.characterEntry}>
      <button
        type="button"
        className={styles.entryButton}
        aria-expanded={expanded}
        onClick={() => setExpanded((open) => !open)}
      >
        <span>
          <span className={styles.chevron} aria-hidden="true">
            {expanded ? '▾' : '▸'}
          </span>
          <span className={styles.name} style={{ color }}>
            {entry.character.name}
          </span>{' '}
          <span style={{ color }}>{entry.character.secondaryName}</span>
        </span>
        <span className={styles.skill}>{entry.skillLevel > 0 ? entry.skillLevel : '—'}</span>
      </button>
      {expanded && (
        <div className={styles.recipePanel}>
          <CharacterRecipes characterId={entry.character.id} profession={entry.profession} />
        </div>
      )}
    </li>
  )
}
