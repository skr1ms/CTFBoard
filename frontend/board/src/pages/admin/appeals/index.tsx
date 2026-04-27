import { api } from '@/shared/api/client'
import type { components } from '@/shared/api/schema.d'
import { EmptyState } from '@/shared/ui/empty-state'
import { Pagination } from '@/shared/ui/pagination'
import { Skeleton } from '@/shared/ui/skeleton'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Fragment, useState } from 'react'
import { toast } from 'sonner'

type Appeal = components['schemas']['BanAppealResponse']
type Decision = 'resolved' | 'rejected'
type StatusFilter = 'all' | 'pending' | 'resolved' | 'rejected'

const STATUS_TABS: { value: StatusFilter; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'pending', label: 'Pending' },
  { value: 'resolved', label: 'Resolved' },
  { value: 'rejected', label: 'Rejected' },
]

function decisionBadge(decision?: string) {
  const base = 'inline-flex items-center px-2 py-0.5 rounded text-xs font-medium'
  if (decision === 'resolved') return `${base} bg-green-500/15 text-green-400`
  if (decision === 'rejected') return `${base} bg-red-500/15 text-red-400`
  return `${base} bg-yellow-500/15 text-yellow-400`
}

function formatDate(iso?: string) {
  if (!iso) return '-'
  return new Date(iso).toLocaleString()
}

function useAdminAppeals(status: StatusFilter, page: number, perPage: number) {
  return useQuery({
    queryKey: ['admin', 'appeals', status, page],
    queryFn: async () => {
      const { data, error } = await api.GET('/admin/appeals', {
        params: {
          query: {
            status: status === 'all' ? undefined : status,
            page,
            per_page: perPage,
          },
        },
      })
      if (error) throw error
      return data
    },
    staleTime: 10_000,
  })
}

function useReviewAppeal() {
  const qc = useQueryClient()

  return useMutation({
    mutationFn: async ({
      id,
      decision,
      adminResponse,
    }: {
      id: string
      decision: Decision
      adminResponse?: string
    }) => {
      const { error } = await api.PATCH('/admin/appeals/{ID}', {
        params: { path: { ID: id } },
        body: {
          decision,
          ...(adminResponse !== undefined ? { admin_response: adminResponse } : {}),
        },
      })
      if (error) throw error
    },
    onSuccess() {
      void qc.invalidateQueries({ queryKey: ['admin', 'appeals'] })
      toast.success('Appeal reviewed')
    },
    onError(err: { message?: string }) {
      toast.error(err.message ?? 'Failed to review appeal')
    },
  })
}

function ReviewForm({ appeal, onCancel }: { appeal: Appeal; onCancel: () => void }) {
  const [decision, setDecision] = useState<Decision>('resolved')
  const [adminResponse, setAdminResponse] = useState('')
  const reviewMutation = useReviewAppeal()

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!appeal.id) return
    const trimmed = adminResponse.trim()
    reviewMutation.mutate(
      { id: appeal.id, decision, ...(trimmed ? { adminResponse: trimmed } : {}) },
      { onSuccess: onCancel },
    )
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="flex flex-col gap-3 p-3 bg-space-bg rounded border border-space-border mt-2"
    >
      <div className="flex gap-3">
        <label className="flex items-center gap-1.5 cursor-pointer text-sm">
          <input
            type="radio"
            name="decision"
            value="resolved"
            checked={decision === 'resolved'}
            onChange={() => setDecision('resolved')}
            className="accent-green-500"
          />
          <span className="text-green-400">Resolve (unban)</span>
        </label>
        <label className="flex items-center gap-1.5 cursor-pointer text-sm">
          <input
            type="radio"
            name="decision"
            value="rejected"
            checked={decision === 'rejected'}
            onChange={() => setDecision('rejected')}
            className="accent-red-500"
          />
          <span className="text-red-400">Reject</span>
        </label>
      </div>
      <textarea
        value={adminResponse}
        onChange={(e) => setAdminResponse(e.target.value)}
        placeholder="Optional admin response to the user..."
        rows={2}
        className="w-full rounded border border-space-border bg-space-card px-3 py-2 text-sm text-text-primary placeholder:text-text-muted resize-none focus:outline-none focus:ring-1 focus:ring-cosmic-blue"
      />
      <div className="flex gap-2 justify-end">
        <button
          type="button"
          onClick={onCancel}
          className="px-3 py-1.5 text-xs rounded border border-space-border text-text-muted hover:text-text-primary transition-colors"
        >
          Cancel
        </button>
        <button
          type="submit"
          disabled={reviewMutation.isPending}
          className="px-3 py-1.5 text-xs rounded bg-cosmic-blue text-white hover:opacity-90 disabled:opacity-50 transition-opacity"
        >
          {reviewMutation.isPending ? 'Submitting…' : 'Submit'}
        </button>
      </div>
    </form>
  )
}

