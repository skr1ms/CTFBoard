import { type FileItem, useChallengeDetail } from '@/features/challenges/useChallenges'
import { useCompetitionStatus } from '@/features/competition/useCompetitionStatus'
import { useKeyboardShortcut } from '@/shared/lib/hooks/useKeyboardShortcut'
import { Badge } from '@/shared/ui/badge'
import { CodeBlock } from '@/shared/ui/code-block'
import { MarkdownRenderer } from '@/shared/ui/markdown-renderer'
import { Modal } from '@/shared/ui/modal'
import { Skeleton, SkeletonText } from '@/shared/ui/skeleton'
import { useCallback, useState } from 'react'
import { FlagSubmitForm } from './FlagSubmitForm'
import { HintsSection } from './HintsSection'
import { CommentsSection, RatingsSection, SolutionSection } from './PostCompSection'
import { RequirementsSection } from './RequirementsSection'

type ModalTab = 'info' | 'hints' | 'requirements' | 'comments' | 'ratings' | 'solution'

const CATEGORY_COLORS: Record<string, 'blue' | 'purple' | 'green' | 'cyan' | 'yellow' | 'red'> = {
  web: 'blue',
  crypto: 'purple',
  pwn: 'red',
  rev: 'yellow',
  reverse: 'yellow',
  forensics: 'cyan',
  misc: 'green',
  osint: 'cyan',
  network: 'blue',
  blockchain: 'purple',
  mobile: 'green',
}

// eslint-disable-next-line react-refresh/only-export-components
export function categoryBadgeVariant(
  category?: string,
): 'blue' | 'purple' | 'green' | 'cyan' | 'yellow' | 'red' | 'default' {
  if (!category) return 'default'
  return CATEGORY_COLORS[category.toLowerCase()] ?? 'default'
}

interface ChallengeModalProps {
  challengeId: string | null
  onClose: () => void
  onNavigate: (id: string) => void
}

function FirstBloodBadge({ name }: { name: string }) {
  return (
    <div className="flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-solar-yellow/10 border border-solar-yellow/30 text-solar-yellow text-xs font-medium">
      <svg className="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor">
        <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 0 0 .95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 0 0-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 0 0-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 0 0-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 0 0 .951-.69l1.07-3.292Z" />
      </svg>
      First blood: {name}
    </div>
  )
}

function ModalSkeleton() {
  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-start justify-between gap-4">
        <Skeleton className="h-7 w-48" />
        <Skeleton className="h-8 w-20 rounded-full" />
      </div>
      <div className="flex gap-2">
        <Skeleton className="h-5 w-16 rounded-full" />
        <Skeleton className="h-5 w-20 rounded-full" />
      </div>
      <SkeletonText lines={4} />
      <Skeleton className="h-10 w-full" />
    </div>
  )
}

function FilesSection({ files }: { files: FileItem[] }) {
  if (!files || files.length === 0) return null
  return (
    <div className="flex flex-col gap-2">
      <h4 className="text-xs font-semibold text-text-muted uppercase tracking-wide">Files</h4>
      <div className="flex flex-wrap gap-2">
        {files.map((f) => (
          <a
            key={f.id}
            href={f.url ?? '#'}
            download={f.filename}
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-2 px-3 py-2 rounded-[var(--radius-md)] border border-space-border
              bg-space-dark hover:border-cosmic-blue/50 hover:bg-cosmic-blue/5 transition-colors text-sm text-text-secondary hover:text-text-primary"
          >
            <svg
              className="w-4 h-4 shrink-0 text-cosmic-blue"
              viewBox="0 0 16 16"
              fill="currentColor"
            >
              <path d="M2.5 3.5a.5.5 0 0 1 .5-.5h6.793a.5.5 0 0 1 .353.146l2.707 2.707A.5.5 0 0 1 13 6.207V12.5a.5.5 0 0 1-.5.5h-10a.5.5 0 0 1-.5-.5v-9Z" />
            </svg>
            <span className="truncate max-w-[160px]">{f.filename ?? 'file'}</span>
            {f.size && (
              <span className="text-xs text-text-muted shrink-0">
                {f.size > 1024 * 1024
                  ? `${(f.size / 1024 / 1024).toFixed(1)} MB`
                  : `${(f.size / 1024).toFixed(1)} KB`}
              </span>
            )}
          </a>
        ))}
      </div>
    </div>
  )
}

