import { cn } from '@/shared/lib/utils'

export interface Column<T> {
  key: string
  header: React.ReactNode
  cell: (row: T, index: number) => React.ReactNode
  className?: string
}

export interface TableProps<T> {
  columns: Column<T>[]
  data: T[]
  keyFn: (row: T, index: number) => string | number
  className?: string
  emptyState?: React.ReactNode
}

export function Table<T>({ columns, data, keyFn, className, emptyState }: TableProps<T>) {
  return (
    <div
      className={cn(
        'w-full overflow-x-auto rounded-[var(--radius-md)] border border-space-border',
        className,
      )}
    >
      <table className="w-full min-w-full text-sm">
        <thead className="sticky top-0 z-10 bg-space-dark border-b border-space-border">
          <tr>
            {columns.map((col) => (
              <th
                key={col.key}
                className={cn(
                  'px-4 py-3 text-left text-xs font-semibold text-text-muted uppercase tracking-wider whitespace-nowrap',
                  col.className,
                )}
              >
                {col.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {data.length === 0 ? (
            <tr>
              <td colSpan={columns.length} className="px-4 py-12 text-center text-text-muted">
                {emptyState ?? 'No data'}
              </td>
            </tr>
          ) : (
            data.map((row, i) => (
              <tr
                key={keyFn(row, i)}
                className={cn(
                  'border-b border-space-border/50 transition-colors',
                  i % 2 === 0 ? 'bg-space-card' : 'bg-space-dark',
                  'hover:bg-space-border/30',
                )}
              >
                {columns.map((col) => (
                  <td key={col.key} className={cn('px-4 py-3 text-text-primary', col.className)}>
                    {col.cell(row, i)}
                  </td>
                ))}
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  )
}
