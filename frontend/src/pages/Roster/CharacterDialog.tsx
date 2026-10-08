import { useState, type FormEvent } from 'react'
import { Dialog } from '../../components/Dialog/Dialog'
import { primaryProfessions, secondaryProfessions } from '../../lib/professions'
import { maxLevel, races } from '../../lib/races'
import { classSpecs, wowClasses, type WowClass } from '../../lib/wowClasses'
import type { Character, Role } from '../../types/roster'
import { roleLabels } from './rosterLabels'
import {
  buildCreateRequest,
  buildUpdateRequest,
  emptyCharacterForm,
  formValuesFor,
  type CharacterFormValues,
} from './characterForm'
import styles from './Roster.module.css'
import { useRosterManagement } from './useRosterManagement'

const roles = Object.keys(roleLabels) as Role[]

function specsFor(className: string): readonly string[] {
  return classSpecs[className as WowClass] ?? []
}

interface CharacterDialogProps {
  character?: Character
  onClose: () => void
}

export function CharacterDialog({ character, onClose }: CharacterDialogProps) {
  const { create, update } = useRosterManagement()
  const initial = character ? formValuesFor(character) : emptyCharacterForm
  const [values, setValues] = useState<CharacterFormValues>(initial)
  const mutation = character ? update : create
  const specs = specsFor(values.class)

  function change(changes: Partial<CharacterFormValues>) {
    setValues({ ...values, ...changes })
  }

  function changeSecondSpec(spec2: string) {
    change({ spec2, role2: spec2 === '' ? '' : values.role2 || values.role })
  }

  function toggleSecondaryProfession(profession: string, checked: boolean) {
    change({
      secondaryProfessions: checked
        ? [...values.secondaryProfessions, profession]
        : values.secondaryProfessions.filter((name) => name !== profession),
    })
  }

  function submit(event: FormEvent) {
    event.preventDefault()
    if (!character) {
      create.mutate(buildCreateRequest(values), { onSuccess: onClose })
      return
    }
    const request = buildUpdateRequest(initial, values)
    if (Object.keys(request).length === 0) {
      onClose()
      return
    }
    update.mutate({ id: character.id, request }, { onSuccess: onClose })
  }

  return (
    <Dialog title={character ? `Edit ${character.name}` : 'Add character'} onClose={onClose}>
      <form className={styles.form} onSubmit={submit}>
        <label className={styles.field}>
          <span>Name</span>
          <input required value={values.name} onChange={(event) => change({ name: event.target.value })} />
        </label>
        <label className={styles.field}>
          <span>Secondary name</span>
          <input
            required
            value={values.secondaryName}
            onChange={(event) => change({ secondaryName: event.target.value })}
          />
        </label>
        <label className={styles.field}>
          <span>Realm</span>
          <input
            placeholder="Emberreach"
            value={values.realm}
            onChange={(event) => change({ realm: event.target.value })}
          />
        </label>
        <label className={styles.field}>
          <span>Race</span>
          <select value={values.race} onChange={(event) => change({ race: event.target.value })}>
            <option value="">Not set</option>
            {races.map((race) => (
              <option key={race} value={race}>
                {race}
              </option>
            ))}
          </select>
        </label>
        <label className={styles.field}>
          <span>Level</span>
          <input
            type="number"
            min={1}
            max={maxLevel}
            value={values.level}
            onChange={(event) => change({ level: event.target.value })}
          />
        </label>
        <label className={styles.field}>
          <span>Class</span>
          <select
            required
            value={values.class}
            onChange={(event) => change({ class: event.target.value, spec: '', spec2: '', role2: '' })}
          >
            <option value="">Select class</option>
            {wowClasses.map((wowClass) => (
              <option key={wowClass} value={wowClass}>
                {wowClass}
              </option>
            ))}
          </select>
        </label>
        <label className={styles.field}>
          <span>Spec</span>
          <select
            required
            disabled={specs.length === 0}
            value={values.spec}
            onChange={(event) => change({ spec: event.target.value })}
          >
            <option value="">Select spec</option>
            {specs.map((spec) => (
              <option key={spec} value={spec}>
                {spec}
              </option>
            ))}
          </select>
        </label>
        <label className={styles.field}>
          <span>Role</span>
          <select value={values.role} onChange={(event) => change({ role: event.target.value as Role })}>
            {roles.map((role) => (
              <option key={role} value={role}>
                {roleLabels[role]}
              </option>
            ))}
          </select>
        </label>
        <label className={styles.field}>
          <span>Second spec</span>
          <select
            disabled={specs.length === 0}
            value={values.spec2}
            onChange={(event) => changeSecondSpec(event.target.value)}
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
          <label className={styles.field}>
            <span>Second role</span>
            <select
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
        <label className={styles.field}>
          <span>Raid team</span>
          <input value={values.raidTeam} onChange={(event) => change({ raidTeam: event.target.value })} />
        </label>
        <label className={styles.field}>
          <span>Primary profession 1</span>
          <select
            value={values.primaryProfession1}
            onChange={(event) => change({ primaryProfession1: event.target.value })}
          >
            <option value="">None</option>
            {primaryProfessions
              .filter((profession) => profession !== values.primaryProfession2)
              .map((profession) => (
                <option key={profession} value={profession}>
                  {profession}
                </option>
              ))}
          </select>
        </label>
        <label className={styles.field}>
          <span>Primary profession 2</span>
          <select
            value={values.primaryProfession2}
            onChange={(event) => change({ primaryProfession2: event.target.value })}
          >
            <option value="">None</option>
            {primaryProfessions
              .filter((profession) => profession !== values.primaryProfession1)
              .map((profession) => (
                <option key={profession} value={profession}>
                  {profession}
                </option>
              ))}
          </select>
        </label>
        <fieldset className={styles.checkboxGroup}>
          <legend>Secondary professions</legend>
          {secondaryProfessions.map((profession) => (
            <label key={profession} className={styles.toggle}>
              <input
                type="checkbox"
                checked={values.secondaryProfessions.includes(profession)}
                onChange={(event) => toggleSecondaryProfession(profession, event.target.checked)}
              />
              <span>{profession}</span>
            </label>
          ))}
        </fieldset>
        <label className={styles.toggle}>
          <input
            type="checkbox"
            checked={values.isMain}
            onChange={(event) => change({ isMain: event.target.checked })}
          />
          <span>Main character</span>
        </label>

        {mutation.isError && (
          <p role="alert" className={styles.error}>
            {mutation.error.message}
          </p>
        )}
        <div className={styles.actions}>
          <button type="submit" className={styles.button} disabled={mutation.isPending}>
            Save character
          </button>
          <button type="button" className={styles.button} onClick={onClose}>
            Cancel
          </button>
        </div>
      </form>
    </Dialog>
  )
}
