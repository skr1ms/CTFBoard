import { api, isApiError } from '@/shared/api/client'
import type { components, operations } from '@/shared/api/schema.d'
import { useDebounce } from '@/shared/lib/hooks/useDebounce'
import { Badge } from '@/shared/ui/badge'
import { EmptyState } from '@/shared/ui/empty-state'
import { Skeleton } from '@/shared/ui/skeleton'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { toast } from 'sonner'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------
type Submission = NonNullable<components['schemas']['SubmissionListResponse']['data']>[number]

type PatchBody = NonNullable<
  operations['PatchAdminSubmissionsID']['requestBody']
>['content']['application/json']

type FilterTab = 'all' | 'correct' | 'incorrect'

const PER_PAGE = 50

// ---------------------------------------------------------------------------
// Hooks
// ---------------------------------------------------------------------------
function useAdminSubmissions(page: number) {
  return useQuery({
    queryKey: ['admin', 'submissions', page],
    queryFn: async () => {
      const { data, error } = await api.GET('/admin/submissions', {
        params: { query: { page, per_page: PER_PAGE, live: true } },
      })
      if (error) throw error
      return data as { data?: Submission[]; meta?: components['schemas']['PaginationMeta'] }
    },
    staleTime: 0,
  })
}

function useMarkCorrect() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async ({ id, is_correct }: { id: string; is_correct: boolean }) => {
      const body: PatchBody = { is_correct }
      const { error } = await api.PATCH('/admin/submissions/{ID}', {
        params: { path: { ID: id } },
        body,
      })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'submissions'] })
      toast.success('Submission updated')
    },
    onError: (err) => toast.error(isApiError(err) ? err.message : 'Update failed'),
  })
}

function useDeleteSubmission() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      const { error } = await api.DELETE('/admin/submissions/{ID}', {
        params: { path: { ID: id } },
      })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'submissions'] })
      toast.success('Submission deleted')
    },
    onError: (err) => toast.error(isApiError(err) ? err.message : 'Delete failed'),
  })
}

function useCreateSubmission() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (body: {
      user_id: string
      challenge_id: string
      submitted_flag: string
      is_correct: boolean
      team_id?: string
      ip?: string
    }) => {
      const { error } = await api.POST('/admin/submissions', { body })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'submissions'] })
      toast.success('Submission created')
    },
    onError: (err) => toast.error(isApiError(err) ? err.message : 'Create failed'),
  })
}

// ---------------------------------------------------------------------------
// Create submission form
// ---------------------------------------------------------------------------
const inputCls =
  'h-9 w-full rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40'

function CreateSubmissionForm() {
  const { mutateAsync: create, isPending } = useCreateSubmission()
  const [open, setOpen] = useState(false)
  const [userId, setUserId] = useState('')
  const [teamId, setTeamId] = useState('')
  const [challengeId, setChallengeId] = useState('')
  const [flag, setFlag] = useState('')
  const [isCorrect, setIsCorrect] = useState(false)

  if (!open) {
    return (
      <button
        onClick={() => setOpen(true)}
        className="self-start px-3 py-1.5 text-sm rounded-[var(--radius-md)] border border-space-border text-text-secondary hover:text-text-primary hover:border-cosmic-blue/50 transition-colors"
      >
        + Create submission
      </button>
    )
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    await create({
      user_id: userId.trim(),
      challenge_id: challengeId.trim(),
      submitted_flag: flag,
      is_correct: isCorrect,
      ...(teamId.trim() ? { team_id: teamId.trim() } : {}),
    })
    setUserId('')
    setTeamId('')
    setChallengeId('')
    setFlag('')
    setIsCorrect(false)
    setOpen(false)
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="rounded-[var(--radius-lg)] border border-space-border bg-space-surface p-4 flex flex-col gap-3"
    >
      <p className="text-sm font-semibold text-text-primary">Create Submission</p>
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div className="flex flex-col gap-1">
          <label className="text-xs text-text-muted">User ID *</label>
          <input
            className={inputCls}
            value={userId}
            onChange={(e) => setUserId(e.target.value)}
            required
            placeholder="uuid"
          />
        </div>
        <div className="flex flex-col gap-1">
          <label className="text-xs text-text-muted">Team ID</label>
          <input
            className={inputCls}
            value={teamId}
            onChange={(e) => setTeamId(e.target.value)}
            placeholder="uuid (optional)"
          />
        </div>
        <div className="flex flex-col gap-1">
          <label className="text-xs text-text-muted">Challenge ID *</label>
          <input
            className={inputCls}
            value={challengeId}
            onChange={(e) => setChallengeId(e.target.value)}
            required
            placeholder="uuid"
          />
        </div>
        <div className="flex flex-col gap-1">
          <label className="text-xs text-text-muted">Submitted flag *</label>
          <input
            className={inputCls}
            value={flag}
            onChange={(e) => setFlag(e.target.value)}
            required
            placeholder="flag{...}"
          />
        </div>
      </div>
      <label className="flex items-center gap-2 cursor-pointer">
        <input
          type="checkbox"
          checked={isCorrect}
          onChange={(e) => setIsCorrect(e.target.checked)}
          className="rounded border-space-border"
        />
        <span className="text-sm text-text-secondary">Mark as correct</span>
      </label>
      <div className="flex gap-2 justify-end">
        <button
          type="button"
          onClick={() => setOpen(false)}
          className="px-3 py-1.5 text-sm text-text-muted hover:text-text-primary transition-colors"
        >
          Cancel
        </button>
        <button
          type="submit"
          disabled={isPending}
          className="px-4 py-1.5 text-sm rounded-[var(--radius-md)] bg-cosmic-blue text-white hover:bg-cosmic-blue/90 disabled:opacity-50 transition-colors"
        >
          {isPending ? 'Creating…' : 'Create'}
        </button>
      </div>
    </form>
  )
}

