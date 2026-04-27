import { api } from '@/shared/api/client'
import type { operations } from '@/shared/api/schema.d'
import { EmptyState } from '@/shared/ui/empty-state'
import { Skeleton } from '@/shared/ui/skeleton'
import { useQuery } from '@tanstack/react-query'
import { lazy, Suspense, useMemo, useState, type ReactNode } from 'react'

const ScoreProgressionChart = lazy(() =>
  import('@/widgets/admin-charts/AdminCharts').then((m) => ({ default: m.ScoreProgressionChart })),
)
const SolvePercentagesChart = lazy(() =>
  import('@/widgets/admin-charts/AdminCharts').then((m) => ({ default: m.SolvePercentagesChart })),
)
const ScoreDistributionChart = lazy(() =>
  import('@/widgets/admin-charts/AdminCharts').then((m) => ({ default: m.ScoreDistributionChart })),
)
const SubmissionTimelineChart = lazy(() =>
  import('@/widgets/admin-charts/AdminCharts').then((m) => ({
    default: m.SubmissionTimelineChart,
  })),
)
const RegistrationGrowthChart = lazy(() =>
  import('@/widgets/admin-charts/AdminCharts').then((m) => ({
    default: m.RegistrationGrowthChart,
  })),
)

// ---------------------------------------------------------------------------
// Hooks
// ---------------------------------------------------------------------------
const LIVE = { params: { query: { live: true } } } as const

function useGeneralStats() {
  return useQuery({
    queryKey: ['admin', 'statistics', 'general'],
    queryFn: async () => {
      const { data, error } = await api.GET('/statistics/general', LIVE)
      if (error) throw error
      return data
    },
    staleTime: 0,
    refetchInterval: 30_000,
  })
}

function useScoreProgression() {
  return useQuery({
    queryKey: ['admin', 'statistics', 'scoreboard'],
    queryFn: async () => {
      const { data, error } = await api.GET('/statistics/scoreboard', {
        params: { query: { live: true, limit: 10 } },
      })
      if (error) throw error
      return (data ?? []) as Array<{ points?: number; team_name?: string; timestamp?: string }>
    },
    staleTime: 0,
  })
}

function useSolvePercentages() {
  return useQuery({
    queryKey: ['admin', 'statistics', 'solve-pct'],
    queryFn: async () => {
      const { data, error } = await api.GET('/statistics/challenges/solves/percentages', LIVE)
      if (error) throw error
      return (data ?? []) as Array<{
        title?: string
        percentage?: number
        solve_count?: number
      }>
    },
    staleTime: 0,
  })
}

function useScoreDistribution() {
  return useQuery({
    queryKey: ['admin', 'statistics', 'score-dist'],
    queryFn: async () => {
      const { data, error } = await api.GET('/statistics/scores/distribution', LIVE)
      if (error) throw error
      return (data ?? []) as Array<{ bucket?: string; count?: number }>
    },
    staleTime: 0,
  })
}

function useSubmissionTimeline() {
  return useQuery({
    queryKey: ['admin', 'statistics', 'submissions'],
    queryFn: async () => {
      const { data, error } = await api.GET('/statistics/submissions', LIVE)
      if (error) throw error
      return (data?.items ?? []) as Array<{
        date?: string
        correct?: number
        incorrect?: number
      }>
    },
    staleTime: 0,
  })
}

function useRegistrationGrowth() {
  return useQuery({
    queryKey: ['admin', 'statistics', 'users'],
    queryFn: async () => {
      const { data, error } = await api.GET('/statistics/users')
      if (error) throw error
      return (data ?? []) as Array<{ date?: string; count?: number }>
    },
    staleTime: 0,
  })
}

// ---------------------------------------------------------------------------
// Solve matrix hook + component
// ---------------------------------------------------------------------------
type MatrixRow = NonNullable<
  operations['GetAdminStatisticsSolveMatrix']['responses']['200']['content']['application/json']
>[number]

function useSolveMatrix() {
  return useQuery({
    queryKey: ['admin', 'statistics', 'solve-matrix'],
    queryFn: async () => {
      const { data, error } = await api.GET('/admin/statistics/solve-matrix', {
        params: { query: { live: true } },
      })
      if (error) throw error
      return (data ?? []) as MatrixRow[]
    },
    staleTime: 0,
    refetchInterval: 30_000,
  })
}

