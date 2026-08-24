import { cn } from '@/lib/utils'

interface MonogramProps {
  /** Rendered size in px. Outline strokes vanish below ~24 px — use filled there. */
  size?: number
  /** Filled disc for small sizes; outline elsewhere. */
  variant?: 'outline' | 'filled'
  className?: string
}

/**
 * The «34» inside a circle — avatar, favicon and document seal.
 * Stroke weight is 1/24 of the diameter; the figures sit optically centred
 * at 42% of the diameter. Drawn in currentColor: ink on bone, bone on ink —
 * never brass.
 */
export default function Monogram({ size = 48, variant = 'outline', className }: MonogramProps) {
  const stroke = 48 / 24

  return (
    <svg
      viewBox="0 0 48 48"
      width={size}
      height={size}
      role="img"
      aria-label="lab34 monogram"
      className={cn('shrink-0', className)}
    >
      {variant === 'filled' ? (
        <circle cx="24" cy="24" r="24" fill="currentColor" />
      ) : (
        <circle cx="24" cy="24" r={24 - stroke / 2} fill="none" stroke="currentColor" strokeWidth={stroke} />
      )}
      <text
        x="24"
        y="24.5"
        textAnchor="middle"
        dominantBaseline="central"
        fontFamily="'IBM Plex Mono', ui-monospace, monospace"
        fontWeight="500"
        fontSize={48 * 0.42}
        fill={variant === 'filled' ? 'var(--background)' : 'currentColor'}
      >
        34
      </text>
    </svg>
  )
}
