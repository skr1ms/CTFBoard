import { api } from '@/shared/api/client'
import type { components } from '@/shared/api/schema.d'
import { avatarSrc } from '@/shared/lib/utils'
import { Avatar } from '@/shared/ui/avatar'
import { Badge } from '@/shared/ui/badge'
import { EmptyState } from '@/shared/ui/empty-state'
import { Skeleton } from '@/shared/ui/skeleton'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useLocation, useNavigate, useParams } from 'react-router'

type AdminTeam = components['schemas']['AdminTeamResponse']
type AdminUser = components['schemas']['AdminUserResponse']
type ChallengeResponse = components['schemas']['ChallengeResponse']

// ---------------------------------------------------------------------------
// Hooks
// ---------------------------------------------------------------------------
function useAdminTeam(id: string, initial: AdminTeam | null) {
  return useQuery({
    queryKey: ['admin', 'team', id],
    queryFn: async () => {
      let page = 1
      while (true) {
        const { data, error } = await api.GET('/admin/teams', {
          params: { query: { page, per_page: 50 } },
        })
        if (error) throw error
        const rows = data?.data ?? []
        const found = rows.find((t) => t.id === id)
        if (found) return found
        const totalPages = data?.meta?.total_pages ?? 1
        if (page >= totalPages || rows.length === 0) return null
        page++
      }
    },
    placeholderData: initial ?? null,
    staleTime: 0,
  })
}

function useAdminTeamMembers(id: string) {
  return useQuery({
    queryKey: ['admin', 'team', id, 'members'],
    queryFn: async () => {
      const { data, error } = await api.GET('/admin/teams/{ID}/members', {
        params: { path: { ID: id } },
      })
      if (error) throw error
      return (data ?? []) as AdminUser[]
    },
    staleTime: 0,
  })
}

function useUpdateTeam(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (body: {
      name?: string
      captain_id?: string
      bracket_id?: string
      is_hidden?: boolean
    }) => {
      const { error } = await api.PATCH('/admin/teams/{ID}', {
        params: { path: { ID: id } },
        body,
      })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'team', id] })
      void qc.invalidateQueries({ queryKey: ['admin', 'teams'] })
    },
  })
}

function useAdminTeamSubmissions(id: string, page: number) {
  return useQuery({
    queryKey: ['admin', 'team', id, 'submissions', page],
    queryFn: async () => {
      const { data, error } = await api.GET('/admin/submissions/team/{teamID}', {
        params: { path: { teamID: id }, query: { page, per_page: 20 } },
      })
      if (error) throw error
      return data ?? { data: [], meta: undefined }
    },
    staleTime: 0,
  })
}

function useAdminTeamMissing(id: string) {
  return useQuery({
    queryKey: ['admin', 'team', id, 'missing'],
    queryFn: async () => {
      const { data, error } = await api.GET('/admin/teams/{ID}/missing-challenges', {
        params: { path: { ID: id } },
      })
      if (error) throw error
      return (data ?? []) as ChallengeResponse[]
    },
    staleTime: 0,
  })
}

// ---------------------------------------------------------------------------
// Tabs
// ---------------------------------------------------------------------------
type Tab = 'edit' | 'members' | 'missing' | 'submissions'
const TABS: Array<{ id: Tab; label: string }> = [
  { id: 'edit', label: 'Edit' },
  { id: 'members', label: 'Members' },
  { id: 'missing', label: 'Missing Challenges' },
  { id: 'submissions', label: 'Submissions' },
]

const inputCls =
  'w-full bg-space-dark border border-space-border rounded-[var(--radius)] px-3 py-1.5 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:border-cosmic-blue/60 transition-colors'

function EditTab({ team, teamId }: { team: AdminTeam; teamId: string }) {
  const { mutateAsync, isPending } = useUpdateTeam(teamId)
  const [name, setName] = useState(team.name ?? '')
  const [isHidden, setIsHidden] = useState(team.is_hidden ?? false)
  const [savedOk, setSavedOk] = useState(false)
  const [saveErr, setSaveErr] = useState<string | null>(null)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setSaveErr(null)
    setSavedOk(false)
    try {
      const trimmed = name.trim()
      await mutateAsync({ ...(trimmed && { name: trimmed }), is_hidden: isHidden })
      setSavedOk(true)
    } catch {
      setSaveErr('Failed to save changes.')
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4 max-w-md">
      <div className="flex flex-col gap-1">
        <label className="text-xs font-medium text-text-muted uppercase tracking-wide">Name</label>
        <input
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          className={inputCls}
          placeholder="Team name"
        />
      </div>
      <label className="flex items-center gap-2 cursor-pointer select-none">
        <input
          type="checkbox"
          checked={isHidden}
          onChange={(e) => setIsHidden(e.target.checked)}
          className="rounded border border-space-border bg-space-dark text-cosmic-blue focus:ring-0"
        />
        <span className="text-sm text-text-secondary">Hidden (not shown in public scoreboard)</span>
      </label>
      {savedOk && <p className="text-xs text-nebula-green">Saved.</p>}
      {saveErr && <p className="text-xs text-nova-red">{saveErr}</p>}
      <button
        type="submit"
        disabled={isPending}
        className="w-fit px-4 py-1.5 rounded-[var(--radius)] bg-cosmic-blue text-white text-sm font-medium hover:bg-cosmic-blue/80 disabled:opacity-50 transition-colors"
      >
        {isPending ? 'Saving…' : 'Save changes'}
      </button>
    </form>
  )
}

