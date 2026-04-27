import { cn } from '@/shared/lib/utils'

export interface SkeletonProps {
  className?: string
  /** If true renders as a circle (for avatars) */
  circle?: boolean
}

export function Skeleton({ className, circle = false }: SkeletonProps) {
  return (
    <div
      aria-hidden="true"
      className={cn(
        'animate-pulse bg-space-border/60',
        circle ? 'rounded-full' : 'rounded-md',
        className,
      )}
    />
  )
}

/** Convenience wrapper for text line skeletons */
export function SkeletonText({ lines = 3, className }: { lines?: number; className?: string }) {
  return (
    <div className={cn('flex flex-col gap-2', className)}>
      {Array.from({ length: lines }).map((_, i) => (
        <Skeleton
          key={i}
          className={cn('h-4', i === lines - 1 && lines > 1 ? 'w-3/4' : 'w-full')}
        />
      ))}
    </div>
  )
}
