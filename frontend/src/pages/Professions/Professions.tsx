import { useState } from 'react'
import { classColor } from '../../lib/classColor'
import type { ProfessionEntry } from '../../types/professions'
import { CharacterRecipes } from './CharacterRecipes'
import { CrafterResults } from './CrafterResults'
import { crafterKey, groupProfessions } from './groupProfessions'
import { professionPanelId, ProfessionTabs, professionTabId } from './ProfessionTabs'
import { resolveSelectedTab, tabRows } from './tabSelection'
import styles from './Professions.module.css'
import { useProfessionsData } from './useProfessionsData'
import { useRecipes } from './useRecipes'

const minRecipeSearchLength = 2
const tabRowLabels = ['Primary professions', 'Secondary professions']

export function Professions() {
  const professions = useProfessionsData()
  const [query, setQuery] = useState('')

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

  const counts = Object.fromEntries(groups.map((group) => [group.profession, group.entries.length]))
  const [selectedTab, setSelectedTab] = useState<string | null>(null)
  const activeTab =
    waitingForRecipes && selectedTab !== null ? selectedTab : resolveSelectedTab(selectedTab, counts)
  if (!waitingForRecipes && professions.isSuccess && activeTab !== selectedTab) {
    setSelectedTab(activeTab)
  }
  const activeGroup = groups.find((group) => group.profession === activeTab)

  return (
    <section className={styles.page}>
      <h1 className={styles.title}>Professions</h1>

      <label className={styles.searchField} htmlFor="professions-search">
        Search
        <input
          id="professions-search"
          type="search"
          className={styles.search}
          placeholder="Character or recipe"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
        />
      </label>

      {professions.isPending && <p className={styles.noticeText}>Loading professions…</p>}
      {professions.isError && <p className={styles.noticeText}>Professions could not be loaded.</p>}

      {searchingRecipes && recipes.isError && <p className={styles.noticeText}>Recipes could not be loaded.</p>}
      {matchedRecipes.length > 0 && <CrafterResults recipes={matchedRecipes} />}

      {professions.isSuccess && professions.data.length === 0 && (
        <p className={styles.noticeText}>Nobody has recorded a profession yet.</p>
      )}
      {professions.isSuccess &&
        professions.data.length > 0 &&
        activeTab === null &&
        !waitingForRecipes &&
        matchedRecipes.length === 0 && <p className={styles.noticeText}>No matches for "{trimmedQuery}".</p>}

      {professions.isSuccess && professions.data.length > 0 && (
        <ProfessionTabs
          rows={tabRows}
          rowLabels={tabRowLabels}
          counts={counts}
          selected={activeTab}
          onSelect={setSelectedTab}
        />
      )}
      {activeTab !== null && activeGroup && (
        <div role="tabpanel" id={professionPanelId} aria-labelledby={professionTabId(activeTab)}>
          <ul className={styles.entries}>
            {activeGroup.entries.map((entry) => (
              <CharacterEntry key={entry.id} entry={entry} />
            ))}
          </ul>
        </div>
      )}
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
          <span style={{ color }}>
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