// ---------------------------------------------------------------------------
// Flag cell - expandable
// ---------------------------------------------------------------------------
function FlagCell({ flag }: { flag?: string }) {
  const [expanded, setExpanded] = useState(false)
  if (!flag) return <span className="text-text-muted italic">-</span>
  return (
    <button
      onClick={() => setExpanded((v) => !v)}
      className="text-left text-xs font-mono text-text-secondary hover:text-text-primary transition-colors"
    >
      {expanded ? flag : `${flag.slice(0, 20)}${flag.length > 20 ? '…' : ''}`}
    </button>
  )
}

// ---------------------------------------------------------------------------
// Row skeleton
// ---------------------------------------------------------------------------
function RowSkeleton() {
  return (
    <>
      {Array.from({ length: 8 }).map((_, i) => (
        <tr key={i} className="border-b border-space-border">
          {Array.from({ length: 7 }).map((__, j) => (
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
export function AdminSubmissionsPage() {
  const [page, setPage] = useState(1)
  const [tab, setTab] = useState<FilterTab>('all')
  const [search, setSearch] = useState('')
  const debouncedSearch = useDebounce(search, 300)

  const { data, isLoading, isFetching } = useAdminSubmissions(page)
  const { mutate: markCorrect, isPending: marking } = useMarkCorrect()
  const { mutate: deleteSubmission, isPending: deleting } = useDeleteSubmission()

  const submissions = data?.data ?? []
  const meta = data?.meta

  const filtered = submissions.filter((s) => {
    if (tab === 'correct' && !s.is_correct) return false
    if (tab === 'incorrect' && s.is_correct) return false
    if (debouncedSearch) {
      const q = debouncedSearch.toLowerCase()
      return (
        s.username?.toLowerCase().includes(q) ||
        s.team_name?.toLowerCase().includes(q) ||
        s.challenge_title?.toLowerCase().includes(q) ||
        s.submitted_flag?.toLowerCase().includes(q)
      )
    }
    return true
  })

  const TABS: { value: FilterTab; label: string }[] = [
    { value: 'all', label: 'All' },
    { value: 'correct', label: 'Correct' },
    { value: 'incorrect', label: 'Incorrect' },
  ]

  return (
    <div className="flex flex-col gap-5">
      <h1
        className="text-2xl font-black text-text-primary"
        style={{ fontFamily: 'var(--font-display)' }}
      >
        Submissions
      </h1>

      <CreateSubmissionForm />

      {/* Filter bar */}
      <div className="flex flex-col sm:flex-row items-start sm:items-center gap-3">
        {/* Tab filter */}
        <div className="flex rounded-[var(--radius-md)] border border-space-border overflow-hidden shrink-0">
          {TABS.map((t) => (
            <button
              key={t.value}
              onClick={() => {
                setTab(t.value)
                setPage(1)
              }}
              className={`px-3 py-1.5 text-sm font-medium transition-colors ${
                tab === t.value
                  ? 'bg-cosmic-blue text-white'
                  : 'bg-space-dark text-text-secondary hover:text-text-primary hover:bg-space-border/30'
              }`}
            >
              {t.label}
            </button>
          ))}
        </div>

        {/* Search */}
        <input
          type="search"
          value={search}
          onChange={(e) => {
            setSearch(e.target.value)
            setPage(1)
          }}
          placeholder="Search user, team, challenge, flag…"
          className="flex-1 min-w-0 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 py-1.5 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
        />

        {isFetching && !isLoading && (
          <span className="text-xs text-text-muted animate-pulse shrink-0">Refreshing…</span>
        )}
      </div>

      {/* Table */}
      <div
        className={`overflow-x-auto rounded-[var(--radius-lg)] border border-space-border transition-opacity ${isFetching && !isLoading ? 'opacity-60' : ''}`}
      >
        <table className="w-full min-w-[780px]">
          <thead>
            <tr className="border-b border-space-border bg-space-dark/60">
              <th className="px-3 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
                User
              </th>
              <th className="px-3 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide hidden sm:table-cell">
                Team
              </th>
              <th className="px-3 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
                Challenge
              </th>
              <th className="px-3 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
                Flag
              </th>
              <th className="px-3 py-2.5 text-center text-xs font-semibold text-text-muted uppercase tracking-wide">
                Status
              </th>
              <th className="px-3 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide hidden md:table-cell">
                Time
              </th>
              <th className="px-3 py-2.5 text-right text-xs font-semibold text-text-muted uppercase tracking-wide">
                Actions
              </th>
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              <RowSkeleton />
            ) : filtered.length === 0 ? (
              <tr>
                <td colSpan={7}>
                  <EmptyState
                    message={
                      submissions.length === 0
                        ? 'No submissions yet.'
                        : 'No submissions match your filters.'
                    }
                  />
                </td>
              </tr>
            ) : (
              filtered.map((s) => (
                <tr
                  key={s.id}
                  className="border-b border-space-border last:border-b-0 hover:bg-space-border/20 transition-colors"
                >
                  <td className="px-3 py-2.5 text-sm text-text-primary font-medium">
                    {s.username ?? <span className="text-text-muted italic">-</span>}
                  </td>
                  <td className="px-3 py-2.5 text-sm text-text-secondary hidden sm:table-cell">
                    {s.team_name ?? <span className="text-text-muted italic">-</span>}
                  </td>
                  <td className="px-3 py-2.5">
                    <div className="flex flex-col">
                      <span className="text-sm text-text-primary">{s.challenge_title ?? '-'}</span>
                      {s.challenge_category && (
                        <span className="text-xs text-text-muted">{s.challenge_category}</span>
                      )}
                    </div>
                  </td>
                  <td className="px-3 py-2.5 max-w-[180px]">
                    <FlagCell
                      {...(s.submitted_flag !== undefined ? { flag: s.submitted_flag } : {})}
                    />
                  </td>
                  <td className="px-3 py-2.5 text-center">
                    <Badge variant={s.is_correct ? 'green' : 'red'}>
                      {s.is_correct ? 'Correct' : 'Incorrect'}
                    </Badge>
                  </td>
                  <td className="px-3 py-2.5 text-xs text-text-muted hidden md:table-cell whitespace-nowrap">
                    {s.created_at ? new Date(s.created_at).toLocaleString() : '-'}
                  </td>
                  <td className="px-3 py-2.5 text-right">
                    <div className="flex items-center justify-end gap-2">
                      {!s.is_correct && (
                        <button
                          onClick={() => s.id && markCorrect({ id: s.id, is_correct: true })}
                          disabled={marking || deleting}
                          className="text-xs text-nebula-green hover:underline disabled:opacity-40 whitespace-nowrap"
                        >
                          Mark correct
                        </button>
                      )}
                      {s.is_correct && (
                        <button
                          onClick={() => s.id && markCorrect({ id: s.id, is_correct: false })}
                          disabled={marking || deleting}
                          className="text-xs text-solar-yellow hover:underline disabled:opacity-40 whitespace-nowrap"
                        >
                          Mark incorrect
                        </button>
                      )}
                      <button
                        onClick={() => s.id && deleteSubmission(s.id)}
                        disabled={marking || deleting}
                        className="text-xs text-nova-red hover:underline disabled:opacity-40"
                      >
                        Delete
                      </button>
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {meta && (meta.total_pages ?? 1) > 1 && (
        <div className="flex items-center justify-between text-sm">
          <span className="text-text-muted">
            Page {meta.page ?? page} of {meta.total_pages ?? 1}
            {meta.total !== undefined && (
              <span className="ml-2 text-text-muted/60">({meta.total.toLocaleString()} total)</span>
            )}
          </span>
          <div className="flex gap-2">
            <button
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page <= 1 || isFetching}
              className="px-3 py-1 rounded-[var(--radius-md)] border border-space-border text-text-secondary hover:text-text-primary hover:bg-space-border/30 disabled:opacity-40 transition-colors"
            >
              ‹ Prev
            </button>
            <button
              onClick={() => setPage((p) => p + 1)}
              disabled={page >= (meta.total_pages ?? 1) || isFetching}
              className="px-3 py-1 rounded-[var(--radius-md)] border border-space-border text-text-secondary hover:text-text-primary hover:bg-space-border/30 disabled:opacity-40 transition-colors"
            >
              Next ›
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
