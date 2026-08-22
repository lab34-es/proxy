import { cn } from '@/lib/utils'

/**
 * The open-source lockup: the project hangs off the parent name with a brass
 * slash. Parent in Medium 500 with the wordmark's tightened tracking, project
 * name always lowercase Regular 400.
 */
export default function Lockup({
  project = 'proxy',
  className,
}: {
  project?: string
  className?: string
}) {
  return (
    <span className={cn('font-mono lowercase leading-none', className)}>
      <span className="tracking-logotype font-medium">lab34</span>
      <span className="text-brass">/</span>
      <span className="font-normal">{project}</span>
    </span>
  )
}
