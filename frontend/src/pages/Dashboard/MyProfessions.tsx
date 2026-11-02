import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, type FormEvent } from 'react'
import { addMyProfession, deleteMyProfession, updateMyProfession } from '../../api/professions'
import { primaryProfessions, secondaryProfessions } from '../../lib/professions'
import type { MyCharacter, MyCharacterProfession } from '../../types/roster'
import styles from './Dashboard.module.css'
import { myCharactersQueryKey } from './myCharactersQueryKey'

const allProfessions: readonly string[] = [...primaryProfessions, ...secondaryProfessions]

function useRefreshAfterEdit() {
  const queryClient = useQueryClient()
  return () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: myCharactersQueryKey }),
      queryClient.invalidateQueries({ queryKey: ['roster'] }),
      queryClient.invalidateQueries({ queryKey: ['professions'] }),
    ])
}

function errorMessage(error: Error | null): string | null {
  return error ? error.message : null
}

function ProfessionRow({ profession, fullName }: { profession: MyCharacterProfession; fullName: string }) {
  const refresh = useRefreshAfterEdit()
  const [skillLevel, setSkillLevel] = useState(String(profession.skillLevel))
  const save = useMutation({
    mutationFn: () => updateMyProfession(profession.id, { skillLevel: Number(skillLevel) }),
    onSuccess: refresh,
  })
  const remove = useMutation({ mutationFn: () => deleteMyProfession(profession.id), onSuccess: refresh })
  const changed = skillLevel !== String(profession.skillLevel)
  const busy = save.isPending || remove.isPending

  function submit(event: FormEvent) {
    event.preventDefault()
    save.mutate()
  }

  return (
    <li>
      <form className={styles.professionRow} onSubmit={submit}>
        <span className={styles.professionName}>{profession.profession}</span>
        <input
          className={styles.skillInput}
          type="number"
          min={0}
          required
          value={skillLevel}
          onChange={(event) => setSkillLevel(event.target.value)}
          aria-label={`${profession.profession} skill for ${fullName}`}
        />
        <button type="submit" className={styles.button} disabled={!changed || busy}>
          Save
        </button>
        <button
          type="button"
          className={styles.button}
          onClick={() => remove.mutate()}
          disabled={busy}
          aria-label={`Remove ${profession.profession} from ${fullName}`}
        >
          Remove
        </button>
      </form>
      {(save.isError || remove.isError) && (
        <p role="alert" className={styles.error}>
          {errorMessage(save.error ?? remove.error)}
        </p>
      )}
    </li>
  )
}

function AddProfession({ character, fullName }: { character: MyCharacter; fullName: string }) {
  const refresh = useRefreshAfterEdit()
  const known = new Set(character.professions.map(({ profession }) => profession))
  const available = allProfessions.filter((name) => !known.has(name))
  const [profession, setProfession] = useState('')
  const [skillLevel, setSkillLevel] = useState('1')
  const add = useMutation({
    mutationFn: () => addMyProfession(character.id, { profession, skillLevel: Number(skillLevel) }),
    onSuccess: async () => {
      setProfession('')
      setSkillLevel('1')
      await refresh()
    },
  })

  if (available.length === 0) return null

  function submit(event: FormEvent) {
    event.preventDefault()
    add.mutate()
  }

  return (
    <form className={styles.professionRow} onSubmit={submit} aria-label={`Add a profession to ${fullName}`}>
      <select
        className={styles.select}
        value={profession}
        onChange={(event) => setProfession(event.target.value)}
        required
        aria-label={`Profession to add to ${fullName}`}
      >
        <option value="">Add a profession…</option>
        {available.map((name) => (
          <option key={name} value={name}>
            {name}
          </option>
        ))}
      </select>
      <input
        className={styles.skillInput}
        type="number"
        min={0}
        required
        value={skillLevel}
        onChange={(event) => setSkillLevel(event.target.value)}
        aria-label={`Skill for the new profession on ${fullName}`}
      />
      <button type="submit" className={styles.button} disabled={add.isPending}>
        Add
      </button>
      {add.isError && (
        <p role="alert" className={styles.error}>
          {errorMessage(add.error)}
        </p>
      )}
    </form>
  )
}

export function MyProfessions({ character, fullName }: { character: MyCharacter; fullName: string }) {
  return (
    <div className={styles.professions}>
      {character.professions.length > 0 ? (
        <ul className={styles.professionList} aria-label={`Professions of ${fullName}`}>
          {/* Keyed on the skill level too, so the input resets to the saved value after a refetch. */}
          {character.professions.map((profession) => (
            <ProfessionRow key={`${profession.id}-${profession.skillLevel}`} profession={profession} fullName={fullName} />
          ))}
        </ul>
      ) : (
        <p className={styles.hint}>No professions yet.</p>
      )}
      <AddProfession character={character} fullName={fullName} />
    </div>
  )
}
