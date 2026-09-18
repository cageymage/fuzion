export function formatCountdown(msRemaining: number): string {
  if (msRemaining <= 0) {
    return 'Starting now'
  }
  const totalMinutes = Math.floor(msRemaining / 60_000)
  const days = Math.floor(totalMinutes / 1_440)
  const hours = Math.floor((totalMinutes % 1_440) / 60)
  const minutes = totalMinutes % 60

  if (days > 0) {
    return `${days}d ${hours}h`
  }
  if (hours > 0) {
    return `${hours}h ${minutes}m`
  }
  return `${minutes}m`
}

export function formatViewerCount(viewers: number): string {
  if (viewers < 1_000) {
    return String(viewers)
  }
  const thousands = viewers / 1_000
  return `${thousands < 10 ? thousands.toFixed(1) : Math.round(thousands)}K`
}

export function formatRelativeDate(isoDate: string, now: number = Date.now()): string {
  const days = Math.floor((now - new Date(isoDate).getTime()) / 86_400_000)
  if (days <= 0) {
    return 'today'
  }
  if (days === 1) {
    return 'yesterday'
  }
  return `${days} days ago`
}
