import { useState } from 'react'
import { Link } from 'react-router-dom'
import { loginUrl } from '../../api/auth'
import { Dialog } from '../../components/Dialog/Dialog'
import { useCurrentUser } from '../../hooks/useCurrentUser'
import { classColor } from '../../lib/classColor'
import type { Character } from '../../types/roster'
import { roleLabel, specLabel } from '../Roster/rosterLabels'
import { useRosterData } from '../Roster/useRosterData'
import { CharacterDialog } from './CharacterDialog'
import styles from './OfficerRoster.module.css'
import { useOfficerRoster } from './useOfficerRoster'

type DialogState =
  | { kind: 'add' }
  | { kind: 'edit'; character: Character }
  | { kind: 'delete'; character: Character }

interface DeleteDialogProps {
  character: Character
  onClose: () => void
}

function DeleteDialog({ character, onClose }: DeleteDialogProps) {
  const { remove } = useOfficerRoster()

  return (
    <Dialog title={`Delete ${character.name}`} onClose={onClose}>
      <p>
        Delete {character.name} {character.secondaryName} from the roster? This cannot be undone.
      </p>
      {remove.isError && (
        <p role="alert" className={styles.error}>
          {remove.error.message}
        </p>
      )}
      <div className={styles.actions}>
        <button
          type="button"
          className={`${styles.button} ${styles.dangerButton}`}
          disabled={remove.isPending}
          onClick={() => remove.mutate(character.id, { onSuccess: onClose })}
        >
          Delete character
        </button>
        <button type="button" className={styles.button} onClick={onClose}>
          Cancel
        </button>
      </div>
    </Dialog>
  )
}

function OfficerTools() {
  const roster = useRosterData()
  const [dialog, setDialog] = useState<DialogState | null>(null)
  const closeDialog = () => setDialog(null)

  return (
    <>
      <div className={styles.toolbar}>
        <button type="button" className={styles.button} onClick={() => setDialog({ kind: 'add' })}>
          Add character
        </button>
      </div>

      {roster.isPending && <p className={styles.noticeText}>Loading the roster…</p>}
      {roster.isError && <p className={styles.noticeText}>The roster could not be loaded.</p>}
      {roster.isSuccess && roster.data.length === 0 && <p className={styles.noticeText}>No characters yet.</p>}

      {roster.isSuccess && roster.data.length > 0 && (
        <div className={styles.tableScroll} role="region" aria-label="Officer roster table" tabIndex={0}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th scope="col">Name</th>
                <th scope="col">Secondary name</th>
                <th scope="col">Class</th>
                <th scope="col">Spec</th>
                <th scope="col">Role</th>
                <th scope="col">Main/Alt</th>
                <th scope="col">Raid Team</th>
                <th scope="col">Actions</th>
              </tr>
            </thead>
            <tbody>
              {roster.data.map((character) => (
                <tr key={character.id}>
                  <td>{character.name}</td>
                  <td>{character.secondaryName}</td>
                  <td style={{ color: classColor(character.class) }}>{character.class}</td>
                  <td>{specLabel(character)}</td>
                  <td>{roleLabel(character)}</td>
                  <td>{character.isMain ? 'Main' : 'Alt'}</td>
                  <td>{character.raidTeam ?? '—'}</td>
                  <td>
                    <div className={styles.rowActions}>
                      <button
                        type="button"
                        className={styles.smallButton}
                        aria-label={`Edit ${character.name}`}
                        onClick={() => setDialog({ kind: 'edit', character })}
                      >
                        Edit
                      </button>
                      <button
                        type="button"
                        className={styles.smallButton}
                        aria-label={`Delete ${character.name}`}
                        onClick={() => setDialog({ kind: 'delete', character })}
                      >
                        Delete
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {dialog?.kind === 'add' && <CharacterDialog onClose={closeDialog} />}
      {dialog?.kind === 'edit' && <CharacterDialog character={dialog.character} onClose={closeDialog} />}
      {dialog?.kind === 'delete' && <DeleteDialog character={dialog.character} onClose={closeDialog} />}
    </>
  )
}

export function OfficerRoster() {
  const currentUser = useCurrentUser()

  return (
    <section className={styles.page}>
      <h1 className={styles.title}>Manage Roster</h1>
      <Link to="/roster" className={styles.backLink}>
        Back to Roster
      </Link>

      {currentUser.isPending && <p className={styles.noticeText}>Checking your access…</p>}
      {currentUser.isError && <p className={styles.noticeText}>Your access could not be checked.</p>}
      {currentUser.isSuccess && !currentUser.data?.isOfficer && (
        <div className={styles.noticeText}>
          <p>Only officers can manage the roster.</p>
          {!currentUser.data && (
            // A plain anchor, not a router Link: the server has to answer this one so it can redirect to Discord.
            <a href={loginUrl}>Log in with Discord</a>
          )}
        </div>
      )}
      {currentUser.data?.isOfficer && <OfficerTools />}
    </section>
  )
}
