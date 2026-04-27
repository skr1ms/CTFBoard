import { cn } from '@/shared/lib/utils'
import { forwardRef } from 'react'

export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string
  error?: string | undefined
  hint?: string | undefined
}

export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ label, error, hint, className, id, ...props }, ref) => {
    const inputId = id ?? label?.toLowerCase().replace(/\s+/g, '-')

    return (
      <div className="flex flex-col gap-1.5">
        {label && (
          <label htmlFor={inputId} className="text-sm font-medium text-text-secondary">
            {label}
          </label>
        )}
        <input
          ref={ref}
          id={inputId}
          aria-invalid={Boolean(error)}
          aria-describedby={error ? `${inputId}-error` : hint ? `${inputId}-hint` : undefined}
          className={cn(
            'h-10 w-full rounded-md border bg-space-card px-3 text-sm text-text-primary placeholder:text-text-muted',
            'transition-all duration-150',
            'focus:outline-none focus:ring-2 focus:ring-offset-1 focus:ring-offset-deep-space',
            error
              ? 'border-nova-red focus:ring-nova-red/60 focus:border-nova-red'
              : 'border-space-border focus:ring-cosmic-blue/50 focus:border-cosmic-blue/60',
            'disabled:opacity-50 disabled:cursor-not-allowed',
            className,
          )}
          {...props}
        />
        {error && (
          <p id={`${inputId}-error`} role="alert" className="text-xs text-nova-red">
            {error}
          </p>
        )}
        {!error && hint && (
          <p id={`${inputId}-hint`} className="text-xs text-text-muted">
            {hint}
          </p>
        )}
      </div>
    )
  },
)

Input.displayName = 'Input'
