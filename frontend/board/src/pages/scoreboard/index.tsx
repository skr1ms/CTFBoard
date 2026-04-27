import type { ScoreboardEntry } from '@/features/scoreboard/useScoreboard'
import { useBrackets, useScoreboard, useScoreboardGraph } from '@/features/scoreboard/useScoreboard'
import { avatarSrc } from '@/shared/lib/utils'
import { Avatar } from '@/shared/ui/avatar'
import { EmptyState } from '@/shared/ui/empty-state'
import { Skeleton } from '@/shared/ui/skeleton'
import { VisibilityGate } from '@/shared/ui/visibility-gate'
import { lazy, Suspense, useState } from 'react'
import { useNavigate } from 'react-router'

const ScoreboardGraph = lazy(() =>
  import('@/widgets/scoreboard-graph/ScoreboardGraph').then((m) => ({
    default: m.ScoreboardGraph,
  })),
)

// ---------------------------------------------------------------------------
// Medal icons for top-3
// ---------------------------------------------------------------------------
const MEDALS = ['🥇', '🥈', '🥉']

function MedalOrRank({ rank }: { rank: number }) {
  if (rank <= 3) {
    return (
      <span className="text-lg leading-none w-7 text-center shrink-0" aria-label={`Rank ${rank}`}>
        {MEDALS[rank - 1]}
      </span>
    )
  }
  return (
    <span className="w-7 text-center tabular-nums text-sm text-text-muted font-mono shrink-0">
      {rank}
    </span>
  )
}

// ---------------------------------------------------------------------------
// Table row
// ---------------------------------------------------------------------------
interface RowProps {
  entry: ScoreboardEntry
  rank: number
  onClick: () => void
}

function ScoreboardRow({ entry, rank, onClick }: RowProps) {
  const isTop3 = rank <= 3

  return (
    <tr
      data-testid={`scoreboard-row-${entry.team_id}`}
      onClick={onClick}
      className={`
        group cursor-pointer transition-colors
        ${isTop3 ? 'bg-cosmic-blue/5 hover:bg-cosmic-blue/10' : 'hover:bg-space-border/30'}
        border-b border-space-border last:border-b-0
      `}
    >
      {/* Rank */}
      <td className="px-4 py-3 w-12">
        <MedalOrRank rank={rank} />
      </td>

      {/* Team */}
      <td className="px-4 py-3">
        <div className="flex items-center gap-3 min-w-0">
          <Avatar
            src={avatarSrc(entry.team_avatar_thumbnail_url)}
            {...(entry.team_name ? { alt: entry.team_name } : {})}
            size="sm"
          />
          <span className="text-sm font-medium text-text-primary group-hover:text-cosmic-blue transition-colors truncate">
            {entry.team_name ?? 'Unknown team'}
          </span>
        </div>
      </td>

      {/* Score */}
      <td className="px-4 py-3 text-right">
        <span
          className="text-sm font-bold tabular-nums"
          style={{ fontFamily: 'var(--font-display)', color: 'var(--color-cosmic-blue)' }}
        >
          {entry.points ?? 0}
        </span>
      </td>

      {/* Last solved */}
      <td className="px-4 py-3 text-right text-xs text-text-muted hidden sm:table-cell whitespace-nowrap">
        {entry.last_solved
          ? new Date(entry.last_solved).toLocaleString(undefined, {
              month: 'short',
              day: 'numeric',
              hour: '2-digit',
              minute: '2-digit',
            })
          : '-'}
      </td>
    </tr>
  )
}

// ---------------------------------------------------------------------------
// Skeleton rows
// ---------------------------------------------------------------------------
function TableSkeleton() {
  return (
    <>
      {Array.from({ length: 8 }).map((_, i) => (
        <tr key={i} className="border-b border-space-border last:border-b-0">
          <td className="px-4 py-3 w-12">
            <Skeleton className="h-5 w-7" />
          </td>
          <td className="px-4 py-3">
            <div className="flex items-center gap-3">
              <Skeleton circle className="w-8 h-8" />
              <Skeleton className="h-4 w-32" />
            </div>
          </td>
          <td className="px-4 py-3 text-right">
            <Skeleton className="h-4 w-12 ml-auto" />
          </td>
          <td className="px-4 py-3 hidden sm:table-cell text-right">
            <Skeleton className="h-4 w-24 ml-auto" />
          </td>
        </tr>
      ))}
    </>
  )
}

// ---------------------------------------------------------------------------
// Bracket tab bar
// ---------------------------------------------------------------------------
interface BracketTabsProps {
  brackets: Array<{ id?: string; name?: string }>
  selected: string | null
  onSelect: (id: string | null) => void
}

