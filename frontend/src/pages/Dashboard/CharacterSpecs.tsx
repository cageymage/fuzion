import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, type FormEvent } from 'react'
import { updateMyCharacter } from '../../api/roster'
import { classSpecs, type WowClass } from '../../lib/wowClasses'
import type { Character, Role, UpdateMyCharacterRequest } from '../../types/roster'
import { roleLabels } from '../Roster/rosterLabels'
import styles from './Dashboard.module.css'
import { myCharactersQueryKey } from './myCharactersQueryKey'

const roles = Object.keys(roleLabels) as Role[]

function describe(spec: string | null, role: Role | null): string {
  if (spec && role) return `${spec} (${roleLabels[role]})`
  if (spec) return spec
  if (role) return roleLabels[role]
  return 'Not set'
}

interface SpecValues {
  spec: string
  role: Role | ''
  spec2: string
  role2: Role | ''
}

function valuesFor(character: Character): SpecValues {
  return {
    spec: character.spec ?? '',
    role: character.role ?? '',
    spec2: character.spec2 ?? '',
    role2: character.role2 ?? '',
  }
}

function changedFields(initial: SpecValues, current: SpecValues): UpdateMyCharacterRequest {
  const request: UpdateMyCharacterRequest = {}
  if (current.spec !== initial.spec) request.spec = current.spec
  if (current.role !== initial.role) request.role = current.role
  if (current.spec2 !== initial.spec2 || current.role2 !== initial.role2) {
    request.spec2 = current.spec2
    request.role2 = current.role2
  }
  return request
}

interface SpecsFormProps {
  character: Character
  fullName: string
  onDone: () => void
}

function SpecsForm({ character, fullName, onDone }: SpecsFormProps) {
  const queryClient = useQueryClient()
  const initial = valuesFor(character)
  const [values, setValues] = useState(initial)
  const specs = classSpecs[character.class as WowClass] ?? []
  const save = useMutation({
    mutationFn: () => updateMyCharacter(character.id, changedFields(initial, values)),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: myCharactersQueryKey }),
        queryClient.invalidateQueries({ queryKey: ['roster'] }),
      ])
      onDone()
    },
  })

  function change(changes: Partial<SpecValues>) {
    setValues({ ...values, ...changes })
  }

  function submit(event: FormEvent) {
    event.preventDefault()
    save.mutate()
  }

  return (
    <form className={styles.specsForm} onSubmit={submit} aria-label={`Specs for ${fullName}`}>
      <label className={styles.specField}>
        <span>Primary spec</span>
        <select className={styles.select} value={values.spec} onChange={(event) => change({ spec: event.target.value })}>
          <option value="">Not set</option>
          {specs.map((spec) => (
            <option key={spec} value={spec}>
              {spec}
            </option>
          ))}
        </select>
      </label>
      <label className={styles.specField}>
        <span>Primary role</span>
        <select
          className={styles.select}
          value={values.role}
          onChange={(event) => change({ role: event.target.value as Role | '' })}
        >
          <option value="">Not set</option>
          {roles.map((role) => (
            <option key={role} value={role}>
              {roleLabels[role]}
            </option>
          ))}
        </select>
      </label>
      <label className={styles.specField}>
        <span>Secondary spec</span>
        <select
          className={styles.select}
          value={values.spec2}
          onChange={(event) => {
            const spec2 = event.target.value
            change({ spec2, role2: spec2 === '' ? '' : values.role2 || values.role || roles[0] })
          }}
        >
          <option value="">None</option>
          {specs
            .filter((spec) => spec !== values.spec)
            .map((spec) => (
              <option key={spec} value={spec}>
                {spec}
              </option>
            ))}
        </select>
      </label>
      {values.spec2 !== '' && (
        <label className={styles.specField}>
          <span>Secondary role</span>
          <select
            className={styles.select}
            value={values.role2}
            onChange={(event) => change({ role2: event.target.value as Role })}
          >
            {roles.map((role) => (
              <option key={role} value={role}>
                {roleLabels[role]}
              </option>
            ))}
          </select>
        </label>
      )}
      <div className={styles.formActions}>
        <button type="submit" className={styles.button} disabled={save.isPending}>
          Save
        </button>
        <button type="button" className={styles.button} onClick={onDone} disabled={save.isPending}>
          Cancel
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

interface CharacterSpecsProps {
  character: Character
  fullName: string
}

export function CharacterSpecs({ character, fullName }: CharacterSpecsProps) {
  const [editing, setEditing] = useState(false)

  if (editing) {
    return <SpecsForm character={character} fullName={fullName} onDone={() => setEditing(false)} />
  }

  return (
    <div className={styles.specs}>
      <dl className={styles.specList}>
        <dt>Primary spec</dt>
        <dd>{describe(character.spec || null, character.role)}</dd>
        {character.spec2 && (
          <>
            <dt>Secondary spec</dt>
            <dd>{describe(character.spec2, character.role2)}</dd>
          </>
        )}
      </dl>
      <button
        type="button"
        className={styles.button}
        onClick={() => setEditing(true)}
        aria-label={`Edit specs for ${fullName}`}
      >
        Edit specs
      </button>
    </div>
  )
}
