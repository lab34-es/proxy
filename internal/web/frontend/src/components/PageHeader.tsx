import type { ReactNode } from 'react'

// Section header in the guidelines' register: a small brass-toned index
// figure beside the title, closed off by a hairline rule.
export default function PageHeader({
  index,
  title,
  description,
  actions,
}: {
  index: string
  title: string
  description?: string
  actions?: ReactNode
}) {
  return (
    <header className="mb-6 border-b pb-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-baseline gap-3">
          <span className="font-mono text-sm text-brass-ink">{index}</span>
          <h1 className="text-2xl">{title}</h1>
        </div>
        {actions && <div className="flex items-center gap-2">{actions}</div>}
      </div>
      {description && <p className="mt-1.5 pl-9 text-sm text-muted-foreground">{description}</p>}
    </header>
  )
}