function BracketTabs({ brackets, selected, onSelect }: BracketTabsProps) {
  return (
    <div className="flex gap-1 overflow-x-auto scrollbar-none border-b border-space-border pb-0">
      <button
        onClick={() => onSelect(null)}
        className={`px-4 py-2 text-sm font-medium whitespace-nowrap transition-colors border-b-2 -mb-px
          focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cosmic-blue rounded-t
          ${
            selected === null
              ? 'border-cosmic-blue text-cosmic-blue'
              : 'border-transparent text-text-secondary hover:text-text-primary hover:border-space-border'
          }`}
      >
        All
      </button>
      {brackets.map((b) => (
        <button
          key={b.id}
          onClick={() => onSelect(b.id ?? null)}
          className={`px-4 py-2 text-sm font-medium whitespace-nowrap transition-colors border-b-2 -mb-px
            focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cosmic-blue rounded-t
            ${
              selected === b.id
                ? 'border-cosmic-blue text-cosmic-blue'
                : 'border-transparent text-text-secondary hover:text-text-primary hover:border-space-border'
            }`}
        >
          {b.name}
        </button>
      ))}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Graph skeleton
// ---------------------------------------------------------------------------
function GraphSkeleton() {
  return <Skeleton className="w-full h-72 sm:h-96 rounded-[var(--radius-lg)]" />
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------
export function ScoreboardPage() {
  const navigate = useNavigate()
  const [selectedBracket, setSelectedBracket] = useState<string | null>(null)

  const { data: entries, isLoading } = useScoreboard(selectedBracket ?? undefined)
  const { data: brackets } = useBrackets()
  const { data: graphData } = useScoreboardGraph(10)

  const hasBrackets = (brackets?.length ?? 0) > 0
  const graphTeams = graphData?.teams ?? []

  return (
    <VisibilityGate configKey="score_visibility">
      <div className="flex flex-col gap-6 max-w-4xl mx-auto px-4 py-6 sm:px-6 lg:px-8">
        {/* Header */}
        <div className="flex items-center justify-between">
          <h1
            className="text-2xl font-black text-text-primary"
            style={{ fontFamily: 'var(--font-display)' }}
          >
            Scoreboard
          </h1>
          {!isLoading && entries && entries.length > 0 && (
            <span className="text-sm text-text-muted">
              {entries.length} team{entries.length !== 1 ? 's' : ''}
            </span>
          )}
        </div>

        {/* Score progression graph (lazy-loaded) */}
        {graphTeams.length > 0 && (
          <div
            data-testid="scoreboard-graph"
            className="rounded-[var(--radius-lg)] border border-space-border bg-space-card p-4"
          >
            <h2 className="text-xs font-semibold text-text-muted uppercase tracking-wide mb-3">
              Score Progression - Top {graphTeams.length}
            </h2>
            <Suspense fallback={<GraphSkeleton />}>
              <ScoreboardGraph teams={graphTeams} />
            </Suspense>
          </div>
        )}

        {/* Bracket filter tabs */}
        {hasBrackets && (
          <BracketTabs
            brackets={brackets ?? []}
            selected={selectedBracket}
            onSelect={setSelectedBracket}
          />
        )}

        {/* Table */}
        <div className="overflow-x-auto rounded-[var(--radius-lg)] border border-space-border">
          <table className="w-full min-w-[480px]">
            <thead>
              <tr className="border-b border-space-border bg-space-dark/60">
                <th className="px-4 py-3 text-left text-xs font-semibold text-text-muted uppercase tracking-wide w-12">
                  #
                </th>
                <th className="px-4 py-3 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
                  Team
                </th>
                <th className="px-4 py-3 text-right text-xs font-semibold text-text-muted uppercase tracking-wide">
                  Score
                </th>
                <th className="px-4 py-3 text-right text-xs font-semibold text-text-muted uppercase tracking-wide hidden sm:table-cell">
                  Last solve
                </th>
              </tr>
            </thead>
            <tbody>
              {isLoading ? (
                <TableSkeleton />
              ) : !entries || entries.length === 0 ? (
                <tr>
                  <td colSpan={4}>
                    <EmptyState message="No teams on the scoreboard yet." />
                  </td>
                </tr>
              ) : (
                entries.map((entry, i) => (
                  <ScoreboardRow
                    key={entry.team_id ?? i}
                    entry={entry}
                    rank={i + 1}
                    onClick={() => entry.team_id && navigate(`/teams/${entry.team_id}`)}
                  />
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </VisibilityGate>
  )
}
