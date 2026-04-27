import { api } from '@/shared/api/client'
import type { components } from '@/shared/api/schema.d'
import { EmptyState } from '@/shared/ui/empty-state'
import { Pagination } from '@/shared/ui/pagination'
import { Skeleton } from '@/shared/ui/skeleton'
import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'

type UnlockRow = components['schemas']['HintUnlockResponse']

function formatDate(iso?: string) {
  if (!iso) return '-'
  return new Date(iso).toLocaleString()
}

export function AdminUnlocksPage() {
  const [page, setPage] = useState(1)
  const perPage = 50

  const { data, isLoading } = useQuery({
    queryKey: ['admin', 'unlocks', page],
    queryFn: async () => {
      const { data, error } = await api.GET('/admin/unlocks', {
        params: { query: { page, per_page: perPage } },
      })
      if (error) throw error
      return data
    },
    staleTime: 0,
    refetchInterval: 30_000,
  })

  const rows: UnlockRow[] = data?.data ?? []
  const meta = data?.meta
  const totalPages = meta ? Math.ceil((meta.total ?? 0) / perPage) : 1

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1
          className="text-2xl font-bold text-text-primary"
          style={{ fontFamily: 'var(--font-display)' }}
        >
          Hint Unlocks
        </h1>
        <p className="text-sm text-text-muted mt-1">
          All hint unlocks across teams and challenges.
        </p>
      </div>

      {isLoading ? (
        <div className="flex flex-col gap-2">
          {Array.from({ length: 8 }).map((_, i) => (
            <Skeleton key={i} className="h-10 w-full" />
          ))}
        </div>
      ) : rows.length === 0 ? (
        <EmptyState
          icon={<span className="text-4xl">🔓</span>}
          message="No hints have been unlocked by any team."
        />
      ) : (
        <>
          <div className="rounded-[var(--radius-lg)] border border-space-border overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-space-border bg-space-card">
                    <th className="text-left px-4 py-3 text-xs font-semibold text-text-muted uppercase tracking-wide">
                      Hint ID
                    </th>
                    <th className="text-left px-4 py-3 text-xs font-semibold text-text-muted uppercase tracking-wide">
                      Challenge ID
                    </th>
                    <th className="text-left px-4 py-3 text-xs font-semibold text-text-muted uppercase tracking-wide">
                      Team ID
                    </th>
                    <th className="text-right px-4 py-3 text-xs font-semibold text-text-muted uppercase tracking-wide">
                      Cost (pts)
                    </th>
                    <th className="text-left px-4 py-3 text-xs font-semibold text-text-muted uppercase tracking-wide">
                      Unlocked At
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((row, i) => (
                    <tr
                      key={row.id ?? `${row.hint_id ?? '?'}-${row.team_id ?? i}`}
                      className={`border-b border-space-border last:border-b-0 ${
                        i % 2 === 1 ? 'bg-space-card/30' : ''
                      }`}
                    >
                      <td className="px-4 py-3 font-mono text-xs text-text-secondary truncate max-w-[140px]">
                        {row.hint_id ?? '-'}
                      </td>
                      <td className="px-4 py-3 font-mono text-xs text-text-secondary truncate max-w-[140px]">
                        {row.challenge_id ?? '-'}
                      </td>
                      <td className="px-4 py-3 font-mono text-xs text-text-secondary truncate max-w-[140px]">
                        {row.team_id ?? '-'}
                      </td>
                      <td className="px-4 py-3 text-right text-nova-yellow font-medium">
                        {row.hint_cost ?? 0}
                      </td>
                      <td className="px-4 py-3 text-text-muted whitespace-nowrap">
                        {formatDate(row.unlocked_at)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>

          {totalPages > 1 && (
            <div className="flex justify-center">
              <Pagination page={page} totalPages={totalPages} onChange={setPage} />
            </div>
          )}
        </>
      )}
    </div>
  )
}
