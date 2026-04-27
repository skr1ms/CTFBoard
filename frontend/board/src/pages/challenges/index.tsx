import type { ChallengeResponse } from '@/features/challenges/useChallenges'
import { useChallenges, useTags } from '@/features/challenges/useChallenges'
import { useCompetitionStatus } from '@/features/competition/useCompetitionStatus'
import { useAuthStore } from '@/shared/stores/authStore'
import { Badge } from '@/shared/ui/badge'
import { EmptyState } from '@/shared/ui/empty-state'
import { Skeleton } from '@/shared/ui/skeleton'
import { VisibilityGate } from '@/shared/ui/visibility-gate'
import { ChallengeModal } from '@/widgets/challenge-modal/ChallengeModal'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router'

// ---------------------------------------------------------------------------
// Category colour mapping
// ---------------------------------------------------------------------------
function categoryColor(cat?: string): string {
  if (!cat) return 'var(--color-text-muted)'
  const key = cat.toLowerCase()
  const map: Record<string, string> = {
    web: 'var(--color-cosmic-blue)',
    crypto: 'var(--color-nebula-purple)',
    pwn: 'var(--color-nova-red)',
    rev: 'var(--color-solar-yellow)',
    reverse: 'var(--color-solar-yellow)',
    forensics: 'var(--color-aurora-cyan)',
    misc: 'var(--color-stellar-green)',
    osint: 'var(--color-aurora-cyan)',
    network: 'var(--color-cosmic-blue)',
    blockchain: 'var(--color-nebula-purple)',
    mobile: 'var(--color-stellar-green)',
  }
  return map[key] ?? 'var(--color-text-muted)'
}

// ---------------------------------------------------------------------------
// Challenge card
// ---------------------------------------------------------------------------
interface ChallengeCardProps {
  challenge: ChallengeResponse
  onClick: () => void
}

function ChallengeCard({ challenge, onClick }: ChallengeCardProps) {
  const isLocked = challenge.state === 'locked'
  const isSolved = challenge.solved

  return (
    <button
      onClick={onClick}
      data-testid="challenge-card"
      className={`
        group relative flex flex-col gap-2 p-4 rounded-[var(--radius-lg)] border text-left w-full
        transition-all duration-200
        ${
          isSolved
            ? 'border-stellar-green/30 bg-stellar-green/5 hover:border-stellar-green/50 hover:bg-stellar-green/10'
            : isLocked
              ? 'border-space-border/50 bg-space-dark/30 opacity-60 hover:opacity-80'
              : 'border-space-border bg-space-card hover:border-cosmic-blue/50 hover:bg-cosmic-blue/5 hover:-translate-y-0.5 hover:shadow-lg hover:shadow-cosmic-blue/5'
        }
      `}
    >
      {/* Solved ribbon */}
      {isSolved && (
        <div className="absolute top-0 right-0 overflow-hidden w-10 h-10 rounded-tr-[var(--radius-lg)]">
          <div className="absolute top-1 right-1 text-stellar-green">
            <svg className="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
              <path
                fillRule="evenodd"
                d="M10 18a8 8 0 1 0 0-16 8 8 0 0 0 0 16Zm3.857-9.809a.75.75 0 0 0-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 1 0-1.06 1.061l2.5 2.5a.75.75 0 0 0 1.137-.089l4-5.5Z"
                clipRule="evenodd"
              />
            </svg>
          </div>
        </div>
      )}

      {/* Lock icon */}
      {isLocked && !isSolved && (
        <div className="absolute top-3 right-3 text-text-muted" data-testid="challenge-locked-icon">
          <svg className="w-4 h-4" viewBox="0 0 16 16" fill="currentColor">
            <path d="M8 1a2 2 0 0 1 2 2v4H6V3a2 2 0 0 1 2-2zm3 6V3a3 3 0 0 0-6 0v4a2 2 0 0 0-2 2v5a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2z" />
          </svg>
        </div>
      )}

      {/* Title */}
      <p className="text-sm font-semibold text-text-primary leading-snug pr-5 truncate">
        {challenge.title ?? 'Untitled'}
      </p>

      {/* Tags */}
      {challenge.tags && challenge.tags.length > 0 && (
        <div className="flex flex-wrap gap-1">
          {challenge.tags.slice(0, 3).map((tag) => (
            <Badge key={tag.id} variant="default" className="text-[10px] px-1.5 py-0">
              {tag.name}
            </Badge>
          ))}
        </div>
      )}

      {/* Footer */}
      <div className="flex items-center justify-between mt-auto pt-1">
        <span className="text-xs text-text-muted">
          {challenge.solve_count ?? 0} solve{challenge.solve_count !== 1 ? 's' : ''}
        </span>
        <span
          className="text-base font-black tabular-nums"
          style={{ fontFamily: 'var(--font-display)', color: categoryColor(challenge.category) }}
        >
          {challenge.points ?? 0}
        </span>
      </div>
    </button>
  )
}

