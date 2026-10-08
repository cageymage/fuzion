import { Dialog } from '../../components/Dialog/Dialog'
import type { Character } from '../../types/roster'
import styles from './Roster.module.css'
import { useRosterManagement } from './useRosterManagement'

interface DeleteCharacterDialogProps {
  character: Character
  onClose: () => void
}

export function DeleteCharacterDialog({ character, onClose }: DeleteCharacterDialogProps) {
  const { remove } = useRosterManagement()

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
