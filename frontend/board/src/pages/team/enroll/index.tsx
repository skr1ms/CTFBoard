import { useCompetitionStatus } from '@/features/competition/useCompetitionStatus'
import { api, apiErrorMessage, isApiError } from '@/shared/api/client'
import { useAuthStore } from '@/shared/stores/authStore'
import type { components } from '@/shared/api/schema.d'
import { validateTeamName } from '@/shared/lib/validation'
import { Button } from '@/shared/ui/button'
import { ConfirmationDialog } from '@/shared/ui/confirmation-dialog'
import { useMutation } from '@tanstack/react-query'
import { type FormEvent, useEffect, useRef, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router'
import { toast } from 'sonner'

type AffectedData = components['schemas']['AffectedData']

interface ConfirmState {
  reason: string
  affected?: AffectedData
}

function formatAffected(d?: AffectedData): string {
  if (!d) return ''
  const parts: string[] = []
  if (d.solve_count) parts.push(`${d.solve_count} solve${d.solve_count !== 1 ? 's' : ''}`)
  if (d.points) parts.push(`${d.points} pts`)
  if (d.hint_unlock_count)
    parts.push(`${d.hint_unlock_count} hint${d.hint_unlock_count !== 1 ? 's' : ''}`)
  return parts.length ? `This will reset: ${parts.join(', ')}.` : ''
}

// ---------------------------------------------------------------------------
// Create card
// ---------------------------------------------------------------------------
function CreateCard() {
  const navigate = useNavigate()
  const [name, setName] = useState('')
  const [nameError, setNameError] = useState<string | null>(null)
  const [confirm, setConfirm] = useState<ConfirmState | null>(null)

  const { mutate, isPending } = useMutation({
    mutationFn: async ({ teamName, confirmReset }: { teamName: string; confirmReset: boolean }) => {
      const { data, error } = await api.POST('/teams', {
        body: { name: teamName, ...(confirmReset ? { confirm_reset: true } : {}) },
      })
      if (error) throw error
      if (data && 'reason' in data) {
        const conf = data as components['schemas']['ConfirmationRequired']
        return { needsConfirm: true as const, reason: conf.reason, affected: conf.affected_data }
      }
      return { needsConfirm: false as const }
    },
    onSuccess: async (result) => {
      if (result.needsConfirm) {
        setConfirm({
          reason: result.reason,
          ...(result.affected ? { affected: result.affected } : {}),
        })
        return
      }
      const { data: me } = await api.GET('/auth/me')
      if (me) useAuthStore.getState().setUser(me)
      toast.success('Team created!')
      navigate('/team')
    },
    onError: (err) => {
      toast.error(apiErrorMessage(err, 'Failed to create team'))
    },
  })

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault()
    const err = validateTeamName(name.trim())
    if (err) {
      setNameError(err)
      return
    }
    setNameError(null)
    mutate({ teamName: name.trim(), confirmReset: false })
  }

  return (
    <>
      <div className="flex flex-col gap-5 p-6 rounded-[var(--radius-lg)] border border-space-border bg-space-card hover:border-cosmic-blue/40 transition-colors">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-[var(--radius-md)] bg-cosmic-blue/10 border border-cosmic-blue/30 flex items-center justify-center text-cosmic-blue shrink-0">
            <svg className="w-5 h-5" viewBox="0 0 20 20" fill="currentColor">
              <path d="M10 9a3 3 0 1 0 0-6 3 3 0 0 0 0 6ZM6 8a2 2 0 1 1-4 0 2 2 0 0 1 4 0ZM1.49 15.326a.78.78 0 0 1-.358-.442 3 3 0 0 1 4.308-3.516 6.484 6.484 0 0 0-1.905 3.959c-.023.222-.014.442.025.654a4.97 4.97 0 0 1-2.07-.655ZM16.44 15.98a4.97 4.97 0 0 0 2.07-.654.78.78 0 0 0 .357-.442 3 3 0 0 0-4.308-3.517 6.484 6.484 0 0 1 1.907 3.96 2.32 2.32 0 0 1-.026.654ZM18 8a2 2 0 1 1-4 0 2 2 0 0 1 4 0ZM5.304 16.19a.844.844 0 0 1-.277-.71 5 5 0 0 1 9.947 0 .843.843 0 0 1-.277.71A6.975 6.975 0 0 1 10 18a6.974 6.974 0 0 1-4.696-1.81Z" />
            </svg>
          </div>
          <div>
            <h3 className="font-semibold text-text-primary">Create a Team</h3>
            <p className="text-xs text-text-muted mt-0.5">Start your own team and invite others</p>
          </div>
        </div>

        <form onSubmit={handleSubmit} className="flex flex-col gap-3">
          <div className="flex flex-col gap-1">
            <input
              data-testid="team-create-name"
              type="text"
              value={name}
              onChange={(e) => {
                setName(e.target.value)
                setNameError(null)
              }}
              placeholder="Team name"
              maxLength={50}
              required
              className="h-10 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/50 focus:border-cosmic-blue"
            />
            {nameError && <p className="text-xs text-nova-red">{nameError}</p>}
          </div>
          <Button
            data-testid="team-create-submit"
            type="submit"
            variant="primary"
            loading={isPending}
            disabled={!name.trim()}
          >
            Create Team
          </Button>
        </form>
      </div>

      <ConfirmationDialog
        open={!!confirm}
        onClose={() => setConfirm(null)}
        onConfirm={() => {
          setConfirm(null)
          mutate({ teamName: name.trim(), confirmReset: true })
        }}
        title="Reset progress?"
        message={
          <span>
            {confirm?.reason}
            {confirm?.affected && (
              <span className="block mt-1 text-xs text-text-muted">
                {formatAffected(confirm.affected)}
              </span>
            )}
          </span>
        }
        confirmLabel="Yes, reset & create"
        danger
        loading={isPending}
      />
    </>
  )
}

