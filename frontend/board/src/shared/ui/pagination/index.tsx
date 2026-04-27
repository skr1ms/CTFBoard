import { cn } from '@/shared/lib/utils'

export interface PaginationProps {
  page: number
  totalPages: number
  onChange: (page: number) => void
  className?: string
}

// eslint-disable-next-line react-refresh/only-export-components
export function buildRange(current: number, total: number): (number | '...')[] {
  if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1)

  const pages: (number | '...')[] = [1]

  if (current > 3) pages.push('...')

  const start = Math.max(2, current - 1)
  const end = Math.min(total - 1, current + 1)

  for (let i = start; i <= end; i++) pages.push(i)

  if (current < total - 2) pages.push('...')
  pages.push(total)

  return pages
}

export function Pagination({ page, totalPages, onChange, className }: PaginationProps) {
  if (totalPages <= 1) return null

  const range = buildRange(page, totalPages)

  const btn = (content: React.ReactNode, target: number, active = false, disabled = false) => (
    <button
      key={`${content}-${target}`}
      onClick={() => !disabled && onChange(target)}
      disabled={disabled}
      aria-label={typeof content === 'number' ? `Page ${content}` : String(content)}
      aria-current={active ? 'page' : undefined}
      data-testid={
        content === '‹'
          ? 'pagination-prev'
          : content === '›'
            ? 'pagination-next'
            : typeof content === 'number'
              ? `pagination-page-${content}`
              : undefined
      }
      className={cn(
        'h-8 min-w-8 px-2 rounded text-sm transition-colors',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cosmic-blue',
        active
          ? 'bg-cosmic-blue text-deep-space font-semibold'
          : 'text-text-secondary hover:bg-space-border hover:text-text-primary',
        disabled && 'opacity-40 pointer-events-none',
      )}
    >
      {content}
    </button>
  )

  return (
    <nav aria-label="Pagination" className={cn('flex items-center gap-1', className)}>
      {btn('‹', page - 1, false, page === 1)}
      {range.map((item, i) =>
        item === '...' ? (
          <span key={`dots-${i}`} className="px-1 text-text-muted select-none">
            …
          </span>
        ) : (
          btn(item, item, item === page)
        ),
      )}
      {btn('›', page + 1, false, page === totalPages)}
    </nav>
  )
}