function SolveStats({ solveCount, maxAttempts }: { solveCount?: number; maxAttempts?: number }) {
  if (solveCount === undefined && !maxAttempts) return null
  return (
    <div className="flex items-center gap-4 text-xs text-text-muted">
      {typeof solveCount === 'number' && (
        <span className="flex items-center gap-1">
          <svg className="w-3.5 h-3.5" viewBox="0 0 16 16" fill="currentColor">
            <path d="M8 15A7 7 0 1 1 8 1a7 7 0 0 1 0 14Zm0 1A8 8 0 1 0 8 0a8 8 0 0 0 0 16Z" />
            <path d="M10.97 4.97a.235.235 0 0 0-.02.022L7.477 9.417 5.384 7.323a.75.75 0 0 0-1.06 1.06L6.97 11.03a.75.75 0 0 0 1.079-.02l3.992-4.99a.75.75 0 0 0-.01-1.05l-.03-.02Z" />
          </svg>
          {solveCount} {solveCount === 1 ? 'solve' : 'solves'}
        </span>
      )}
      {maxAttempts && maxAttempts > 0 && (
        <span className="flex items-center gap-1">
          <svg className="w-3.5 h-3.5" viewBox="0 0 16 16" fill="currentColor">
            <path d="M8 16A8 8 0 1 0 8 0a8 8 0 0 0 0 16Zm.93-9.412-1 4.705c-.07.34.029.533.304.533.194 0 .487-.07.686-.246l-.088.416c-.287.346-.92.598-1.465.598-.703 0-1.002-.422-.808-1.319l.738-3.468c.064-.293.006-.399-.287-.47l-.451-.081.082-.381 2.29-.287zM8 5.5a1 1 0 1 1 0-2 1 1 0 0 1 0 2Z" />
          </svg>
          Max {maxAttempts} attempts
        </span>
      )}
    </div>
  )
}

