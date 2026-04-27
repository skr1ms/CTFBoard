import { api, isApiError } from '@/shared/api/client'
import type { components, operations } from '@/shared/api/schema.d'
import { EmptyState } from '@/shared/ui/empty-state'
import { Skeleton } from '@/shared/ui/skeleton'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { toast } from 'sonner'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------
type Award = NonNullable<
  operations['GetAdminAwards']['responses']['200']['content']['application/json']
>[number]

type CreateBody = NonNullable<
  operations['PostAdminAwards']['requestBody']
>['content']['application/json']

type TeamOption = NonNullable<components['schemas']['AdminTeamListResponse']['data']>[number]

// ---------------------------------------------------------------------------
// Hooks
// ---------------------------------------------------------------------------
function useAwards() {
  return useQuery({
    queryKey: ['admin', 'awards'],
    queryFn: async () => {
      const { data, error } = await api.GET('/admin/awards')
      if (error) throw error
      return (data ?? []) as Award[]
    },
    staleTime: 0,
  })
}

function useTeamsAll() {
  return useQuery({
    queryKey: ['admin', 'teams-all'],
    queryFn: async () => {
      const { data, error } = await api.GET('/admin/teams', {
        params: { query: { per_page: 500 } },
      })
      if (error) throw error
      return ((data as components['schemas']['AdminTeamListResponse']).data ?? []) as TeamOption[]
    },
    staleTime: 60_000,
  })
}

function useCreateAward() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (body: CreateBody) => {
      const { error } = await api.POST('/admin/awards', { body })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'awards'] })
      toast.success('Award created')
    },
    onError: (err) => toast.error(isApiError(err) ? err.message : 'Create failed'),
  })
}

function useDeleteAward() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      const { error } = await api.DELETE('/admin/awards/{ID}', {
        params: { path: { ID: id } },
      })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'awards'] })
      toast.success('Award deleted')
    },
    onError: (err) => toast.error(isApiError(err) ? err.message : 'Delete failed'),
  })
}

// ---------------------------------------------------------------------------
// Create form
// ---------------------------------------------------------------------------
function CreateAwardForm({ teams }: { teams: TeamOption[] }) {
  const { mutateAsync: create, isPending } = useCreateAward()
  const [teamId, setTeamId] = useState('')
  const [value, setValue] = useState('')
  const [description, setDescription] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const v = parseInt(value, 10)
    if (!teamId || isNaN(v) || !description.trim()) return
    try {
      await create({ team_id: teamId, value: v, description: description.trim() })
      setValue('')
      setDescription('')
      setTeamId('')
    } catch {
      // error already toasted
    }
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="flex flex-col gap-4 p-4 rounded-[var(--radius-lg)] border border-space-border bg-space-card"
    >
      <h2 className="text-sm font-semibold text-text-primary">Create Award</h2>

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
        {/* Team select */}
        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-text-muted">Team</label>
          <select
            value={teamId}
            onChange={(e) => setTeamId(e.target.value)}
            required
            className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-2 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
          >
            <option value="">Select team…</option>
            {teams.map((t) => (
              <option key={t.id} value={t.id ?? ''}>
                {t.name ?? t.id}
              </option>
            ))}
          </select>
        </div>

        {/* Value */}
        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-text-muted">
            Value <span className="text-text-muted/60">(+/- pts)</span>
          </label>
          <input
            type="number"
            value={value}
            onChange={(e) => setValue(e.target.value)}
            placeholder="e.g. 100 or -50"
            required
            className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
          />
        </div>

        {/* Description */}
        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-text-muted">Description</label>
          <input
            type="text"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Reason for award"
            required
            className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
          />
        </div>
      </div>

      <div className="flex justify-end">
        <button
          type="submit"
          disabled={isPending || !teamId || !value || !description.trim()}
          className="px-4 py-1.5 text-sm rounded-[var(--radius-md)] bg-cosmic-blue hover:bg-cosmic-blue/90 text-white transition-colors disabled:opacity-50"
        >
          {isPending ? 'Creating…' : 'Create award'}
        </button>
      </div>
    </form>
  )
}