// ---------------------------------------------------------------------------
// Skeleton grid
// ---------------------------------------------------------------------------
function ChallengeGridSkeleton() {
  return (
    <div className="flex flex-col gap-8">
      {[0, 1].map((i) => (
        <div key={i} className="flex flex-col gap-3">
          <Skeleton className="h-5 w-32" />
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-3">
            {Array.from({ length: 5 }).map((_, j) => (
              <Skeleton key={j} className="h-28 w-full" />
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Tag filter bar
// ---------------------------------------------------------------------------
interface TagFilterProps {
  tags: Array<{ id?: string; name?: string }>
  selected: string | null
  onSelect: (id: string | null) => void
}

function TagFilterBar({ tags, selected, onSelect }: TagFilterProps) {
  if (tags.length === 0) return null

  return (
    <div className="flex gap-1 overflow-x-auto scrollbar-none pb-1">
      <button
        onClick={() => onSelect(null)}
        className={`px-3 py-1.5 rounded-full text-xs font-medium whitespace-nowrap transition-colors
          ${
            selected === null
              ? 'bg-cosmic-blue text-white'
              : 'bg-space-border/50 text-text-secondary hover:bg-space-border hover:text-text-primary'
          }`}
      >
        All
      </button>
      {tags.map((tag) => (
        <button
          key={tag.id}
          onClick={() => onSelect(tag.id ?? null)}
          className={`px-3 py-1.5 rounded-full text-xs font-medium whitespace-nowrap transition-colors
            ${
              selected === tag.id
                ? 'bg-cosmic-blue text-white'
                : 'bg-space-border/50 text-text-secondary hover:bg-space-border hover:text-text-primary'
            }`}
        >
          {tag.name}
        </button>
      ))}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Category group
// ---------------------------------------------------------------------------
interface CategoryGroupProps {
  category: string
  challenges: ChallengeResponse[]
  onOpen: (id: string) => void
}

function CategoryGroup({ category, challenges, onOpen }: CategoryGroupProps) {
  const solved = challenges.filter((c) => c.solved).length
  const total = challenges.length

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-3">
        <h2
          className="text-sm font-bold uppercase tracking-widest"
          style={{ color: categoryColor(category) }}
        >
          {category}
        </h2>
        <div className="flex-1 h-px bg-space-border" />
        <span className="text-xs text-text-muted font-mono">
          {solved}/{total}
        </span>
      </div>
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-3">
        {[...challenges]
          .sort((a, b) => (a.position ?? 0) - (b.position ?? 0))
          .map((c) => (
            <ChallengeCard key={c.id} challenge={c} onClick={() => c.id && onOpen(c.id)} />
          ))}
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Countdown banner
// ---------------------------------------------------------------------------
function useCountdown(targetISO: string | null | undefined) {
  const [remaining, setRemaining] = useState(0)

  useEffect(() => {
    if (!targetISO) return
    const target = new Date(targetISO).getTime()
    const tick = () => setRemaining(Math.max(0, Math.floor((target - Date.now()) / 1000)))
    tick()
    const id = setInterval(tick, 1000)
    return () => clearInterval(id)
  }, [targetISO])

  const d = Math.floor(remaining / 86400)
  const h = Math.floor((remaining % 86400) / 3600)
  const m = Math.floor((remaining % 3600) / 60)
  const s = remaining % 60
  return { remaining, d, h, m, s }
}

function pad(n: number) {
  return String(n).padStart(2, '0')
}

function CompetitionCountdownBanner({ startTime }: { startTime: string }) {
  const { remaining, d, h, m, s } = useCountdown(startTime)

  return (
    <div className="rounded-[var(--radius-lg)] border border-cosmic-blue/30 bg-cosmic-blue/5 p-6 flex flex-col items-center gap-4 text-center">
      <p className="text-lg font-bold text-text-primary">Competition hasn't started yet</p>
      {remaining > 0 ? (
        <div className="flex items-end gap-3">
          {d > 0 && (
            <div className="flex flex-col items-center">
              <span
                className="text-3xl font-black tabular-nums text-cosmic-blue"
                style={{ fontFamily: 'var(--font-display)' }}
              >
                {d}
              </span>
              <span className="text-xs text-text-muted uppercase tracking-widest">days</span>
            </div>
          )}
          <div className="flex flex-col items-center">
            <span
              className="text-3xl font-black tabular-nums text-cosmic-blue"
              style={{ fontFamily: 'var(--font-display)' }}
            >
              {pad(h)}
            </span>
            <span className="text-xs text-text-muted uppercase tracking-widest">hrs</span>
          </div>
          <div className="flex flex-col items-center">
            <span
              className="text-3xl font-black tabular-nums text-cosmic-blue"
              style={{ fontFamily: 'var(--font-display)' }}
            >
              {pad(m)}
            </span>
            <span className="text-xs text-text-muted uppercase tracking-widest">min</span>
          </div>
          <div className="flex flex-col items-center">
            <span
              className="text-3xl font-black tabular-nums text-cosmic-blue"
              style={{ fontFamily: 'var(--font-display)' }}
            >
              {pad(s)}
            </span>
            <span className="text-xs text-text-muted uppercase tracking-widest">sec</span>
          </div>
        </div>
      ) : (
        <p className="text-sm text-text-muted">Starting soon - refresh the page</p>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------
export function ChallengesPage() {
  const { id: urlChallengeId } = useParams<{ id?: string }>()
  const navigate = useNavigate()

  const [selectedTag, setSelectedTag] = useState<string | null>(null)

  const { data: challenges, isLoading } = useChallenges(selectedTag ?? undefined)
  const { data: tags } = useTags()
  const { data: compStatus } = useCompetitionStatus()
  const { user } = useAuthStore()

  const openModal = useCallback(
    (id: string) => {
      navigate(`/challenges/${id}`, { replace: false })
    },
    [navigate],
  )

  const closeModal = useCallback(() => {
    navigate('/challenges', { replace: true })
  }, [navigate])

  const navigateToChallenge = useCallback(
    (id: string) => {
      navigate(`/challenges/${id}`, { replace: true })
    },
    [navigate],
  )

  // Group by category
  const grouped = useMemo(() => {
    if (!challenges) return []
    const map = new Map<string, ChallengeResponse[]>()
    for (const c of challenges) {
      const cat = c.category ?? 'Uncategorized'
      const arr = map.get(cat) ?? []
      arr.push(c)
      map.set(cat, arr)
    }
    return Array.from(map.entries()).sort(([a], [b]) => a.localeCompare(b))
  }, [challenges])

  const totalChallenges = challenges?.length ?? 0
  const solvedChallenges = challenges?.filter((c) => c.solved).length ?? 0

  return (
    <VisibilityGate configKey="challenge_visibility">
      <div className="flex flex-col gap-6 max-w-7xl mx-auto px-4 py-6 sm:px-6 lg:px-8">
        {/* Page header */}
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
          <div>
            <h1
              className="text-2xl font-black text-text-primary"
              style={{ fontFamily: 'var(--font-display)' }}
            >
              Challenges
            </h1>
            {!isLoading && totalChallenges > 0 && (
              <p className="text-sm text-text-muted mt-0.5">
                {solvedChallenges} / {totalChallenges} solved
              </p>
            )}
          </div>

          {/* Progress bar */}
          {!isLoading && totalChallenges > 0 && (
            <div className="flex items-center gap-2 w-full sm:w-48">
              <div className="flex-1 h-1.5 rounded-full bg-space-border overflow-hidden">
                <div
                  className="h-full bg-stellar-green rounded-full transition-all duration-500"
                  style={{ width: `${(solvedChallenges / totalChallenges) * 100}%` }}
                />
              </div>
              <span className="text-xs font-mono text-text-muted shrink-0">
                {Math.round((solvedChallenges / totalChallenges) * 100)}%
              </span>
            </div>
          )}
        </div>

        {/* Tag filter */}
        <TagFilterBar tags={tags ?? []} selected={selectedTag} onSelect={setSelectedTag} />

        {/* Team-needed banner (flexible / teams_only mode, user has no team) */}
        {!isLoading && !user?.team_id && compStatus?.status === 'running' && (
          <div className="rounded-[var(--radius-lg)] border border-solar-yellow/30 bg-solar-yellow/5 p-4 flex items-center justify-between gap-4">
            <p className="text-sm text-text-primary">
              Join or create a team to start solving challenges.
            </p>
            <Link
              to="/team/enroll"
              className="shrink-0 px-3 py-1.5 rounded text-xs font-medium bg-solar-yellow/20 text-solar-yellow hover:bg-solar-yellow/30 transition-colors"
            >
              Join / Create Team
            </Link>
          </div>
        )}

        {/* Challenge grid */}
        {isLoading ? (
          <ChallengeGridSkeleton />
        ) : compStatus?.status === 'not_started' ? (
          <CompetitionCountdownBanner startTime={compStatus.start_time ?? ''} />
        ) : grouped.length === 0 ? (
          <EmptyState message="No challenges yet." />
        ) : (
          <div className="flex flex-col gap-8">
            {grouped.map(([category, items]) => (
              <CategoryGroup
                key={category}
                category={category}
                challenges={items}
                onOpen={openModal}
              />
            ))}
          </div>
        )}

        {/* Modal */}
        <ChallengeModal
          key={urlChallengeId}
          challengeId={urlChallengeId ?? null}
          onClose={closeModal}
          onNavigate={navigateToChallenge}
        />
      </div>
    </VisibilityGate>
  )
}
