import type { ReactNode } from 'react'
import { XIcon } from 'lucide-react'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

interface Props {
  variant?: 'default' | 'destructive' | 'success'
  title?: string
  children: ReactNode
  onClose: () => void
  endAction?: ReactNode
  className?: string
}

const DEFAULT_TITLES = {
  default: 'notice',
  success: 'ok',
  destructive: 'error',
} as const

export default function DismissibleAlert({
  variant = 'default',
  title,
  children,
  onClose,
  endAction,
  className,
}: Props) {
  return (
    <Alert variant={variant} className={cn('pr-24', className)}>
      <AlertTitle className={variant === 'destructive' ? 'text-destructive' : undefined}>
        {title ?? DEFAULT_TITLES[variant]}
      </AlertTitle>
      <AlertDescription>{children}</AlertDescription>
      <div className="absolute top-2 right-2 flex items-center gap-1">
        {endAction}
        <Button variant="ghost" size="icon-sm" aria-label="Dismiss" onClick={onClose}>
          <XIcon />
        </Button>
      </div>
    </Alert>
  )
}
