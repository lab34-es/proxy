import * as React from 'react'

import { cn } from '@/lib/utils'

function Textarea({ className, ...props }: React.ComponentProps<'textarea'>) {
  return (
    <textarea
      data-slot="textarea"
      className={cn(
        'flex min-h-16 w-full border border-input bg-card px-3 py-2 text-sm text-foreground transition-colors outline-none',
        'placeholder:text-muted-foreground/70 selection:bg-brass/25',
        'focus-visible:border-brass focus-visible:ring-2 focus-visible:ring-ring/30',
        'disabled:cursor-not-allowed disabled:opacity-50',
        className,
      )}
      {...props}
    />
  )
}

export { Textarea }
