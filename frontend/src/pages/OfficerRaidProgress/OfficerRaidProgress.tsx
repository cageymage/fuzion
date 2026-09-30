import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { loginUrl } from '../../api/auth'
import { useCurrentUser } from '../../hooks/useCurrentUser'
import type { RaidTier } from '../../types/raidProgress'
import { useRaidTiersData } from '../RaidProgress/useRaidProgressData'
import styles from './OfficerRaidProgress.module.css'
import { useOfficerRaidProgress } from './useOfficerRaidProgress'

function swapped<T>(items: T[], index: number, offset: -1 | 1): T[] {
  const target = index + offset
  const reordered = [...items]
  ;[reordered[index], reordered[target]] = [reordered[target], reordered[index]]
  return reordered
}

interface CreateTierFormProps {
  nextSortOrder: number
}

function CreateTierForm({ nextSortOrder }: CreateTierFormProps) {
  const { createTier } = useOfficerRaidProgress()
  const [name, setName] = useState('')
  const [bossNames, setBossNames] = useState([''])

  function renameBoss(index: number, bossName: string) {
    setBossNames(bossNames.map((existing, i) => (i === index ? bossName : existing)))
  }

  function moveBoss(index: number, offset: -1 | 1) {
    setBossNames(swapped(bossNames, index, offset))
  }

  function submit(event: FormEvent) {
    event.preventDefault()
    createTier.mutate(
      {
        name,
        sortOrder: nextSortOrder,
        bosses: bossNames.map((bossName) => bossName.trim()).filter((bossName) => bossName !== ''),
      },
      {
        onSuccess: () => {
          setName('')
          setBossNames([''])
        },
      },
    )
  }

  return (
    <form className={`card ${styles.card}`} onSubmit={submit}>
      <h2 className={styles.sectionTitle}>Create a raid</h2>
      <label className={styles.field}>
        <span>Raid name</span>
        <input value={name} onChange={(event) => setName(event.target.value)} />
      </label>

      <ol className={styles.bossRows}>
        {bossNames.map((bossName, index) => (
          <li key={index} className={styles.bossRow}>
            <label className={styles.field}>
              <span>Boss {index + 1} name</span>
              <input value={bossName} onChange={(event) => renameBoss(index, event.target.value)} />
            </label>
            <button
              type="button"
              className={styles.smallButton}
              aria-label={`Move boss ${index + 1} up`}
              disabled={index === 0}
              onClick={() => moveBoss(index, -1)}
            >
              ↑
            </button>
            <button
              type="button"
              className={styles.smallButton}
              aria-label={`Move boss ${index + 1} down`}
              disabled={index === bossNames.length - 1}
              onClick={() => moveBoss(index, 1)}
            >
              ↓
            </button>
            <button
              type="button"
              className={styles.smallButton}
              aria-label={`Remove boss ${index + 1}`}
              disabled={bossNames.length === 1}
              onClick={() => setBossNames(bossNames.filter((_, i) => i !== index))}
            >
              ✕
            </button>
          </li>
        ))}
      </ol>

      <div className={styles.actions}>
        <button type="button" className={styles.button} onClick={() => setBossNames([...bossNames, ''])}>
          Add boss
        </button>
        <button type="submit" className={styles.button} disabled={createTier.isPending}>
          Create raid
        </button>
      </div>
      {createTier.isError && (
        <p role="alert" className={styles.error}>
          {createTier.error.message}
        </p>
      )}
    </form>
  )
}

interface ConfirmDeleteButtonProps {
  target: string
  children: string
  disabled?: boolean
  onConfirm: () => void
}

function ConfirmDeleteButton({ target, children, disabled = false, onConfirm }: ConfirmDeleteButtonProps) {
  const [confirming, setConfirming] = useState(false)

  if (!confirming) {
    return (
      <button
        type="button"
        className={styles.smallButton}
        aria-label={`Delete ${target}`}
        disabled={disabled}
        onClick={() => setConfirming(true)}
      >
        {children}
      </button>
    )
  }

  return (
    <span className={styles.confirm}>
      <button
        type="button"
        className={`${styles.smallButton} ${styles.dangerButton}`}
        aria-label={`Confirm delete ${target}`}
        onClick={() => {
          setConfirming(false)
          onConfirm()
        }}
      >
        Delete
      </button>
      <button
        type="button"
        className={styles.smallButton}
        aria-label={`Cancel deleting ${target}`}
        onClick={() => setConfirming(false)}
      >
        Cancel
      </button>
    </span>
  )
}

