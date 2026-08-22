// Legible, dated and consistent: timestamps in a fixed log format.
export function formatDateTime(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

// Stat-tile values compact once they stop being scannable; tables keep full figures.
export function formatCompact(n: number): string {
  if (Math.abs(n) >= 100_000) {
    return new Intl.NumberFormat(undefined, { notation: 'compact', maximumFractionDigits: 1 }).format(n)
  }
  return n.toLocaleString()
}
