import type { ChallengeRequirement } from '@/features/challenges/useChallenges'
import { useChallengeRequirements } from '@/features/challenges/useChallenges'
import { Skeleton } from '@/shared/ui/skeleton'

interface RequirementsSectionProps {
  challengeId: string
  onNavigate: (id: string) => void
}

function RequirementsList({
  requirements,
  onNavigate,
}: {
  requirements: ChallengeRequirement[]
  onNavigate: (id: string) => void
}) {
  return (
    <div className="flex flex-col gap-2">
      {requirements.map((req) => (
        <button
          key={req.challenge_id}
          onClick={() => req.challenge_id && onNavigate(req.challenge_id)}
          className="flex items-center gap-3 px-4 py-3 rounded-[var(--radius-md)] border border-space-border
            bg-space-card hover:border-cosmic-blue/50 hover:bg-cosmic-blue/5 transition-colors text-left group"
        >
          <svg className="w-4 h-4 text-nova-red shrink-0" viewBox="0 0 20 20" fill="currentColor">
            <path
              fillRule="evenodd"
              d="M10 1a4.5 4.5 0 0 0-4.5 4.5V9H5a2 2 0 0 0-2 2v6a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2v-6a2 2 0 0 0-2-2h-.5V5.5A4.5 4.5 0 0 0 10 1Zm3 8V5.5a3 3 0 1 0-6 0V9h6Z"
              clipRule="evenodd"
            />
          </svg>
          <div className="flex-1 min-w-0">
            <p className="text-sm font-medium text-text-primary group-hover:text-cosmic-blue transition-colors truncate">
              {req.challenge_title ?? 'Unknown challenge'}
            </p>
            {req.challenge_category && (
              <p className="text-xs text-text-muted">{req.challenge_category}</p>
            )}
          </div>
          <svg
            className="w-4 h-4 text-text-muted group-hover:text-cosmic-blue transition-colors shrink-0"
            viewBox="0 0 16 16"
            fill="currentColor"
          >
            <path
              fillRule="evenodd"
              d="M6.22 3.22a.75.75 0 0 1 1.06 0l4.25 4.25a.75.75 0 0 1 0 1.06l-4.25 4.25a.751.751 0 0 1-1.042-.018.751.751 0 0 1-.018-1.042L9.94 8 6.22 4.28a.75.75 0 0 1 0-1.06Z"
              clipRule="evenodd"
            />
          </svg>
        </button>
      ))}
    </div>
  )
}

export function RequirementsSection({ challengeId, onNavigate }: RequirementsSectionProps) {
  const { data, isLoading } = useChallengeRequirements(challengeId)

  if (isLoading) {
    return (
      <div className="flex flex-col gap-2">
        <Skeleton className="h-14 w-full" />
        <Skeleton className="h-14 w-full" />
      </div>
    )
  }

  const reqs = data ?? []

  if (reqs.length === 0) {
    return <p className="text-sm text-text-muted italic">No requirements for this challenge.</p>
  }

  return (
    <div className="flex flex-col gap-4">
      <p className="text-sm text-text-secondary">
        Complete the following challenges first to unlock this one:
      </p>
      <RequirementsList requirements={reqs} onNavigate={onNavigate} />
    </div>
  )
}