interface RenameControlProps {
  target: string
  isPending: boolean
  onSave: (name: string, onSaved: () => void) => void
}

function RenameControl({ target, isPending, onSave }: RenameControlProps) {
  const [draft, setDraft] = useState<string | null>(null)

  if (draft === null) {
    return (
      <button
        type="button"
        className={styles.smallButton}
        aria-label={`Rename ${target}`}
        onClick={() => setDraft(target)}
      >
        Rename
      </button>
    )
  }

  function submit(event: FormEvent) {
    event.preventDefault()
    onSave(draft ?? '', () => setDraft(null))
  }

  return (
    <form className={styles.inlineForm} onSubmit={submit}>
      <input
        className={styles.inlineInput}
        aria-label={`New name for ${target}`}
        value={draft}
        onChange={(event) => setDraft(event.target.value)}
      />
      <button
        type="submit"
        className={styles.smallButton}
        aria-label={`Save name for ${target}`}
        disabled={isPending}
      >
        Save
      </button>
      <button
        type="button"
        className={styles.smallButton}
        aria-label={`Cancel renaming ${target}`}
        onClick={() => setDraft(null)}
      >
        Cancel
      </button>
    </form>
  )
}

interface MoveButtonsProps {
  target: string
  isFirst: boolean
  isLast: boolean
  disabled: boolean
  onMove: (offset: -1 | 1) => void
}

function MoveButtons({ target, isFirst, isLast, disabled, onMove }: MoveButtonsProps) {
  return (
    <>
      <button
        type="button"
        className={styles.smallButton}
        aria-label={`Move ${target} up`}
        disabled={disabled || isFirst}
        onClick={() => onMove(-1)}
      >
        ↑
      </button>
      <button
        type="button"
        className={styles.smallButton}
        aria-label={`Move ${target} down`}
        disabled={disabled || isLast}
        onClick={() => onMove(1)}
      >
        ↓
      </button>
    </>
  )
}

interface AddBossFormProps {
  tier: RaidTier
}

function AddBossForm({ tier }: AddBossFormProps) {
  const { addBoss } = useOfficerRaidProgress()
  const [name, setName] = useState('')

  function submit(event: FormEvent) {
    event.preventDefault()
    addBoss.mutate({ tierId: tier.id, name }, { onSuccess: () => setName('') })
  }

  return (
    <form className={styles.inlineForm} onSubmit={submit}>
      <input
        className={styles.inlineInput}
        aria-label={`New boss for ${tier.name}`}
        placeholder="New boss name"
        value={name}
        onChange={(event) => setName(event.target.value)}
      />
      <button
        type="submit"
        className={styles.smallButton}
        aria-label={`Add boss to ${tier.name}`}
        disabled={addBoss.isPending}
      >
        Add boss
      </button>
      {addBoss.isError && (
        <p role="alert" className={styles.error}>
          {addBoss.error.message}
        </p>
      )}
    </form>
  )
}

interface TierManagerProps {
  tier: RaidTier
  tierIds: string[]
  index: number
}

