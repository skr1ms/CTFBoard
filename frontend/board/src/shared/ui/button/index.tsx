import { cn } from '@/shared/lib/utils'
import { Spinner } from '@/shared/ui/spinner'
import { forwardRef } from 'react'

export type ButtonVariant = 'primary' | 'secondary' | 'danger' | 'ghost'
export type ButtonSize = 'sm' | 'md' | 'lg'

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant
  size?: ButtonSize
  loading?: boolean
  children: React.ReactNode
}

const base =
  'inline-flex items-center justify-center gap-2 font-medium rounded-md transition-all duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cosmic-blue focus-visible:ring-offset-2 focus-visible:ring-offset-deep-space disabled:pointer-events-none disabled:opacity-50 select-none'

const variants: Record<ButtonVariant, string> = {
  primary:
    'text-white bg-gradient-to-r from-cosmic-blue to-nebula-purple hover:opacity-90 hover:shadow-[var(--glow-blue-sm)] active:scale-[0.98]',
  secondary:
    'text-cosmic-blue border border-cosmic-blue/50 bg-transparent hover:bg-cosmic-blue/10 hover:border-cosmic-blue hover:shadow-[var(--glow-blue-sm)] active:scale-[0.98]',
  danger:
    'text-white bg-nova-red hover:bg-nova-red/90 hover:shadow-[var(--glow-red-sm)] active:scale-[0.98]',
  ghost:
    'text-text-secondary bg-transparent hover:bg-space-card hover:text-text-primary active:scale-[0.98]',
}

const sizes: Record<ButtonSize, string> = {
  sm: 'h-8 px-3 text-sm',
  md: 'h-10 px-4 text-sm',
  lg: 'h-12 px-6 text-base',
}

const spinnerSizes: Record<ButtonSize, 'sm' | 'sm' | 'md'> = {
  sm: 'sm',
  md: 'sm',
  lg: 'md',
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  (
    { variant = 'primary', size = 'md', loading = false, disabled, className, children, ...props },
    ref,
  ) => (
    <button
      ref={ref}
      disabled={disabled ?? loading}
      aria-busy={loading}
      className={cn(base, variants[variant], sizes[size], className)}
      {...props}
    >
      {loading && <Spinner size={spinnerSizes[size]} />}
      {children}
    </button>
  ),
)

Button.displayName = 'Button'
