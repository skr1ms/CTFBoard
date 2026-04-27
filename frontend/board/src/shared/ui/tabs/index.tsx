import { cn } from '@/shared/lib/utils'

export interface Tab<T extends string = string> {
  value: T
  label: React.ReactNode
  disabled?: boolean
}

export interface TabsProps<T extends string = string> {
  tabs: Tab<T>[]
  value: T
  onChange: (value: T) => void
  className?: string
  /** Render a scrollable horizontal strip (for tag filters) */
  scrollable?: boolean
}

export function Tabs<T extends string = string>({
  tabs,
  value,
  onChange,
  className,
  scrollable = false,
}: TabsProps<T>) {
  return (
    <div
      role="tablist"
      className={cn(
        'flex gap-1 border-b border-space-border pb-0',
        scrollable && 'overflow-x-auto scrollbar-none',
        className,
      )}
    >
      {tabs.map((tab) => (
        <button
          key={tab.value}
          role="tab"
          aria-selected={value === tab.value}
          disabled={tab.disabled}
          onClick={() => onChange(tab.value)}
          className={cn(
            'px-4 py-2 text-sm font-medium whitespace-nowrap transition-colors',
            'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cosmic-blue rounded-t',
            'border-b-2 -mb-px',
            value === tab.value
              ? 'border-cosmic-blue text-cosmic-blue'
              : 'border-transparent text-text-secondary hover:text-text-primary hover:border-space-border-bright',
            tab.disabled && 'opacity-40 pointer-events-none',
          )}
        >
          {tab.label}
        </button>
      ))}
    </div>
  )
}