function TierManager({ tier, tierIds, index }: TierManagerProps) {
  const {
    setTierCurrent,
    renameTier,
    reorderTiers,
    setBossKilled,
    renameBoss,
    reorderBosses,
    deleteTier,
    deleteBoss,
  } = useOfficerRaidProgress()
  const bossIds = tier.bosses.map((boss) => boss.id)
  const errors = [
    setTierCurrent,
    renameTier,
    reorderTiers,
    setBossKilled,
    renameBoss,
    reorderBosses,
    deleteTier,
    deleteBoss,
  ].flatMap((mutation) => (mutation.error ? [mutation.error.message] : []))

  return (
    <div className={`card ${styles.card}`}>
      <div className={styles.tierHeader}>
        <h3 className={styles.tierName}>{tier.name}</h3>
        <div className={styles.controls}>
          <RenameControl
            target={tier.name}
            isPending={renameTier.isPending}
            onSave={(name, onSaved) => renameTier.mutate({ id: tier.id, name }, { onSuccess: onSaved })}
          />
          <MoveButtons
            target={tier.name}
            isFirst={index === 0}
            isLast={index === tierIds.length - 1}
            disabled={reorderTiers.isPending}
            onMove={(offset) => reorderTiers.mutate(swapped(tierIds, index, offset))}
          />
          <ConfirmDeleteButton
            target={tier.name}
            disabled={tier.isCurrent}
            onConfirm={() => deleteTier.mutate(tier.id)}
          >
            Delete raid
          </ConfirmDeleteButton>
        </div>
      </div>
      {tier.isCurrent && <p className={styles.noticeText}>Uncheck "Current raid" before deleting this raid.</p>}
      <label className={styles.toggle}>
        <input
          type="checkbox"
          aria-label={`${tier.name} is current`}
          checked={tier.isCurrent}
          onChange={(event) => setTierCurrent.mutate({ id: tier.id, isCurrent: event.target.checked })}
        />
        <span>Current raid</span>
      </label>
      {tier.bosses.length === 0 && <p className={styles.noticeText}>No bosses yet.</p>}
      <ul className={styles.bossList}>
        {tier.bosses.map((boss, bossIndex) => (
          <li key={boss.id} className={styles.bossItem}>
            <label className={styles.toggle}>
              <input
                type="checkbox"
                aria-label={`${boss.name} killed`}
                checked={boss.killedAt !== null}
                onChange={(event) => setBossKilled.mutate({ id: boss.id, killed: event.target.checked })}
              />
              <span>{boss.name}</span>
            </label>
            <div className={styles.controls}>
              <RenameControl
                target={boss.name}
                isPending={renameBoss.isPending}
                onSave={(name, onSaved) => renameBoss.mutate({ id: boss.id, name }, { onSuccess: onSaved })}
              />
              <MoveButtons
                target={boss.name}
                isFirst={bossIndex === 0}
                isLast={bossIndex === bossIds.length - 1}
                disabled={reorderBosses.isPending}
                onMove={(offset) =>
                  reorderBosses.mutate({ tierId: tier.id, ids: swapped(bossIds, bossIndex, offset) })
                }
              />
              <ConfirmDeleteButton target={boss.name} onConfirm={() => deleteBoss.mutate(boss.id)}>
                ✕
              </ConfirmDeleteButton>
            </div>
          </li>
        ))}
      </ul>
      <AddBossForm tier={tier} />
      {errors.map((message) => (
        <p key={message} role="alert" className={styles.error}>
          {message}
        </p>
      ))}
    </div>
  )
}

function OfficerTools() {
  const tiers = useRaidTiersData()

  if (tiers.isPending) {
    return <p className={styles.noticeText}>Loading raids…</p>
  }
  if (tiers.isError) {
    return <p className={styles.noticeText}>Raids could not be loaded.</p>
  }

  const nextSortOrder = tiers.data.reduce((highest, tier) => Math.max(highest, tier.sortOrder), 0) + 1
  const tierIds = tiers.data.map((tier) => tier.id)

  return (
    <>
      <CreateTierForm nextSortOrder={nextSortOrder} />
      <h2 className={styles.sectionTitle}>Raids</h2>
      {tiers.data.length === 0 && <p className={styles.noticeText}>No raids yet.</p>}
      <div className={styles.tierList}>
        {tiers.data.map((tier, index) => (
          <TierManager key={tier.id} tier={tier} tierIds={tierIds} index={index} />
        ))}
      </div>
    </>
  )
}

export function OfficerRaidProgress() {
  const currentUser = useCurrentUser()

  return (
    <section className={styles.page}>
      <h1 className={styles.title}>Manage Raid Progress</h1>
      <Link to="/raid-progress" className={styles.backLink}>
        Back to Raid Progress
      </Link>

      {currentUser.isPending && <p className={styles.noticeText}>Checking your access…</p>}
      {currentUser.isError && <p className={styles.noticeText}>Your access could not be checked.</p>}
      {currentUser.isSuccess && !currentUser.data?.isOfficer && (
        <div className={styles.noticeText}>
          <p>Only officers can manage raid progress.</p>
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