// ---------------------------------------------------------------------------
// Join card
// ---------------------------------------------------------------------------
function JoinCard() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const initialToken = searchParams.get('token') ?? ''
  const [token, setToken] = useState(initialToken)
  const [confirm, setConfirm] = useState<ConfirmState | null>(null)
  const hasAutoSubmitted = useRef(false)

  const { mutate, isPending } = useMutation({
    mutationFn: async ({
      inviteToken,
      confirmReset,
    }: {
      inviteToken: string
      confirmReset: boolean
    }) => {
      const { data, error } = await api.POST('/teams/join', {
        body: { invite_token: inviteToken, ...(confirmReset ? { confirm_reset: true } : {}) },
      })
      if (error) throw error
      if (data && 'reason' in data) {
        const conf = data as components['schemas']['ConfirmationRequired']
        return { needsConfirm: true as const, reason: conf.reason, affected: conf.affected_data }
      }
      return { needsConfirm: false as const }
    },
    onSuccess: async (result) => {
      if (result.needsConfirm) {
        setConfirm({
          reason: result.reason,
          ...(result.affected ? { affected: result.affected } : {}),
        })
        return
      }
      const { data: me } = await api.GET('/auth/me')
      if (me) useAuthStore.getState().setUser(me)
      toast.success('Joined team!')
      navigate('/team')
    },
    onError: (err) => {
      const code = isApiError(err) ? err.code : undefined
      const violations = (err as { violations?: Array<{ field: string; message: string }> })
        .violations
      const hasUUIDViolation = violations?.some((v) => v.message?.toLowerCase().includes('uuid'))
      const msg =
        code === 'TEAM_NOT_FOUND'
          ? "Invalid invite code. Make sure you copied it from your captain's team page."
          : code === 'TEAM_FULL'
            ? 'This team is full.'
            : hasUUIDViolation
              ? 'Invite code must be a 36-character UUID (e.g. xxxxxxxx-xxxx-…). Your API token will not work here.'
              : apiErrorMessage(err, 'Failed to join team')
      toast.error(msg)
    },
  })

  // Auto-submit when navigating to /team/enroll?token=<uuid> via a deep link
  useEffect(() => {
    if (initialToken && !hasAutoSubmitted.current) {
      hasAutoSubmitted.current = true
      mutate({ inviteToken: initialToken, confirmReset: false })
    }
  }, [initialToken, mutate])

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault()
    if (!token.trim()) return
    mutate({ inviteToken: token.trim(), confirmReset: false })
  }

  return (
    <>
      <div className="flex flex-col gap-5 p-6 rounded-[var(--radius-lg)] border border-space-border bg-space-card hover:border-nebula-purple/40 transition-colors">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-[var(--radius-md)] bg-nebula-purple/10 border border-nebula-purple/30 flex items-center justify-center text-nebula-purple shrink-0">
            <svg className="w-5 h-5" viewBox="0 0 20 20" fill="currentColor">
              <path
                fillRule="evenodd"
                d="M17.707 9.293a1 1 0 0 1 0 1.414l-7 7a1 1 0 0 1-1.414 0l-7-7A.997.997 0 0 1 2 10V5a3 3 0 0 1 3-3h5c.256 0 .512.098.707.293l7 7ZM5 6a1 1 0 1 0 0-2 1 1 0 0 0 0 2Z"
                clipRule="evenodd"
              />
            </svg>
          </div>
          <div>
            <h3 className="font-semibold text-text-primary">Join a Team</h3>
            <p className="text-xs text-text-muted mt-0.5">
              Enter the invite code from your captain's team page
            </p>
          </div>
        </div>

        <form onSubmit={handleSubmit} className="flex flex-col gap-3">
          <div className="flex flex-col gap-1">
            <input
              data-testid="team-join-token"
              type="text"
              value={token}
              onChange={(e) => setToken(e.target.value)}
              placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
              required
              className="h-10 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-nebula-purple/50 focus:border-nebula-purple font-mono"
            />
            <p className="text-[11px] text-text-muted">
              This is different from your API token - ask your captain for their 36-character team
              invite code.
            </p>
          </div>
          <Button
            data-testid="team-join-submit"
            type="submit"
            variant="secondary"
            loading={isPending}
            disabled={!token.trim()}
          >
            Join Team
          </Button>
        </form>
      </div>

      <ConfirmationDialog
        open={!!confirm}
        onClose={() => setConfirm(null)}
        onConfirm={() => {
          setConfirm(null)
          mutate({ inviteToken: token.trim(), confirmReset: true })
        }}
        title="Reset progress?"
        message={
          <span>
            {confirm?.reason}
            {confirm?.affected && (
              <span className="block mt-1 text-xs text-text-muted">
                {formatAffected(confirm.affected)}
              </span>
            )}
          </span>
        }
        confirmLabel="Yes, reset & join"
        danger
        loading={isPending}
      />
    </>
  )
}

