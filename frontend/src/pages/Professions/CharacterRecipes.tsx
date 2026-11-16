import { useState } from 'react'
import { wowheadRecipeSearchUrl } from '../../lib/wowhead'
import styles from './Professions.module.css'
import { useRecipes } from './useRecipes'

const collapsedCount = 8

interface CharacterRecipesProps {
  characterId: string
  profession: string
}

export function CharacterRecipes({ characterId, profession }: CharacterRecipesProps) {
  const recipes = useRecipes({ query: '', characterId })
  const [showAll, setShowAll] = useState(false)

  if (recipes.isError) {
    return <p className={styles.noticeText}>Recipes could not be loaded.</p>
  }
  if (!recipes.isSuccess) {
    return <p className={styles.noticeText}>Loading recipes…</p>
  }

  const known = recipes.data.filter((recipe) => recipe.profession === profession)
  if (known.length === 0) {
    return <p className={styles.noticeText}>No recipes recorded yet.</p>
  }

  const shown = showAll ? known : known.slice(0, collapsedCount)
  return (
    <div className={styles.recipeList}>
      {shown.map((recipe) => (
        <a
          key={recipe.id}
          className={styles.recipeChip}
          href={wowheadRecipeSearchUrl(recipe.name)}
          target="_blank"
          rel="noreferrer"
        >
          {recipe.name}
        </a>
      ))}
      {known.length > collapsedCount && !showAll && (
        <button type="button" className={styles.showAll} onClick={() => setShowAll(true)}>
          Show all {known.length}
        </button>
      )}
    </div>
  )
}