function SubmissionsTab({ teamId }: { teamId: string }) {
  const [page, setPage] = useState(1)
  const { data, isLoading } = useAdminTeamSubmissions(teamId, page)
  const rows = data?.data ?? []
  const meta = data?.meta

  if (isLoading) {
    return (
      <div className="flex flex-col gap-2">
        {[0, 1, 2].map((i) => (
          <Skeleton key={i} className="h-10 w-full" />
        ))}
      </div>
    )
  }

  if (rows.length === 0) return <EmptyState message="No submissions found." />

  return (
    <div className="flex flex-col gap-3">
      <div className="overflow-x-auto rounded-[var(--radius-lg)] border border-space-border">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-space-border bg-space-dark/60">
              <th className="px-4 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
                Challenge
              </th>
              <th className="px-4 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide hidden sm:table-cell">
                Flag
              </th>
              <th className="px-4 py-2.5 text-center text-xs font-semibold text-text-muted uppercase tracking-wide">
                Result
              </th>
              <th className="px-4 py-2.5 text-right text-xs font-semibold text-text-muted uppercase tracking-wide hidden md:table-cell">
                Time
              </th>
            </tr>
          </thead>
          <tbody>
            {rows.map((s) => (
              <tr
                key={s.id}
                className="border-b border-space-border last:border-b-0 hover:bg-space-border/20 transition-colors"
              >
                <td className="px-4 py-2.5 text-text-primary">
                  {s.challenge_title ?? s.challenge_id ?? '-'}
                </td>
                <td className="px-4 py-2.5 font-mono text-xs text-text-muted hidden sm:table-cell truncate max-w-[180px]">
                  {s.submitted_flag ?? '-'}
                </td>
                <td className="px-4 py-2.5 text-center">
                  <span
                    className={`px-1.5 py-0.5 rounded text-xs font-medium ${s.is_correct ? 'bg-nebula-green/10 text-nebula-green' : 'bg-nova-red/10 text-nova-red'}`}
                  >
                    {s.is_correct ? 'Correct' : 'Wrong'}
                  </span>
                </td>
                <td className="px-4 py-2.5 text-right text-text-muted text-xs hidden md:table-cell">
                  {s.created_at ? new Date(s.created_at).toLocaleString() : '-'}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {meta && (meta.total_pages ?? 1) > 1 && (
        <div className="flex items-center justify-between text-sm text-text-muted">
          <button
            disabled={page <= 1}
            onClick={() => setPage((p) => p - 1)}
            className="px-3 py-1 rounded border border-space-border hover:bg-space-border/30 disabled:opacity-40 transition-colors"
          >
            Prev
          </button>
          <span>
            Page {page} / {meta.total_pages ?? 1}
          </span>
          <button
            disabled={page >= (meta.total_pages ?? 1)}
            onClick={() => setPage((p) => p + 1)}
            className="px-3 py-1 rounded border border-space-border hover:bg-space-border/30 disabled:opacity-40 transition-colors"
          >
            Next
          </button>
        </div>
      )}
    </div>
  )
}

function MembersTab({ teamId }: { teamId: string }) {
  const { data: members = [], isLoading } = useAdminTeamMembers(teamId)

  if (isLoading) {
    return (
      <div className="flex flex-col gap-2">
        {[0, 1, 2].map((i) => (
          <Skeleton key={i} className="h-12 w-full" />
        ))}
      </div>
    )
  }

  if (members.length === 0) return <EmptyState message="No members." />

  return (
    <div className="overflow-x-auto rounded-[var(--radius-lg)] border border-space-border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-space-border bg-space-dark/60">
            <th className="px-4 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
              User
            </th>
            <th className="px-4 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide hidden sm:table-cell">
              Email
            </th>
            <th className="px-4 py-2.5 text-center text-xs font-semibold text-text-muted uppercase tracking-wide">
              Role
            </th>
            <th className="px-4 py-2.5 text-center text-xs font-semibold text-text-muted uppercase tracking-wide hidden sm:table-cell">
              Status
            </th>
          </tr>
        </thead>
        <tbody>
          {members.map((m) => (
            <tr
              key={m.id}
              className="border-b border-space-border last:border-b-0 hover:bg-space-border/20 transition-colors"
            >
              <td className="px-4 py-2.5">
                <div className="flex items-center gap-2">
                  <Avatar src={avatarSrc(m.avatar_url)} alt={m.username ?? ''} size="xs" />
                  <span className="text-text-primary">{m.username ?? '-'}</span>
                </div>
              </td>
              <td className="px-4 py-2.5 text-text-muted text-xs hidden sm:table-cell">
                {m.email ?? '-'}
              </td>
              <td className="px-4 py-2.5 text-center">
                <Badge variant={m.role === 'admin' ? 'blue' : 'default'}>{m.role ?? 'user'}</Badge>
              </td>
              <td className="px-4 py-2.5 text-center hidden sm:table-cell">
                {m.is_banned ? (
                  <Badge variant="red">Banned</Badge>
                ) : m.is_verified ? (
                  <span className="text-xs text-nebula-green">Verified</span>
                ) : (
                  <span className="text-xs text-text-muted">Unverified</span>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function MissingTab({ teamId }: { teamId: string }) {
  const { data: challenges = [], isLoading } = useAdminTeamMissing(teamId)

  if (isLoading) {
    return (
      <div className="flex flex-col gap-2">
        {[0, 1, 2].map((i) => (
          <Skeleton key={i} className="h-10 w-full" />
        ))}
      </div>
    )
  }

  if (challenges.length === 0) return <EmptyState message="All challenges solved!" />

  return (
    <div className="overflow-x-auto rounded-[var(--radius-lg)] border border-space-border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-space-border bg-space-dark/60">
            <th className="px-4 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
              Title
            </th>
            <th className="px-4 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide hidden sm:table-cell">
              Category
            </th>
            <th className="px-4 py-2.5 text-right text-xs font-semibold text-text-muted uppercase tracking-wide">
              Points
            </th>
          </tr>
        </thead>
        <tbody>
          {challenges.map((c) => (
            <tr
              key={c.id}
              className="border-b border-space-border last:border-b-0 hover:bg-space-border/20 transition-colors"
            >
              <td className="px-4 py-2.5 text-text-primary">{c.title ?? '-'}</td>
              <td className="px-4 py-2.5 text-text-secondary hidden sm:table-cell text-xs">
                {c.category ?? '-'}
              </td>
              <td className="px-4 py-2.5 text-right font-mono font-bold text-cosmic-blue">
                {c.points ?? 0}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------
export function AdminTeamDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const stateTeam = (location.state as { team?: AdminTeam } | null)?.team ?? null
  const [tab, setTab] = useState<Tab>('edit')

  const { data: team, isLoading } = useAdminTeam(id ?? '', stateTeam)

  if (!id) return null

  return (
    <div className="flex flex-col gap-5 max-w-3xl">
      <button
        onClick={() => navigate('/admin/teams')}
        className="flex items-center gap-1.5 text-sm text-text-muted hover:text-text-primary transition-colors w-fit"
      >
        <svg className="w-4 h-4" viewBox="0 0 16 16" fill="currentColor">
          <path
            fillRule="evenodd"
            d="M9.78 4.22a.75.75 0 0 1 0 1.06L7.06 8l2.72 2.72a.749.749 0 0 1-.326 1.275.749.749 0 0 1-.734-.215L5.97 8.53a.75.75 0 0 1 0-1.06l2.75-2.75a.75.75 0 0 1 1.06 0Z"
            clipRule="evenodd"
          />
        </svg>
        Back to teams
      </button>

      {isLoading ? (
        <div className="flex items-center gap-4">
          <Skeleton className="w-14 h-14 rounded-full" />
          <div className="flex flex-col gap-2">
            <Skeleton className="h-5 w-40" />
            <Skeleton className="h-4 w-24" />
          </div>
        </div>
      ) : !team ? (
        <p className="text-sm text-nova-red">Team not found.</p>
      ) : (
        <>
          <div className="flex items-center gap-4">
            <Avatar src={avatarSrc(team.avatar_url)} alt={team.name ?? ''} size="lg" />
            <div>
              <h1 className="text-xl font-bold text-text-primary">{team.name ?? '-'}</h1>
              <div className="flex items-center gap-2 mt-1">
                {team.is_banned && <Badge variant="red">Banned</Badge>}
                {team.is_hidden && <Badge variant="default">Hidden</Badge>}
                <span className="text-sm text-text-muted">{team.member_count ?? 0} members</span>
              </div>
            </div>
          </div>

          <div className="rounded-[var(--radius-lg)] border border-space-border bg-space-surface p-5 flex flex-col gap-4">
            <div className="flex gap-1 border-b border-space-border overflow-x-auto scrollbar-none">
              {TABS.map((t) => (
                <button
                  key={t.id}
                  onClick={() => setTab(t.id)}
                  className={`px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors focus-visible:outline-none
                    ${
                      tab === t.id
                        ? 'border-cosmic-blue text-cosmic-blue'
                        : 'border-transparent text-text-secondary hover:text-text-primary hover:border-space-border'
                    }`}
                >
                  {t.label}
                </button>
              ))}
            </div>
            <div className="pt-2">
              {tab === 'edit' && team && <EditTab team={team} teamId={id} />}
              {tab === 'members' && <MembersTab teamId={id} />}
              {tab === 'missing' && <MissingTab teamId={id} />}
              {tab === 'submissions' && <SubmissionsTab teamId={id} />}
            </div>
          </div>
        </>
      )}
    </div>
  )
}
