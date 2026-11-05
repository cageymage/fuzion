import { classColor } from '../../lib/classColor'
import { wowheadRecipeSearchUrl } from '../../lib/wowhead'
import type { Recipe } from '../../types/professions'
import styles from './Professions.module.css'

interface CrafterResultsProps {
  recipes: Recipe[]
}

export function CrafterResults({ recipes }: CrafterResultsProps) {
  return (
    <section className={styles.group} aria-labelledby="crafters-heading">
      <h2 id="crafters-heading" className={styles.groupTitle}>
        Who can craft it
      </h2>
      <ul className={styles.entries}>
        {recipes.map((recipe) => (
          <li key={recipe.id} className={styles.entry}>
            <span>
              <a href={wowheadRecipeSearchUrl(recipe.name)} target="_blank" rel="noreferrer">
                {recipe.name}
              </a>{' '}
              <span className={styles.recipeProfession}>{recipe.profession}</span>
            </span>
            <span className={styles.crafters}>
              {recipe.characters.map((character) => (
                <span key={character.id} style={{ color: classColor(character.class) }}>
                  {character.name} {character.secondaryName}
                </span>
              ))}
            </span>
          </li>
        ))}
      </ul>
    </section>
  )
}
