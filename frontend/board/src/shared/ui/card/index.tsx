import { cn } from '@/shared/lib/utils'

export interface CardProps {
  children: React.ReactNode
  className?: string
  /** Enable cosmic glow hover effect */
  glow?: boolean
  onClick?: () => void
}

export function Card({ children, className, glow = false, onClick }: CardProps) {
  return (
    <div
      onClick={onClick}
      className={cn(
        'rounded-[var(--radius-md)] border border-space-border bg-space-card p-4',
        'transition-all duration-200',
        glow && 'hover:border-cosmic-blue/40 hover:shadow-[var(--glow-blue-sm)] cursor-pointer',
        onClick && 'cursor-pointer',
        className,
      )}
    >
      {children}
    </div>
  )
}

export function CardHeader({
  children,
  className,
}: {
  children: React.ReactNode
  className?: string
}) {
  return <div className={cn('mb-3', className)}>{children}</div>
}

export function CardBody({
  children,
  className,
}: {
  children: React.ReactNode
  className?: string
}) {
  return <div className={cn('', className)}>{children}</div>
}

export function CardFooter({
  children,
  className,
}: {
  children: React.ReactNode
  className?: string
}) {
  return (
    <div
      className={cn('mt-4 pt-3 border-t border-space-border flex items-center gap-2', className)}
    >
      {children}
    </div>
  )
}
