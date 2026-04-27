import { cn } from '@/shared/lib/utils'

export type BadgeVariant = 'default' | 'blue' | 'purple' | 'green' | 'red' | 'yellow' | 'cyan'

export interface BadgeProps {
  variant?: BadgeVariant
  children: React.ReactNode
  className?: string
}

const variants: Record<BadgeVariant, string> = {
  default: 'bg-space-border text-text-secondary',
  blue: 'bg-cosmic-blue-dim text-cosmic-blue border border-cosmic-blue/30',
  purple: 'bg-nebula-purple-dim text-nebula-purple border border-nebula-purple/30',
  green: 'bg-stellar-green-dim text-stellar-green border border-stellar-green/30',
  red: 'bg-nova-red-dim text-nova-red border border-nova-red/30',
  yellow: 'bg-solar-yellow-dim text-solar-yellow border border-solar-yellow/30',
  cyan: 'bg-aurora-cyan-dim text-aurora-cyan border border-aurora-cyan/30',
}

export function Badge({ variant = 'default', children, className }: BadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium',
        variants[variant],
        className,
      )}
    >
      {children}
    </span>
  )
}