function SolveMatrix() {
  const { data: rows = [], isLoading } = useSolveMatrix()
  const [search, setSearch] = useState('')

  const { teams, challenges, solvedSet } = useMemo(() => {
    const teamsMap = new Map<string, string>()
    const challengesMap = new Map<string, { title: string; category: string }>()
    const solvedSet = new Set<string>()

    for (const row of rows) {
      if (row.team_id && row.team_name) teamsMap.set(row.team_id, row.team_name)
      if (row.challenge_id && row.challenge_title) {
        challengesMap.set(row.challenge_id, {
          title: row.challenge_title,
          category: row.challenge_category ?? '',
        })
      }
      if (row.solved && row.team_id && row.challenge_id) {
        solvedSet.add(`${row.team_id}:${row.challenge_id}`)
      }
    }

    const q = search.toLowerCase()
    const teams: [string, string][] = [...teamsMap.entries()]
      .filter(([, name]) => !q || name.toLowerCase().includes(q))
      .sort((a, b) => a[1].localeCompare(b[1]))

    const challenges: [string, { title: string; category: string }][] = [
      ...challengesMap.entries(),
    ].sort((a, b) => (a[1].category + a[1].title).localeCompare(b[1].category + b[1].title))

    return { teams, challenges, solvedSet }
  }, [rows, search])

  if (isLoading) {
    return <Skeleton className="w-full h-48" />
  }

  if (rows.length === 0) {
    return <EmptyState message="No solve data yet." />
  }

  return (
    <div className="flex flex-col gap-3">
      <input
        type="search"
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        placeholder="Filter teams…"
        className="w-full max-w-xs rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 py-1.5 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
      />

      <div className="overflow-auto max-h-[600px] rounded-[var(--radius-lg)] border border-space-border">
        <table
          className="border-collapse text-xs"
          style={{ minWidth: `${180 + challenges.length * 36}px` }}
        >
          <thead className="sticky top-0 z-10 bg-space-card">
            <tr>
              {/* Team name header */}
              <th className="sticky left-0 z-20 bg-space-card border-b border-r border-space-border px-3 py-2 text-left font-semibold text-text-muted whitespace-nowrap min-w-[160px]">
                Team
                <span className="ml-1 font-normal text-text-muted/60">({teams.length})</span>
              </th>
              {challenges.map(([cid, ch]) => (
                <th
                  key={cid}
                  title={`${ch.category ? ch.category + ' - ' : ''}${ch.title}`}
                  className="border-b border-r border-space-border px-1 py-2 font-medium text-text-muted"
                  style={{
                    writingMode: 'vertical-rl',
                    transform: 'rotate(180deg)',
                    height: 80,
                    maxWidth: 32,
                  }}
                >
                  <span className="truncate block max-w-[72px]">{ch.title}</span>
                </th>
              ))}
              <th className="sticky right-0 z-20 bg-space-card border-b border-l border-space-border px-3 py-2 text-right font-semibold text-text-muted whitespace-nowrap">
                Score
              </th>
            </tr>
          </thead>
          <tbody>
            {teams.map(([teamId, teamName]) => {
              const solveCount = challenges.filter(([cid]) =>
                solvedSet.has(`${teamId}:${cid}`),
              ).length
              return (
                <tr
                  key={teamId}
                  className="border-b border-space-border last:border-b-0 hover:bg-space-border/20 transition-colors"
                >
                  <td className="sticky left-0 z-10 bg-space-card border-r border-space-border px-3 py-1.5 font-medium text-text-primary whitespace-nowrap">
                    {teamName}
                  </td>
                  {challenges.map(([cid]) => {
                    const solved = solvedSet.has(`${teamId}:${cid}`)
                    return (
                      <td
                        key={cid}
                        className="border-r border-space-border text-center py-1.5"
                        title={solved ? 'Solved' : 'Not solved'}
                      >
                        {solved ? (
                          <span className="inline-block w-5 h-5 rounded-sm bg-nebula-green/20 text-nebula-green leading-5 text-center">
                            ✓
                          </span>
                        ) : (
                          <span className="inline-block w-5 h-5 rounded-sm bg-space-border/30" />
                        )}
                      </td>
                    )
                  })}
                  <td className="sticky right-0 z-10 bg-space-card border-l border-space-border px-3 py-1.5 text-right font-mono font-semibold text-text-primary">
                    {solveCount}/{challenges.length}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Counter card
// ---------------------------------------------------------------------------
interface StatCard {
  label: string
  icon: string
  value: number | undefined
  color: string
}

function CounterCard({ label, icon, value, color }: StatCard) {
  return (
    <div
      className="flex items-center gap-4 rounded-[var(--radius-lg)] border border-space-border bg-space-card px-5 py-4"
      style={{ borderLeftWidth: 3, borderLeftColor: color }}
    >
      <span className="text-3xl leading-none shrink-0">{icon}</span>
      <div className="min-w-0">
        <p className="text-xs font-semibold text-text-muted uppercase tracking-wide truncate">
          {label}
        </p>
        {value === undefined ? (
          <Skeleton className="h-7 w-16 mt-1" />
        ) : (
          <p
            className="text-2xl font-black tabular-nums"
            style={{ fontFamily: 'var(--font-display)', color }}
          >
            {value.toLocaleString()}
          </p>
        )}
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Chart card wrapper
// ---------------------------------------------------------------------------
function ChartCard({
  title,
  children,
  loading,
}: {
  title: string
  children: ReactNode
  loading: boolean
}) {
  return (
    <div className="rounded-[var(--radius-lg)] border border-space-border bg-space-card p-4">
      <h2 className="text-xs font-semibold text-text-muted uppercase tracking-wide mb-3">
        {title}
      </h2>
      {loading ? (
        <Skeleton className="w-full h-64 rounded-[var(--radius-md)]" />
      ) : (
        <Suspense fallback={<Skeleton className="w-full h-64 rounded-[var(--radius-md)]" />}>
          {children}
        </Suspense>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------
export function AdminStatisticsPage() {
  const { data: general, isLoading: loadingGeneral } = useGeneralStats()
  const { data: scoreData, isLoading: loadingScore } = useScoreProgression()
  const { data: pctData, isLoading: loadingPct } = useSolvePercentages()
  const { data: distData, isLoading: loadingDist } = useScoreDistribution()
  const { data: subData, isLoading: loadingSub } = useSubmissionTimeline()
  const { data: regData, isLoading: loadingReg } = useRegistrationGrowth()

  const cards: StatCard[] = [
    {
      label: 'Users',
      icon: '👤',
      value: loadingGeneral ? undefined : (general?.user_count ?? 0),
      color: 'var(--color-cosmic-blue)',
    },
    {
      label: 'Teams',
      icon: '🛡️',
      value: loadingGeneral ? undefined : (general?.team_count ?? 0),
      color: 'var(--color-stellar-green)',
    },
    {
      label: 'Challenges',
      icon: '🎯',
      value: loadingGeneral ? undefined : (general?.challenge_count ?? 0),
      color: 'var(--color-solar-yellow)',
    },
    {
      label: 'Solves',
      icon: '✅',
      value: loadingGeneral ? undefined : (general?.solve_count ?? 0),
      color: 'var(--color-nova-red)',
    },
  ]

  return (
    <div className="flex flex-col gap-6">
      <h1
        className="text-2xl font-black text-text-primary"
        style={{ fontFamily: 'var(--font-display)' }}
      >
        Statistics
      </h1>

      {/* Counter cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {cards.map((card) => (
          <CounterCard key={card.label} {...card} />
        ))}
      </div>

      {/* Charts - 2 col on lg+ */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <ChartCard title="Score Progression - Top 10" loading={loadingScore}>
          <ScoreProgressionChart data={scoreData ?? []} />
        </ChartCard>

        <ChartCard title="Solve Percentages" loading={loadingPct}>
          <SolvePercentagesChart data={pctData ?? []} />
        </ChartCard>

        <ChartCard title="Score Distribution" loading={loadingDist}>
          <ScoreDistributionChart data={distData ?? []} />
        </ChartCard>

        <ChartCard title="Submission Timeline" loading={loadingSub}>
          <SubmissionTimelineChart data={subData ?? []} />
        </ChartCard>

        <div className="lg:col-span-2">
          <ChartCard title="Registration Growth" loading={loadingReg}>
            <RegistrationGrowthChart data={regData ?? []} />
          </ChartCard>
        </div>
      </div>

      {/* Solve matrix */}
      <div className="rounded-[var(--radius-lg)] border border-space-border bg-space-card p-4">
        <h2 className="text-xs font-semibold text-text-muted uppercase tracking-wide mb-3">
          Solve Matrix - Teams × Challenges
        </h2>
        <SolveMatrix />
      </div>
    </div>
  )
}
