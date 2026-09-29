import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { loginUrl } from '../../api/auth'
import { useCurrentUser } from '../../hooks/useCurrentUser'
import type { RaidTier } from '../../types/raidProgress'
import { useRaidTiersData } from '../RaidProgress/useRaidProgressData'
import styles from './OfficerRaidProgress.module.css'
import { useOfficerRaidProgress } from './useOfficerRaidProgress'

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
    const target = index + offset
    const reordered = [...bossNames]
    ;[reordered[index], reordered[target]] = [reordered[target], reordered[index]]
    setBossNames(reordered)
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

interface TierManagerProps {
  tier: RaidTier
}

function TierManager({ tier }: TierManagerProps) {
  const { setTierCurrent, setBossKilled, deleteTier, deleteBoss } = useOfficerRaidProgress()

  return (
    <div className={`card ${styles.card}`}>
      <div className={styles.tierHeader}>
        <h3 className={styles.tierName}>{tier.name}</h3>
        <ConfirmDeleteButton
          target={tier.name}
          disabled={tier.isCurrent}
          onConfirm={() => deleteTier.mutate(tier.id)}
        >
          Delete raid
        </ConfirmDeleteButton>
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
      <ul className={styles.bossList}>
        {tier.bosses.map((boss) => (
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
            <ConfirmDeleteButton target={boss.name} onConfirm={() => deleteBoss.mutate(boss.id)}>
              ✕
            </ConfirmDeleteButton>
          </li>
        ))}
      </ul>
      {setTierCurrent.isError && (
        <p role="alert" className={styles.error}>
          {setTierCurrent.error.message}
        </p>
      )}
      {setBossKilled.isError && (
        <p role="alert" className={styles.error}>
          {setBossKilled.error.message}
        </p>
      )}
      {deleteTier.isError && (
        <p role="alert" className={styles.error}>
          {deleteTier.error.message}
        </p>
      )}
      {deleteBoss.isError && (
        <p role="alert" className={styles.error}>
          {deleteBoss.error.message}
        </p>
      )}
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

  return (
    <>
      <CreateTierForm nextSortOrder={nextSortOrder} />
      <h2 className={styles.sectionTitle}>Raids</h2>
      {tiers.data.length === 0 && <p className={styles.noticeText}>No raids yet.</p>}
      <div className={styles.tierList}>
        {tiers.data.map((tier) => (
          <TierManager key={tier.id} tier={tier} />
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