// ---------------------------------------------------------------------------
// Solo card
// ---------------------------------------------------------------------------
function SoloCard() {
  const navigate = useNavigate()
  const [confirm, setConfirm] = useState<ConfirmState | null>(null)

  const { mutate, isPending } = useMutation({
    mutationFn: async ({ confirmReset }: { confirmReset: boolean }) => {
      const { data, error } = await api.POST('/teams/solo', {
        body: confirmReset ? { confirm_reset: true } : {},
      })
      if (error) throw error
      if (data && 'reason' in data) {
        const conf = data as components['schemas']['ConfirmationRequired']
        return { needsConfirm: true as const, reason: conf.reason, affected: conf.affected_data }
      }
      return { needsConfirm: false as const }
    },
    onSuccess: async (result) => {
      if (result.needsConfirm) {
        setConfirm({
          reason: result.reason,
          ...(result.affected ? { affected: result.affected } : {}),
        })
        return
      }
      const { data: me } = await api.GET('/auth/me')
      if (me) useAuthStore.getState().setUser(me)
      toast.success('Solo mode started!')
      navigate('/team')
    },
    onError: (err) => {
      toast.error(apiErrorMessage(err, 'Failed to start solo mode'))
    },
  })

  return (
    <>
      <div className="flex flex-col gap-5 p-6 rounded-[var(--radius-lg)] border border-space-border bg-space-card hover:border-stellar-green/40 transition-colors">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-[var(--radius-md)] bg-stellar-green/10 border border-stellar-green/30 flex items-center justify-center text-stellar-green shrink-0">
            <svg className="w-5 h-5" viewBox="0 0 20 20" fill="currentColor">
              <path d="M10 9a3 3 0 1 0 0-6 3 3 0 0 0 0 6ZM6 8a2 2 0 1 1-4 0 2 2 0 0 1 4 0ZM1.49 15.326a.78.78 0 0 1-.358-.442 3 3 0 0 1 4.308-3.516 6.484 6.484 0 0 0-1.905 3.959c-.023.222-.014.442.025.654a4.97 4.97 0 0 1-2.07-.655ZM16.44 15.98a4.97 4.97 0 0 0 2.07-.654.78.78 0 0 0 .357-.442 3 3 0 0 0-4.308-3.517 6.484 6.484 0 0 1 1.907 3.96 2.32 2.32 0 0 1-.026.654ZM18 8a2 2 0 1 1-4 0 2 2 0 0 1 4 0Z" />
            </svg>
          </div>
          <div>
            <h3 className="font-semibold text-text-primary">Go Solo</h3>
            <p className="text-xs text-text-muted mt-0.5">Compete individually without a team</p>
          </div>
        </div>

        <Button
          data-testid="team-solo-submit"
          variant="secondary"
          loading={isPending}
          onClick={() => mutate({ confirmReset: false })}
        >
          Start Solo
        </Button>
      </div>

      <ConfirmationDialog
        open={!!confirm}
        onClose={() => setConfirm(null)}
        onConfirm={() => {
          setConfirm(null)
          mutate({ confirmReset: true })
        }}
        title="Reset progress?"
        message={
          <span>
            {confirm?.reason}
            {confirm?.affected && (
              <span className="block mt-1 text-xs text-text-muted">
                {formatAffected(confirm.affected)}
              </span>
            )}
          </span>
        }
        confirmLabel="Yes, reset & go solo"
        danger
        loading={isPending}
      />
    </>
  )
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------
export function TeamEnrollPage() {
  const { data: competitionStatus } = useCompetitionStatus()
  const mode = (competitionStatus as (typeof competitionStatus & { mode?: string }) | undefined)
    ?.mode

  const showCreate = mode !== 'solo_only'
  const showJoin = mode !== 'solo_only'
  const showSolo = mode !== 'teams_only'

  const visibleCount = [showCreate, showJoin, showSolo].filter(Boolean).length

  return (
    <div className="flex flex-col items-center justify-center min-h-[60vh] px-4 py-8">
      <div className="w-full max-w-2xl flex flex-col gap-6">
        <div className="text-center">
          <h1
            className="text-3xl font-black text-text-primary"
            style={{ fontFamily: 'var(--font-display)' }}
          >
            Join the competition
          </h1>
          <p className="text-text-muted mt-2 text-sm">
            Create a team, join an existing one, or compete solo.
          </p>
        </div>

        <div
          className={`grid gap-4 ${
            visibleCount === 1
              ? 'grid-cols-1 max-w-sm mx-auto w-full'
              : visibleCount === 2
                ? 'grid-cols-1 sm:grid-cols-2'
                : 'grid-cols-1 sm:grid-cols-3'
          }`}
        >
          {showCreate && <CreateCard />}
          {showJoin && <JoinCard />}
          {showSolo && <SoloCard />}
        </div>
      </div>
    </div>
  )
}
