import { cn } from '@/shared/lib/utils'
import { useEffect, useRef, useState } from 'react'

export interface DropdownItem {
  label: React.ReactNode
  value: string
  disabled?: boolean
  danger?: boolean
}

export interface DropdownProps {
  trigger: React.ReactNode
  items: DropdownItem[]
  onSelect: (value: string) => void
  align?: 'left' | 'right'
  className?: string
}

export function Dropdown({ trigger, items, onSelect, align = 'left', className }: DropdownProps) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    const keyHandler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', handler)
    document.addEventListener('keydown', keyHandler)
    return () => {
      document.removeEventListener('mousedown', handler)
      document.removeEventListener('keydown', keyHandler)
    }
  }, [open])

  return (
    <div ref={ref} className={cn('relative inline-block', className)}>
      <div onClick={() => setOpen((v) => !v)} className="cursor-pointer">
        {trigger}
      </div>

      {open && (
        <div
          role="menu"
          className={cn(
            'absolute z-50 mt-1 min-w-40 rounded-[var(--radius-md)] border border-space-border bg-space-card shadow-xl py-1',
            align === 'right' ? 'right-0' : 'left-0',
            'animate-in fade-in-0 zoom-in-95 duration-150',
          )}
        >
          {items.map((item) => (
            <button
              key={item.value}
              role="menuitem"
              disabled={item.disabled}
              onClick={() => {
                onSelect(item.value)
                setOpen(false)
              }}
              className={cn(
                'w-full text-left px-3 py-2 text-sm transition-colors',
                'focus-visible:outline-none focus-visible:bg-space-border',
                item.danger
                  ? 'text-nova-red hover:bg-nova-red/10'
                  : 'text-text-primary hover:bg-space-border/60',
                item.disabled && 'opacity-40 pointer-events-none',
              )}
            >
              {item.label}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
