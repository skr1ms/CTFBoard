import { api } from '@/shared/api/client'
import { useAuthStore } from '@/shared/stores/authStore'
import { Skeleton } from '@/shared/ui/skeleton'
import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'

type ActivityTab =
  | 'my-solves'
  | 'my-fails'
  | 'my-awards'
  | 'my-submissions'
  | 'team-solves'
  | 'team-fails'
  | 'team-awards'

function renderSolvesTable(
  rows: Array<{
    id?: string
    challenge_title?: string
    challenge_category?: string
    challenge_points?: number
    solved_at?: string
  }>,
) {
  if (rows.length === 0) return <p className="text-sm text-text-muted py-2">Nothing yet.</p>
  return (
    <div className="overflow-x-auto rounded-[var(--radius-lg)] border border-space-border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-space-border bg-space-dark/60">
            <th className="px-4 py-2 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
              Challenge
            </th>
            <th className="px-4 py-2 text-left text-xs font-semibold text-text-muted uppercase tracking-wide hidden sm:table-cell">
              Category
            </th>
            <th className="px-4 py-2 text-right text-xs font-semibold text-text-muted uppercase tracking-wide">
              Pts
            </th>
            <th className="px-4 py-2 text-right text-xs font-semibold text-text-muted uppercase tracking-wide hidden md:table-cell">
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
              <td className="px-4 py-2.5 text-text-primary text-sm">{s.challenge_title ?? '-'}</td>
              <td className="px-4 py-2.5 text-text-muted text-xs hidden sm:table-cell">
                {s.challenge_category ?? '-'}
              </td>
              <td className="px-4 py-2.5 text-right font-mono font-bold text-cosmic-blue text-sm">
                {s.challenge_points ?? 0}
              </td>
              <td className="px-4 py-2.5 text-right text-text-muted text-xs hidden md:table-cell">
                {s.solved_at ? new Date(s.solved_at).toLocaleString() : '-'}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function renderSubmissionsTable(
  rows: Array<{
    id?: string
    challenge_title?: string
    submitted_flag?: string
    is_correct?: boolean
  }>,
) {
  if (rows.length === 0) return <p className="text-sm text-text-muted py-2">Nothing yet.</p>
  return (
    <div className="overflow-x-auto rounded-[var(--radius-lg)] border border-space-border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-space-border bg-space-dark/60">
            <th className="px-4 py-2 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
              Challenge
            </th>
            <th className="px-4 py-2 text-left text-xs font-semibold text-text-muted uppercase tracking-wide hidden sm:table-cell">
              Flag
            </th>
            <th className="px-4 py-2 text-center text-xs font-semibold text-text-muted uppercase tracking-wide">
              Result
            </th>
          </tr>
        </thead>
        <tbody>
          {rows.map((s) => (
            <tr
              key={s.id}
              className="border-b border-space-border last:border-b-0 hover:bg-space-border/20 transition-colors"
            >
              <td className="px-4 py-2.5 text-text-primary">{s.challenge_title ?? '-'}</td>
              <td className="px-4 py-2.5 font-mono text-xs text-text-muted hidden sm:table-cell truncate max-w-[160px]">
                {s.submitted_flag ?? '-'}
              </td>
              <td className="px-4 py-2.5 text-center">
                <span
                  className={`px-1.5 py-0.5 rounded text-xs font-medium ${s.is_correct ? 'bg-nebula-green/10 text-nebula-green' : 'bg-nova-red/10 text-nova-red'}`}
                >
                  {s.is_correct ? 'Correct' : 'Wrong'}
                </span>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function renderAwardsTable(
  rows: Array<{
    id?: string
    description?: string
    value?: number
    created_at?: string
  }>,
) {
  if (rows.length === 0) return <p className="text-sm text-text-muted py-2">Nothing yet.</p>
  return (
    <div className="overflow-x-auto rounded-[var(--radius-lg)] border border-space-border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-space-border bg-space-dark/60">
            <th className="px-4 py-2 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
              Description
            </th>
            <th className="px-4 py-2 text-right text-xs font-semibold text-text-muted uppercase tracking-wide">
              Value
            </th>
            <th className="px-4 py-2 text-right text-xs font-semibold text-text-muted uppercase tracking-wide hidden md:table-cell">
              Date
            </th>
          </tr>
        </thead>
        <tbody>
          {rows.map((a) => (
            <tr
              key={a.id}
              className="border-b border-space-border last:border-b-0 hover:bg-space-border/20 transition-colors"
            >
              <td className="px-4 py-2.5 text-text-primary">{a.description ?? '-'}</td>
              <td className="px-4 py-2.5 text-right font-mono font-bold text-nova-yellow">
                {a.value ?? 0}
              </td>
              <td className="px-4 py-2.5 text-right text-text-muted text-xs hidden md:table-cell">
                {a.created_at ? new Date(a.created_at).toLocaleString() : '-'}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export function ActivityPage() {
  const user = useAuthStore((s) => s.user)
  const hasTeam = !!user?.team_id
  const [tab, setTab] = useState<ActivityTab>('my-solves')

  const MY_TABS: { id: ActivityTab; label: string }[] = [
    { id: 'my-solves', label: 'My Solves' },
    { id: 'my-fails', label: 'My Fails' },
    { id: 'my-awards', label: 'My Awards' },
    { id: 'my-submissions', label: 'My Submissions' },
  ]

  const TEAM_TABS: { id: ActivityTab; label: string }[] = hasTeam
    ? [
        { id: 'team-solves', label: 'Team Solves' },
        { id: 'team-fails', label: 'Team Fails' },
        { id: 'team-awards', label: 'Team Awards' },
      ]
    : []

  const ALL_TABS = [...MY_TABS, ...TEAM_TABS]

  const { data: mySolves = [], isLoading: loadMySolves } = useQuery({
    queryKey: ['me', 'solves'],
    queryFn: async () => {
      const { data, error } = await api.GET('/users/me/solves')
      if (error) throw error
      return data ?? []
    },
    enabled: tab === 'my-solves',
    staleTime: 30_000,
  })

  const { data: myFailsData, isLoading: loadMyFails } = useQuery({
    queryKey: ['me', 'fails'],
    queryFn: async () => {
      const { data, error } = await api.GET('/users/me/fails')
      if (error) throw error
      return data
    },
    enabled: tab === 'my-fails',
    staleTime: 30_000,
  })
  const myFails = myFailsData?.data ?? []

  const { data: myAwards = [], isLoading: loadMyAwards } = useQuery({
    queryKey: ['me', 'awards'],
    queryFn: async () => {
      const { data, error } = await api.GET('/users/me/awards')
      if (error) throw error
      return data ?? []
    },
    enabled: tab === 'my-awards',
    staleTime: 30_000,
  })

  const { data: mySubmissionsData, isLoading: loadMySubmissions } = useQuery({
    queryKey: ['me', 'submissions'],
    queryFn: async () => {
      const { data, error } = await api.GET('/users/me/submissions')
      if (error) throw error
      return data
    },
    enabled: tab === 'my-submissions',
    staleTime: 30_000,
  })
  const mySubmissions = mySubmissionsData?.data ?? []

  const { data: teamSolves = [], isLoading: loadTeamSolves } = useQuery({
    queryKey: ['teams', 'me', 'solves'],
    queryFn: async () => {
      const { data, error } = await api.GET('/teams/me/solves')
      if (error) throw error
      return data ?? []
    },
    enabled: tab === 'team-solves' && hasTeam,
    staleTime: 30_000,
  })

  const { data: teamFailsData, isLoading: loadTeamFails } = useQuery({
    queryKey: ['teams', 'me', 'fails'],
    queryFn: async () => {
      const { data, error } = await api.GET('/teams/me/fails')
      if (error) throw error
      return data
    },
    enabled: tab === 'team-fails' && hasTeam,
    staleTime: 30_000,
  })
  const teamFails = teamFailsData?.data ?? []

  const { data: teamAwards = [], isLoading: loadTeamAwards } = useQuery({
    queryKey: ['teams', 'me', 'awards'],
    queryFn: async () => {
      const { data, error } = await api.GET('/teams/me/awards')
      if (error) throw error
      return data ?? []
    },
    enabled: tab === 'team-awards' && hasTeam,
    staleTime: 30_000,
  })

  const isLoading =
    (tab === 'my-solves' && loadMySolves) ||
    (tab === 'my-fails' && loadMyFails) ||
    (tab === 'my-awards' && loadMyAwards) ||
    (tab === 'my-submissions' && loadMySubmissions) ||
    (tab === 'team-solves' && loadTeamSolves) ||
    (tab === 'team-fails' && loadTeamFails) ||
    (tab === 'team-awards' && loadTeamAwards)

  return (
    <div className="flex flex-col gap-6 max-w-4xl mx-auto px-4 py-6 sm:px-6">
      <h1
        className="text-2xl font-black text-text-primary"
        style={{ fontFamily: 'var(--font-display)' }}
      >
        Activity
      </h1>

      <div className="flex gap-1 flex-wrap border-b border-space-border">
        {ALL_TABS.map((t) => (
          <button
            key={t.id}
            onClick={() => setTab(t.id)}
            className={`px-3 py-2 text-sm whitespace-nowrap border-b-2 -mb-px transition-colors focus-visible:outline-none
              ${
                tab === t.id
                  ? 'border-cosmic-blue text-cosmic-blue font-medium'
                  : 'border-transparent text-text-secondary hover:text-text-primary hover:border-space-border'
              }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      <div className="min-h-[120px]">
        {isLoading ? (
          <div className="flex flex-col gap-2">
            {[1, 2, 3].map((i) => (
              <Skeleton key={i} className="h-10 w-full" />
            ))}
          </div>
        ) : (
          <>
            {tab === 'my-solves' && renderSolvesTable(mySolves)}
            {tab === 'my-fails' && renderSubmissionsTable(myFails)}
            {tab === 'my-awards' && renderAwardsTable(myAwards)}
            {tab === 'my-submissions' && renderSubmissionsTable(mySubmissions)}
            {tab === 'team-solves' && renderSolvesTable(teamSolves)}
            {tab === 'team-fails' && renderSubmissionsTable(teamFails)}
            {tab === 'team-awards' && renderAwardsTable(teamAwards)}
          </>
        )}
      </div>
    </div>
  )
}
