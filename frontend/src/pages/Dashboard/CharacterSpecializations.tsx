import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, type FormEvent } from 'react'
import { setSpecialization } from '../../api/professions'
import { specializationsOf } from '../../lib/professions'
import type { Character, CharacterProfession } from '../../types/roster'
import styles from './Dashboard.module.css'
import { myCharactersQueryKey } from './myCharactersQueryKey'

interface SpecializationFormProps {
  profession: CharacterProfession
  fullName: string
}

function SpecializationForm({ profession, fullName }: SpecializationFormProps) {
  const queryClient = useQueryClient()
  const initial = profession.specialization ?? ''
  const [value, setValue] = useState(initial)
  const save = useMutation({
    mutationFn: () => setSpecialization(profession.id, value === '' ? null : value),
    onSuccess: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: myCharactersQueryKey }),
        queryClient.invalidateQueries({ queryKey: ['roster'] }),
        queryClient.invalidateQueries({ queryKey: ['professions'] }),
      ]),
  })

  function submit(event: FormEvent) {
    event.preventDefault()
    save.mutate()
  }

  return (
    <form
      className={styles.specsForm}
      onSubmit={submit}
      aria-label={`${profession.profession} specialization for ${fullName}`}
    >
      <label className={styles.specField}>
        <span>{profession.profession} specialization</span>
        <select className={styles.select} value={value} onChange={(event) => setValue(event.target.value)}>
          <option value="">None</option>
          {specializationsOf(profession.profession).map((specialization) => (
            <option key={specialization} value={specialization}>
              {specialization}
            </option>
          ))}
        </select>
      </label>
      <div className={styles.formActions}>
        <button type="submit" className={styles.button} disabled={save.isPending || value === initial}>
          Save
        </button>
      </div>
      {save.isError && (
        <p role="alert" className={styles.error}>
          {save.error.message}
        </p>
      )}
    </form>
  )
}

interface CharacterSpecializationsProps {
  character: Character
  fullName: string
}

export function CharacterSpecializations({ character, fullName }: CharacterSpecializationsProps) {
  return character.professions
    .filter(({ profession }) => specializationsOf(profession).length > 0)
    .map((profession) =>
      profession.specializationSource === 'blizzard' ? (
        <p key={profession.id} className={styles.hint}>
          {profession.profession} specialization: {profession.specialization} (from Battle.net)
        </p>
      ) : (
        <SpecializationForm
          key={`${profession.id}-${profession.specialization}`}
          profession={profession}
          fullName={fullName}
        />
      ),
    )
}
