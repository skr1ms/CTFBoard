import { useCreateAppeal, useMyAppeals } from '@/features/banned/useAppeal'
import { useAuthStore } from '@/shared/stores/authStore'
import { Button } from '@/shared/ui/button'
import { Skeleton } from '@/shared/ui/skeleton'
import { useState } from 'react'

const APPEAL_MIN_LENGTH = 20

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

export function BannedPage() {
  const user = useAuthStore((s) => s.user)
  const logout = useAuthStore((s) => s.logout)
  const banStatus = user?.ban_status

  const { data: appeals = [], isLoading } = useMyAppeals()
  const createAppeal = useCreateAppeal()

  const [message, setMessage] = useState('')
  const canSubmit = message.trim().length >= APPEAL_MIN_LENGTH && !createAppeal.isPending

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    createAppeal.mutate(message.trim(), { onSuccess: () => setMessage('') })
  }

  return (
    <div className="min-h-screen bg-space-bg flex flex-col items-center justify-start py-16 px-4">
      {/* Header */}
      <div className="w-full max-w-2xl flex items-center justify-between mb-8">
        <span
          className="text-xl font-bold text-text-primary"
          style={{ fontFamily: 'var(--font-display)' }}
        >
          AstroCTF
        </span>
        <Button variant="ghost" size="sm" onClick={() => void logout()}>
          Sign out
        </Button>
      </div>

      {/* Ban card */}
      <div className="w-full max-w-2xl rounded-[var(--radius-lg)] border border-red-500/30 bg-red-500/5 p-6 mb-6">
        <div className="flex items-start gap-3">
          <span className="text-2xl leading-none mt-0.5">🚫</span>
          <div>
            <h1
              className="text-xl font-bold text-red-400 mb-1"
              style={{ fontFamily: 'var(--font-display)' }}
            >
              Account Banned
            </h1>
            {banStatus?.reason && (
              <p className="text-sm text-text-secondary mb-1">
                <span className="font-medium text-text-primary">Reason: </span>
                {banStatus.reason}
              </p>
            )}
            {banStatus?.banned_at && (
              <p className="text-xs text-text-muted">
                Banned at: {formatDate(banStatus.banned_at)}
              </p>
            )}
          </div>
        </div>
      </div>

      {/* Appeals list */}
      <div className="w-full max-w-2xl rounded-[var(--radius-lg)] border border-space-border bg-space-card p-6 mb-6">
        <h2 className="text-base font-semibold text-text-primary mb-4">Your Appeals</h2>
        {isLoading ? (
          <div className="flex flex-col gap-2">
            {Array.from({ length: 2 }).map((_, i) => (
              <Skeleton key={i} className="h-12 w-full" />
            ))}
          </div>
        ) : appeals.length === 0 ? (
          <p className="text-sm text-text-muted">You have not submitted any appeals yet.</p>
        ) : (
          <ul className="flex flex-col gap-3">
            {appeals.map((a) => (
              <li key={a.id} className="rounded border border-space-border bg-space-bg p-4 text-sm">
                <div className="flex items-center justify-between mb-2">
                  <span className={decisionBadge(a.decision)}>{a.decision ?? 'pending'}</span>
                  <span className="text-xs text-text-muted">{formatDate(a.created_at)}</span>
                </div>
                <p className="text-text-secondary mb-1">{a.message}</p>
                {a.admin_response && (
                  <p className="text-xs text-text-muted mt-2 border-t border-space-border pt-2">
                    <span className="font-medium text-text-primary">Admin: </span>
                    {a.admin_response}
                  </p>
                )}
              </li>
            ))}
          </ul>
        )}
      </div>

      {/* Appeal form */}
      {banStatus?.can_appeal && (
        <div className="w-full max-w-2xl rounded-[var(--radius-lg)] border border-space-border bg-space-card p-6">
          {banStatus.has_pending_appeal ? (
            <p className="text-sm text-text-muted">
              You have a pending appeal. We will notify you when it has been reviewed.
            </p>
          ) : (
            <>
              <h2 className="text-base font-semibold text-text-primary mb-4">Submit Appeal</h2>
              <form onSubmit={handleSubmit} className="flex flex-col gap-3">
                <textarea
                  value={message}
                  onChange={(e) => setMessage(e.target.value)}
                  placeholder={`Explain why you believe this ban should be lifted (min. ${APPEAL_MIN_LENGTH} characters)...`}
                  rows={4}
                  className="w-full rounded border border-space-border bg-space-bg px-3 py-2 text-sm text-text-primary placeholder:text-text-muted resize-none focus:outline-none focus:ring-1 focus:ring-cosmic-blue"
                />
                {message.length > 0 && message.trim().length < APPEAL_MIN_LENGTH && (
                  <p role="alert" className="text-xs text-red-400 -mt-1">
                    Appeal must be at least {APPEAL_MIN_LENGTH} characters.
                  </p>
                )}
                <div className="flex items-center justify-between">
                  <span className="text-xs text-text-muted">
                    {message.trim().length} / {APPEAL_MIN_LENGTH} min chars
                  </span>
                  <Button type="submit" disabled={!canSubmit} size="sm">
                    {createAppeal.isPending ? 'Submitting…' : 'Submit Appeal'}
                  </Button>
                </div>
              </form>
            </>
          )}
        </div>
      )}
    </div>
  )
}