// ---------------------------------------------------------------------------
// Row skeleton
// ---------------------------------------------------------------------------
function RowSkeleton() {
  return (
    <>
      {Array.from({ length: 5 }).map((_, i) => (
        <tr key={i} className="border-b border-space-border">
          {Array.from({ length: 5 }).map((__, j) => (
            <td key={j} className="px-3 py-2.5">
              <Skeleton className="h-4 w-full" />
            </td>
          ))}
        </tr>
      ))}
    </>
  )
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------
export function AdminAwardsPage() {
  const { data: awards = [], isLoading: loadingAwards } = useAwards()
  const { data: teams = [], isLoading: loadingTeams } = useTeamsAll()
  const { mutate: deleteAward, isPending: deleting } = useDeleteAward()
  const [filterTeamId, setFilterTeamId] = useState('')

  const displayed = filterTeamId ? awards.filter((a) => a.team_id === filterTeamId) : awards

  // Sort: newest first
  const sorted = [...displayed].sort((a, b) =>
    (b.created_at ?? '').localeCompare(a.created_at ?? ''),
  )

  return (
    <div className="flex flex-col gap-5">
      <h1
        className="text-2xl font-black text-text-primary"
        style={{ fontFamily: 'var(--font-display)' }}
      >
        Awards
      </h1>

      {/* Create form */}
      {loadingTeams ? <Skeleton className="h-36 w-full" /> : <CreateAwardForm teams={teams} />}

      {/* Team filter */}
      <div className="flex items-center gap-3">
        <label className="text-xs font-medium text-text-muted whitespace-nowrap">
          Filter by team:
        </label>
        <select
          value={filterTeamId}
          onChange={(e) => setFilterTeamId(e.target.value)}
          className="h-8 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-2 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
        >
          <option value="">All teams</option>
          {teams.map((t) => (
            <option key={t.id} value={t.id ?? ''}>
              {t.name ?? t.id}
            </option>
          ))}
        </select>
        {filterTeamId && (
          <button
            onClick={() => setFilterTeamId('')}
            className="text-xs text-text-muted hover:text-text-primary transition-colors"
          >
            Clear
          </button>
        )}
        <span className="ml-auto text-xs text-text-muted">
          {sorted.length} award{sorted.length !== 1 ? 's' : ''}
        </span>
      </div>

      {/* Table */}
      <div className="overflow-x-auto rounded-[var(--radius-lg)] border border-space-border">
        <table className="w-full min-w-[580px]">
          <thead>
            <tr className="border-b border-space-border bg-space-dark/60">
              <th className="px-3 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
                Team
              </th>
              <th className="px-3 py-2.5 text-right text-xs font-semibold text-text-muted uppercase tracking-wide">
                Value
              </th>
              <th className="px-3 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
                Description
              </th>
              <th className="px-3 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide hidden md:table-cell">
                Created
              </th>
              <th className="px-3 py-2.5 text-right text-xs font-semibold text-text-muted uppercase tracking-wide">
                Actions
              </th>
            </tr>
          </thead>
          <tbody>
            {loadingAwards ? (
              <RowSkeleton />
            ) : sorted.length === 0 ? (
              <tr>
                <td colSpan={5}>
                  <EmptyState
                    message={awards.length === 0 ? 'No awards yet.' : 'No awards for this team.'}
                  />
                </td>
              </tr>
            ) : (
              sorted.map((award) => {
                const team = teams.find((t) => t.id === award.team_id)
                const isBonus = (award.value ?? 0) >= 0
                return (
                  <tr
                    key={award.id}
                    data-testid={`admin-row-${award.id}`}
                    className="border-b border-space-border last:border-b-0 hover:bg-space-border/20 transition-colors"
                  >
                    <td className="px-3 py-2.5 text-sm text-text-primary font-medium">
                      {team?.name ?? award.team_id ?? '-'}
                    </td>
                    <td className="px-3 py-2.5 text-right text-sm font-mono font-semibold">
                      <span className={isBonus ? 'text-nebula-green' : 'text-nova-red'}>
                        {isBonus ? '+' : ''}
                        {award.value ?? 0}
                      </span>
                    </td>
                    <td className="px-3 py-2.5 text-sm text-text-secondary">
                      {award.description ?? '-'}
                    </td>
                    <td className="px-3 py-2.5 text-xs text-text-muted hidden md:table-cell whitespace-nowrap">
                      {award.created_at ? new Date(award.created_at).toLocaleString() : '-'}
                    </td>
                    <td className="px-3 py-2.5 text-right">
                      <button
                        data-testid={`admin-delete-${award.id}`}
                        onClick={() => award.id && deleteAward(award.id)}
                        disabled={deleting}
                        className="text-xs text-nova-red hover:underline disabled:opacity-40"
                      >
                        Delete
                      </button>
                    </td>
                  </tr>
                )
              })
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}
