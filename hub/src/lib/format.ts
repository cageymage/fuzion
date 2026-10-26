export function formatViewerCount(viewers: number): string {
  if (viewers < 1_000) {
    return String(viewers)
  }
  const thousands = viewers / 1_000
  return `${thousands < 10 ? thousands.toFixed(1) : Math.round(thousands)}K`
}
