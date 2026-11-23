import { Link } from 'react-router-dom'
import type { ProfessionEntry } from '../../types/professions'
import styles from './Professions.module.css'

interface SpecializationFilterProps {
  specializations: readonly string[]
  entries: ProfessionEntry[]
  selected: string | null
  onSelect: (specialization: string | null) => void
}

export function SpecializationFilter({ specializations, entries, selected, onSelect }: SpecializationFilterProps) {
  return (
    <div className={styles.specializationFilter}>
      <div role="group" aria-label="Filter by specialization" className={styles.specializationChips}>
        <button
          type="button"
          className={styles.specializationChip}
          aria-pressed={selected === null}
          onClick={() => onSelect(null)}
        >
          All ({entries.length})
        </button>
        {specializations.map((specialization) => {
          const count = entries.filter((entry) => entry.specialization === specialization).length
          return (
            <button
              key={specialization}
              type="button"
              className={styles.specializationChip}
              aria-pressed={selected === specialization}
              disabled={count === 0}
              onClick={() => onSelect(specialization)}
            >
              {specialization} ({count})
            </button>
          )
        })}
      </div>
      <p className={styles.specializationHint}>
        Set your specialization on your <Link to="/dashboard">Dashboard</Link>.
      </p>
    </div>
  )
}