export function ChallengeModal({ challengeId, onClose, onNavigate }: ChallengeModalProps) {
  const [tab, setTab] = useState<ModalTab>('info')

  // Tab resets to 'info' automatically via key={challengeId} on the parent mount site.
  const { data, isLoading } = useChallengeDetail(challengeId)
  const { data: compStatus } = useCompetitionStatus()

  const isLocked = data?.state === 'locked'
  const hints = data?.hints ?? []
  const hasHints = hints.length > 0
  const isEnded = compStatus?.status === 'ended'
  const submissionAllowed = compStatus?.submission_allowed ?? true

  // '?' jumps to hints tab when modal is open and hints are available
  const jumpToHints = useCallback(() => {
    if (challengeId && !isLocked && hasHints) setTab('hints')
  }, [challengeId, isLocked, hasHints])
  useKeyboardShortcut('?', jumpToHints, { disabled: !challengeId })

  const tabList: Array<{ value: ModalTab; label: string; show: boolean }> = [
    { value: 'info', label: 'Info', show: true },
    { value: 'hints', label: `Hints${hasHints ? ` (${hints.length})` : ''}`, show: !isLocked },
    { value: 'requirements', label: 'Requirements', show: isLocked },
    { value: 'comments', label: 'Comments', show: isEnded },
    { value: 'ratings', label: 'Ratings', show: isEnded },
    { value: 'solution', label: 'Solution', show: isEnded },
  ]

  return (
    <Modal
      open={!!challengeId}
      onClose={onClose}
      maxWidth="max-w-2xl"
      data-testid="challenge-modal"
    >
      {isLoading || !data ? (
        <ModalSkeleton />
      ) : (
        <div className="flex flex-col gap-5">
          {/* Header */}
          <div className="flex items-start justify-between gap-4">
            <div className="flex flex-col gap-1.5 min-w-0 flex-1">
              <h2
                className="text-xl font-bold text-text-primary leading-tight"
                style={{ fontFamily: 'var(--font-display)' }}
              >
                {data.title}
              </h2>
              <div className="flex flex-wrap items-center gap-1.5">
                {data.category && (
                  <Badge variant={categoryBadgeVariant(data.category)}>{data.category}</Badge>
                )}
                {data.tags?.map((tag) => (
                  <Badge key={tag.id} variant="default">
                    {tag.name}
                  </Badge>
                ))}
                {isLocked && <Badge variant="red">Locked</Badge>}
                {data.solved_by_me && <Badge variant="green">Solved</Badge>}
              </div>
            </div>

            <div className="flex items-start gap-3 shrink-0">
              <div className="flex flex-col items-end gap-1.5">
                <span
                  className="text-3xl font-black tabular-nums"
                  style={{
                    fontFamily: 'var(--font-display)',
                    color: 'var(--color-cosmic-blue)',
                  }}
                >
                  {data.points ?? 0}
                </span>
                <span className="text-xs text-text-muted -mt-1">pts</span>
              </div>
              <button
                onClick={onClose}
                aria-label="Close modal"
                data-testid="challenge-modal-close"
                className="text-text-muted hover:text-text-primary transition-colors p-1 rounded hover:bg-space-border focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cosmic-blue"
              >
                <svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor">
                  <path d="M3.72 3.72a.75.75 0 0 1 1.06 0L8 6.94l3.22-3.22a.749.749 0 0 1 1.275.326.749.749 0 0 1-.215.734L9.06 8l3.22 3.22a.749.749 0 0 1-.326 1.275.749.749 0 0 1-.734-.215L8 9.06l-3.22 3.22a.751.751 0 0 1-1.042-.018.751.751 0 0 1-.018-1.042L6.94 8 3.72 4.78a.75.75 0 0 1 0-1.06Z" />
                </svg>
              </button>
            </div>
          </div>

          {/* First blood */}
          {data.first_blood && (data.first_blood.username ?? data.first_blood.team_name) && (
            <FirstBloodBadge name={(data.first_blood.username ?? data.first_blood.team_name)!} />
          )}

          {/* Solve stats */}
          <SolveStats
            {...(data.solve_count !== undefined ? { solveCount: data.solve_count } : {})}
            {...(data.max_attempts !== undefined ? { maxAttempts: data.max_attempts } : {})}
          />

          {/* Tab bar */}
          <div className="flex gap-0 border-b border-space-border -mb-1">
            {tabList
              .filter((t) => t.show)
              .map((t) => (
                <button
                  key={t.value}
                  data-testid={`challenge-modal-tab-${t.value}`}
                  onClick={() => setTab(t.value)}
                  className={`px-4 py-2 text-sm font-medium whitespace-nowrap transition-colors border-b-2 -mb-px
                    focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cosmic-blue rounded-t
                    ${
                      tab === t.value
                        ? 'border-cosmic-blue text-cosmic-blue'
                        : 'border-transparent text-text-secondary hover:text-text-primary hover:border-space-border'
                    }`}
                >
                  {t.label}
                </button>
              ))}
          </div>

          {/* Tab content */}
          {tab === 'info' && (
            <div className="flex flex-col gap-4">
              {data.description && <MarkdownRenderer content={data.description} />}

              {data.connection_info && (
                <div className="flex flex-col gap-1.5">
                  <h4 className="text-xs font-semibold text-text-muted uppercase tracking-wide">
                    Connection
                  </h4>
                  <CodeBlock code={data.connection_info} language="text" />
                </div>
              )}

              {data.files && <FilesSection files={data.files} />}

              {isLocked ? (
                <div className="flex items-center gap-2 px-4 py-3 rounded-[var(--radius-md)] border border-nova-red/30 bg-nova-red/10 text-nova-red text-sm">
                  <svg className="w-4 h-4 shrink-0" viewBox="0 0 16 16" fill="currentColor">
                    <path d="M8 1a2 2 0 0 1 2 2v4H6V3a2 2 0 0 1 2-2zm3 6V3a3 3 0 0 0-6 0v4a2 2 0 0 0-2 2v5a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2z" />
                  </svg>
                  This challenge is locked. Complete the required challenges first.
                </div>
              ) : (
                <div className="flex flex-col gap-2">
                  <h4 className="text-xs font-semibold text-text-muted uppercase tracking-wide">
                    Submit Flag
                  </h4>
                  <FlagSubmitForm
                    key={data.id}
                    challengeId={data.id ?? ''}
                    solved={data.solved_by_me ?? false}
                    submissionAllowed={submissionAllowed}
                    {...(compStatus?.status !== undefined
                      ? { competitionStatus: compStatus.status }
                      : {})}
                    {...(data.max_attempts ? { maxAttempts: data.max_attempts } : {})}
                  />
                </div>
              )}
            </div>
          )}

          {tab === 'hints' && !isLocked && (
            <HintsSection challengeId={data.id ?? ''} hints={hints} />
          )}

          {tab === 'requirements' && isLocked && (
            <RequirementsSection
              challengeId={data.id ?? ''}
              onNavigate={(id) => {
                onNavigate(id)
              }}
            />
          )}

          {tab === 'comments' && isEnded && <CommentsSection challengeId={data.id ?? ''} />}

          {tab === 'ratings' && isEnded && (
            <RatingsSection challengeId={data.id ?? ''} solved={data.solved_by_me ?? false} />
          )}

          {tab === 'solution' && isEnded && <SolutionSection challengeId={data.id ?? ''} />}
        </div>
      )}
    </Modal>
  )
}
