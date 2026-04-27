import { useFlagSubmit } from '@/features/challenges/useFlagSubmit'
import { useCopyToClipboard } from '@/shared/lib/hooks/useCopyToClipboard'
import { cn } from '@/shared/lib/utils'
import { Button } from '@/shared/ui/button'
import { type KeyboardEvent, useState } from 'react'

interface FlagSubmitFormProps {
  challengeId: string
  maxAttempts?: number
  solved: boolean
  submissionAllowed?: boolean
  competitionStatus?: string
}

function submissionClosedMessage(status: string | undefined): string {
  if (status === 'ended') return 'Competition ended. Flag submissions are closed.'
  if (status === 'paused') return 'Competition paused. Submissions temporarily disabled.'
  if (status === 'not_started') return 'Competition has not started yet.'
  return 'Submissions are currently closed.'
}

export function FlagSubmitForm({
  challengeId,
  maxAttempts,
  solved,
  submissionAllowed = true,
  competitionStatus,
}: FlagSubmitFormProps) {
  const [flag, setFlag] = useState('')
  const { submitState, isPending, submit, reset } = useFlagSubmit(challengeId)
  const { copy, copied } = useCopyToClipboard()

  const handleSubmit = () => {
    if (!flag.trim() || isPending) return
    reset()
    submit(flag.trim())
  }

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') handleSubmit()
  }

  if (solved) {
    return (
      <div className="flex items-center justify-between gap-2 px-3 py-2 rounded-[var(--radius-md)] border border-green-500/30 bg-green-500/10 text-green-400 text-sm">
        <div className="flex items-center gap-2">
          <span>✓</span>
          <span>Challenge solved!</span>
        </div>
        {flag && (
          <button
            type="button"
            onClick={() => copy(flag)}
            className="text-xs text-green-400/70 hover:text-green-400 transition-colors px-2 py-0.5 rounded hover:bg-green-500/10 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-green-500"
          >
            {copied ? '✓ Copied' : 'Copy flag'}
          </button>
        )}
      </div>
    )
  }

  if (!submissionAllowed) {
    return (
      <div className="px-3 py-2 rounded-[var(--radius-md)] border border-space-border bg-space-dark text-text-muted text-sm">
        {submissionClosedMessage(competitionStatus)}
      </div>
    )
  }

  const isRateLimited = submitState.type === 'rate_limited'
  const isMaxAttempts = submitState.type === 'max_attempts'
  const disabled = isRateLimited || isMaxAttempts

  return (
    <div className="flex flex-col gap-2">
      <div className="flex gap-2">
        <input
          type="text"
          data-testid="flag-submit-input"
          value={flag}
          onChange={(e) => {
            setFlag(e.target.value)
            if (submitState.type !== 'idle') reset()
          }}
          onKeyDown={handleKeyDown}
          disabled={disabled || isPending}
          placeholder="CTF{flag_here}"
          className={cn(
            'flex-1 h-10 rounded-[var(--radius-md)] border bg-space-dark px-3 text-sm text-text-primary placeholder:text-text-muted',
            'focus:outline-none focus:ring-2 focus:ring-cosmic-blue/50 focus:border-cosmic-blue',
            'disabled:opacity-50 disabled:cursor-not-allowed',
            submitState.type === 'correct'
              ? 'border-green-500'
              : submitState.type === 'incorrect'
                ? 'border-red-500'
                : 'border-space-border',
          )}
        />
        <Button
          data-testid="flag-submit-button"
          variant="primary"
          size="sm"
          onClick={handleSubmit}
          loading={isPending}
          disabled={disabled}
        >
          Submit
        </Button>
      </div>

      <div data-testid="flag-submit-status">
        {submitState.type === 'correct' && (
          <p className="text-sm text-green-400 font-medium animate-pulse">✓ Correct! Well done!</p>
        )}
        {submitState.type === 'incorrect' && (
          <p className="text-sm text-red-400">✗ {submitState.message}</p>
        )}
        {submitState.type === 'rate_limited' && (
          <p className="text-sm text-amber-400">
            Rate limited. Wait <span className="font-mono font-bold">{submitState.seconds}s</span>{' '}
            before trying again.
          </p>
        )}
        {submitState.type === 'max_attempts' && (
          <p className="text-sm text-red-400">
            Maximum attempts reached. This challenge is locked.
          </p>
        )}
        {submitState.type === 'submission_closed' && (
          <p className="text-sm text-amber-400">{submitState.message}</p>
        )}
        {submitState.type === 'error' && (
          <p className="text-sm text-red-400">{submitState.message}</p>
        )}
        {maxAttempts && maxAttempts > 0 && (
          <p className="text-xs text-text-muted">Max {maxAttempts} attempts allowed.</p>
        )}
      </div>
    </div>
  )
}
