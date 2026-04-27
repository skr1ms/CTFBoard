import { api, isApiError } from '@/shared/api/client'
import type { operations } from '@/shared/api/schema.d'
import { EmptyState } from '@/shared/ui/empty-state'
import { Skeleton } from '@/shared/ui/skeleton'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { toast } from 'sonner'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------
type ScoreboardEntry = NonNullable<
  operations['GetScoreboard']['responses']['200']['content']['application/json']
>[number]

// ---------------------------------------------------------------------------
// Hooks
// ---------------------------------------------------------------------------
function useAdminScoreboard() {
  return useQuery({
    queryKey: ['admin', 'scoreboard-live'],
    queryFn: async () => {
      const { data, error } = await api.GET('/scoreboard', {
        params: { query: { live: true } },
      })
      if (error) throw error
      return (data ?? []) as ScoreboardEntry[]
    },
    staleTime: 0,
    refetchInterval: 30_000,
  })
}

function useSetTeamHidden(onToggled: (teamId: string, hidden: boolean) => void) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async ({ teamId, hidden }: { teamId: string; hidden: boolean }) => {
      const { error } = await api.PATCH('/admin/teams/{ID}/hidden', {
        params: { path: { ID: teamId } },
        body: { hidden },
      })
      if (error) throw error
    },
    onSuccess: (_, { teamId, hidden }) => {
      onToggled(teamId, hidden)
      void qc.invalidateQueries({ queryKey: ['admin', 'scoreboard-live'] })
      toast.success(hidden ? 'Team hidden from scoreboard' : 'Team visible on scoreboard')
    },
    onError: (err) => toast.error(isApiError(err) ? err.message : 'Update failed'),
  })
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------
export function AdminScoreboardPage() {
  const { data: entries = [], isLoading } = useAdminScoreboard()
  const [hiddenTeams, setHiddenTeams] = useState<Set<string>>(new Set())
  const { mutate: setHidden, isPending: toggling } = useSetTeamHidden((teamId, hidden) => {
    setHiddenTeams((prev) => {
      const next = new Set(prev)
      if (hidden) next.add(teamId)
      else next.delete(teamId)
      return next
    })
  })

  return (
    <div className="flex flex-col gap-5">
      <div className="flex items-center justify-between">
        <h1
          className="text-2xl font-black text-text-primary"
          style={{ fontFamily: 'var(--font-display)' }}
        >
          Scoreboard
        </h1>
        <span className="text-xs text-text-muted bg-cosmic-blue/10 border border-cosmic-blue/20 rounded px-2 py-1">
          live=true - bypasses freeze
        </span>
      </div>

      <div className="overflow-x-auto rounded-[var(--radius-lg)] border border-space-border">
        <table className="w-full min-w-[480px]">
          <thead>
            <tr className="border-b border-space-border bg-space-dark/60">
              <th className="px-3 py-2.5 text-center text-xs font-semibold text-text-muted uppercase tracking-wide w-12">
                #
              </th>
              <th className="px-3 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
                Team
              </th>
              <th className="px-3 py-2.5 text-right text-xs font-semibold text-text-muted uppercase tracking-wide">
                Points
              </th>
              <th className="px-3 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide hidden md:table-cell">
                Last solve
              </th>
              <th className="px-3 py-2.5 text-center text-xs font-semibold text-text-muted uppercase tracking-wide">
                Visibility
              </th>
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              Array.from({ length: 8 }).map((_, i) => (
                <tr key={i} className="border-b border-space-border">
                  {Array.from({ length: 5 }).map((__, j) => (
                    <td key={j} className="px-3 py-2.5">
                      <Skeleton className="h-4 w-full" />
                    </td>
                  ))}
                </tr>
              ))
            ) : entries.length === 0 ? (
              <tr>
                <td colSpan={5}>
                  <EmptyState message="No scoreboard data yet." />
                </td>
              </tr>
            ) : (
              entries.map((entry, idx) => (
                <tr
                  key={entry.team_id ?? entry.team_name ?? idx}
                  className="border-b border-space-border last:border-b-0 hover:bg-space-border/20 transition-colors"
                >
                  <td className="px-3 py-2.5 text-center text-sm font-mono text-text-muted">
                    {idx + 1}
                  </td>
                  <td className="px-3 py-2.5 text-sm text-text-primary font-medium">
                    {entry.team_name ?? entry.team_id ?? '-'}
                  </td>
                  <td className="px-3 py-2.5 text-right text-sm font-mono font-semibold text-nebula-green">
                    {entry.points ?? 0}
                  </td>
                  <td className="px-3 py-2.5 text-xs text-text-muted hidden md:table-cell whitespace-nowrap">
                    {entry.last_solved ? new Date(entry.last_solved).toLocaleString() : '-'}
                  </td>
                  <td className="px-3 py-2.5 text-center">
                    {entry.team_id &&
                      (() => {
                        const isHidden = hiddenTeams.has(entry.team_id!)
                        return (
                          <button
                            onClick={() => setHidden({ teamId: entry.team_id!, hidden: !isHidden })}
                            disabled={toggling}
                            title={
                              isHidden
                                ? 'Click to show on scoreboard'
                                : 'Click to hide from scoreboard'
                            }
                            className={`text-xs transition-colors disabled:opacity-40 ${isHidden ? 'text-nova-red hover:text-nebula-green' : 'text-nebula-green hover:text-nova-red'}`}
                          >
                            {isHidden ? 'Hidden' : 'Visible'}
                          </button>
                        )
                      })()}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      <p className="text-xs text-text-muted">
        {entries.length} team{entries.length !== 1 ? 's' : ''} · auto-refreshes every 30s
      </p>
    </div>
  )
}
