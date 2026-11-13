import { useRef, type KeyboardEvent } from 'react'
import styles from './Professions.module.css'

export const professionPanelId = 'professions-panel'

export function professionTabId(profession: string): string {
  return `profession-tab-${profession.replace(/\s+/g, '-')}`
}

interface ProfessionTabsProps {
  rows: readonly (readonly string[])[]
  rowLabels: readonly string[]
  counts: Readonly<Record<string, number>>
  selected: string | null
  onSelect: (profession: string) => void
}

export function ProfessionTabs({ rows, rowLabels, counts, selected, onSelect }: ProfessionTabsProps) {
  const tabs = useRef(new Map<string, HTMLButtonElement>())

  function moveWithinRow(row: readonly string[], event: KeyboardEvent<HTMLDivElement>) {
    const enabled = row.filter((profession) => (counts[profession] ?? 0) > 0)
    const current = enabled.findIndex((profession) => tabs.current.get(profession) === event.target)
    if (current === -1) return

    let next: number
    switch (event.key) {
      case 'ArrowRight':
        next = (current + 1) % enabled.length
        break
      case 'ArrowLeft':
        next = (current - 1 + enabled.length) % enabled.length
        break
      case 'Home':
        next = 0
        break
      case 'End':
        next = enabled.length - 1
        break
      default:
        return
    }

    event.preventDefault()
    onSelect(enabled[next])
    tabs.current.get(enabled[next])?.focus()
  }

  return (
    <>
      {rows.map((row, index) => (
        <div
          key={rowLabels[index]}
          role="tablist"
          aria-label={rowLabels[index]}
          className={styles.tabRow}
          onKeyDown={(event) => moveWithinRow(row, event)}
        >
          {row.map((profession) => {
            const count = counts[profession] ?? 0
            const disabled = count === 0
            const isSelected = profession === selected
            const rowHasSelection = selected !== null && row.includes(selected)
            const isRowStop = rowHasSelection
              ? isSelected
              : profession === row.find((candidate) => (counts[candidate] ?? 0) > 0)
            return (
              <button
                key={profession}
                ref={(element) => {
                  if (element) tabs.current.set(profession, element)
                  else tabs.current.delete(profession)
                }}
                type="button"
                role="tab"
                id={professionTabId(profession)}
                aria-selected={isSelected}
                aria-controls={isSelected ? professionPanelId : undefined}
                aria-disabled={disabled}
                tabIndex={isRowStop ? 0 : -1}
                className={styles.tab}
                onClick={() => {
                  if (!disabled) onSelect(profession)
                }}
              >
                {profession} <span className={styles.tabCount}>({count})</span>
              </button>
            )
          })}
        </div>
      ))}
    </>
  )
}