export function AdminAppealsPage() {
  const [status, setStatus] = useState<StatusFilter>('pending')
  const [page, setPage] = useState(1)
  const [reviewingId, setReviewingId] = useState<string | null>(null)
  const perPage = 25

  const { data, isLoading } = useAdminAppeals(status, page, perPage)

  const rows: Appeal[] = data?.appeals ?? []
  const meta = data?.meta
  const totalPages = meta ? Math.ceil((meta.total ?? 0) / perPage) : 1

  function handleTabChange(s: StatusFilter) {
    setStatus(s)
    setPage(1)
    setReviewingId(null)
  }

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1
          className="text-2xl font-bold text-text-primary"
          style={{ fontFamily: 'var(--font-display)' }}
        >
          Ban Appeals
        </h1>
        <p className="text-sm text-text-muted mt-1">Review and resolve user ban appeals.</p>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 border-b border-space-border">
        {STATUS_TABS.map((tab) => (
          <button
            key={tab.value}
            onClick={() => handleTabChange(tab.value)}
            className={`px-4 py-2 text-sm font-medium transition-colors border-b-2 -mb-px ${
              status === tab.value
                ? 'border-cosmic-blue text-cosmic-blue'
                : 'border-transparent text-text-muted hover:text-text-primary'
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {isLoading ? (
        <div className="flex flex-col gap-2">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      ) : rows.length === 0 ? (
        <EmptyState
          icon={<span className="text-4xl">⚖️</span>}
          message={`No ${status === 'all' ? '' : status} appeals found.`}
        />
      ) : (
        <>
          <div className="rounded-[var(--radius-lg)] border border-space-border overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-space-border bg-space-card">
                    <th className="text-left px-4 py-3 text-xs font-semibold text-text-muted uppercase tracking-wide">
                      User ID
                    </th>
                    <th className="text-left px-4 py-3 text-xs font-semibold text-text-muted uppercase tracking-wide">
                      Message
                    </th>
                    <th className="text-left px-4 py-3 text-xs font-semibold text-text-muted uppercase tracking-wide">
                      Status
                    </th>
                    <th className="text-left px-4 py-3 text-xs font-semibold text-text-muted uppercase tracking-wide">
                      Submitted
                    </th>
                    <th className="text-left px-4 py-3 text-xs font-semibold text-text-muted uppercase tracking-wide">
                      Reviewed
                    </th>
                    <th className="text-left px-4 py-3 text-xs font-semibold text-text-muted uppercase tracking-wide">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((row, i) => (
                    <Fragment key={row.id ?? `${row.user_id ?? '?'}-${row.created_at ?? i}`}>
                      <tr
                        className={`border-b border-space-border last:border-b-0 ${
                          i % 2 === 1 ? 'bg-space-card/30' : ''
                        }`}
                      >
                        <td className="px-4 py-3 font-mono text-xs text-text-secondary truncate max-w-[120px]">
                          {row.user_id ?? '-'}
                        </td>
                        <td className="px-4 py-3 text-text-secondary max-w-[280px]">
                          <p className="truncate">{row.message ?? '-'}</p>
                          {row.admin_response && (
                            <p className="text-xs text-text-muted truncate mt-0.5">
                              Admin: {row.admin_response}
                            </p>
                          )}
                        </td>
                        <td className="px-4 py-3">
                          <span className={decisionBadge(row.decision)}>
                            {row.decision ?? 'pending'}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-text-muted whitespace-nowrap text-xs">
                          {formatDate(row.created_at)}
                        </td>
                        <td className="px-4 py-3 text-text-muted whitespace-nowrap text-xs">
                          {formatDate(row.reviewed_at)}
                        </td>
                        <td className="px-4 py-3">
                          {row.decision === 'pending' && row.id && (
                            <button
                              onClick={() =>
                                setReviewingId(reviewingId === row.id ? null : (row.id ?? null))
                              }
                              className="px-3 py-1 text-xs rounded border border-space-border text-text-muted hover:text-text-primary transition-colors"
                            >
                              {reviewingId === row.id ? 'Cancel' : 'Review'}
                            </button>
                          )}
                        </td>
                      </tr>
                      {reviewingId === row.id && row.id && (
                        <tr className="border-b border-space-border bg-space-card/10">
                          <td colSpan={6} className="px-4 py-2">
                            <ReviewForm appeal={row} onCancel={() => setReviewingId(null)} />
                          </td>
                        </tr>
                      )}
                    </Fragment>
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
