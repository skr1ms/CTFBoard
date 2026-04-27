import { DEBOUNCE_MS } from '@/shared/lib/constants'
import { useDebounce } from '@/shared/lib/hooks/useDebounce'
import { cn } from '@/shared/lib/utils'
import { useEffect, useState } from 'react'

export interface SearchInputProps {
  value?: string
  onChange: (value: string) => void
  placeholder?: string
  debounce?: number
  className?: string
  disabled?: boolean
}

export function SearchInput({
  value: externalValue = '',
  onChange,
  placeholder = 'Search…',
  debounce = DEBOUNCE_MS,
  className,
  disabled = false,
}: SearchInputProps) {
  const [prevExternal, setPrevExternal] = useState(externalValue)
  const [local, setLocal] = useState(externalValue)
  const debounced = useDebounce(local, debounce)

  if (prevExternal !== externalValue) {
    setPrevExternal(externalValue)
    setLocal(externalValue)
  }

  useEffect(() => {
    onChange(debounced)
  }, [debounced, onChange])

  return (
    <div className={cn('relative', className)}>
      <svg
        className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-muted pointer-events-none"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={2}
          d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
        />
      </svg>
      <input
        type="search"
        value={local}
        onChange={(e) => setLocal(e.target.value)}
        placeholder={placeholder}
        disabled={disabled}
        className={cn(
          'h-10 w-full rounded-md border border-space-border bg-space-card',
          'pl-9 pr-3 text-sm text-text-primary placeholder:text-text-muted',
          'focus:outline-none focus:ring-2 focus:ring-cosmic-blue/50 focus:border-cosmic-blue/60',
          'transition-all duration-150 disabled:opacity-50',
        )}
      />
    </div>
  )
}
