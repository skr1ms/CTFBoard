import { cn } from '@/shared/lib/utils'
import type { InputProps } from '@/shared/ui/input'
import { forwardRef, useState } from 'react'

export type PasswordInputProps = Omit<InputProps, 'type'>

export const PasswordInput = forwardRef<HTMLInputElement, PasswordInputProps>(
  ({ label, error, hint, className, id, ...props }, ref) => {
    const [show, setShow] = useState(false)
    const inputId = id ?? label?.toLowerCase().replace(/\s+/g, '-')

    return (
      <div className="flex flex-col gap-1.5">
        {label && (
          <label htmlFor={inputId} className="text-sm font-medium text-text-secondary">
            {label}
          </label>
        )}
        <div className="relative">
          <input
            ref={ref}
            id={inputId}
            type={show ? 'text' : 'password'}
            aria-invalid={Boolean(error)}
            aria-describedby={error ? `${inputId}-error` : hint ? `${inputId}-hint` : undefined}
            className={cn(
              'h-10 w-full rounded-md border bg-space-card px-3 pr-10 text-sm text-text-primary placeholder:text-text-muted',
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
          <button
            type="button"
            tabIndex={-1}
            onClick={() => setShow((s) => !s)}
            className="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-primary transition-colors"
            aria-label={show ? 'Hide password' : 'Show password'}
          >
            {show ? (
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94" />
                <path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19" />
                <line x1="1" y1="1" x2="23" y2="23" />
              </svg>
            ) : (
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
                <circle cx="12" cy="12" r="3" />
              </svg>
            )}
          </button>
        </div>
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

PasswordInput.displayName = 'PasswordInput'
