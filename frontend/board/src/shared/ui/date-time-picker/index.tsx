import { cn } from '@/shared/lib/utils'
import { useRef, useState } from 'react'

export interface DateTimePickerProps {
  value?: Date
  onChange?: (date: Date | undefined) => void
  label?: string
  placeholder?: string
  disabled?: boolean
  error?: string
  className?: string
  min?: Date
  max?: Date
}

function toLocalInputValue(date: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  return (
    `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}` +
    `T${pad(date.getHours())}:${pad(date.getMinutes())}`
  )
}

export function DateTimePicker({
  value,
  onChange,
  label,
  placeholder,
  disabled = false,
  error,
  className,
  min,
  max,
}: DateTimePickerProps) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [prevValue, setPrevValue] = useState(value)
  const [inputValue, setInputValue] = useState(() => (value ? toLocalInputValue(value) : ''))

  if (prevValue !== value) {
    setPrevValue(value)
    setInputValue(value ? toLocalInputValue(value) : '')
  }

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const raw = e.target.value
    setInputValue(raw)
    if (!raw) {
      onChange?.(undefined)
      return
    }
    const parsed = new Date(raw)
    if (!isNaN(parsed.getTime())) {
      onChange?.(parsed)
    }
  }

  return (
    <div className={cn('flex flex-col gap-1', className)}>
      {label && <label className="text-sm font-medium text-text-secondary">{label}</label>}
      <input
        ref={inputRef}
        type="datetime-local"
        value={inputValue}
        onChange={handleChange}
        disabled={disabled}
        placeholder={placeholder}
        min={min ? toLocalInputValue(min) : undefined}
        max={max ? toLocalInputValue(max) : undefined}
        className={cn(
          'w-full rounded-[var(--radius-md)] border bg-space-dark px-3 py-2 text-sm text-text-primary',
          'focus:outline-none focus:ring-2 focus:ring-cosmic-blue/50 focus:border-cosmic-blue',
          'disabled:opacity-50 disabled:cursor-not-allowed',
          '[color-scheme:dark]',
          error
            ? 'border-red-500 focus:ring-red-500/50 focus:border-red-500'
            : 'border-space-border hover:border-text-muted',
        )}
      />
      {error && <span className="text-xs text-red-400">{error}</span>}
    </div>
  )
}
