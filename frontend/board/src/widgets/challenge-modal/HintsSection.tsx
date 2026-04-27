import type { HintItem } from '@/features/challenges/useChallenges'
import { useHintUnlock } from '@/features/challenges/useHintUnlock'
import { ConfirmationDialog } from '@/shared/ui/confirmation-dialog'
import { MarkdownRenderer } from '@/shared/ui/markdown-renderer'
import { useState } from 'react'

interface HintsSectionProps {
  challengeId: string
  hints: HintItem[]
}

interface HintRowProps {
  hint: HintItem
  onUnlock: (hintId: string) => void
  isUnlocking: boolean
  pendingHintId: string | null
}

function HintRow({ hint, onUnlock, isUnlocking, pendingHintId }: HintRowProps) {
  const [confirmOpen, setConfirmOpen] = useState(false)
  const id = hint.id ?? ''
  const cost = hint.cost ?? 0
  const isPending = isUnlocking && pendingHintId === id

  return (
    <>
      <div className="border border-space-border rounded-[var(--radius-md)] overflow-hidden">
        <div className="flex items-center justify-between px-4 py-3 bg-space-card">
          <div className="flex items-center gap-2">
            <svg
              className="w-4 h-4 text-solar-yellow shrink-0"
              viewBox="0 0 20 20"
              fill="currentColor"
            >
              <path d="M10 2a6 6 0 0 1 6 6c0 2.22-1.2 4.16-3 5.2V15a1 1 0 0 1-1 1H8a1 1 0 0 1-1-1v-1.8A6.002 6.002 0 0 1 10 2Zm-1 15h2v1a1 1 0 1 1-2 0v-1Z" />
            </svg>
            <span className="text-sm font-medium text-text-primary">
              {hint.title ?? `Hint ${hint.order_index ?? 1}`}
            </span>
          </div>

          {hint.unlocked ? (
            <span className="text-xs text-stellar-green font-medium">Unlocked</span>
          ) : (
            <button
              data-testid={`hint-unlock-${id}`}
              onClick={() => setConfirmOpen(true)}
              disabled={isPending}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-[var(--radius-sm)] text-xs font-medium transition-colors
                bg-solar-yellow/10 border border-solar-yellow/30 text-solar-yellow
                hover:bg-solar-yellow/20 disabled:opacity-40 disabled:cursor-not-allowed"
            >
              <svg className="w-3 h-3" viewBox="0 0 16 16" fill="currentColor">
                <path d="M8 1a2 2 0 0 1 2 2v4H6V3a2 2 0 0 1 2-2zm3 6V3a3 3 0 0 0-6 0v4a2 2 0 0 0-2 2v5a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2z" />
              </svg>
              {cost} pts
            </button>
          )}
        </div>

        {hint.unlocked && hint.content && (
          <div
            data-testid={`hint-content-${id}`}
            className="px-4 py-3 border-t border-space-border bg-space-dark/50"
          >
            <MarkdownRenderer content={hint.content} className="text-sm" />
          </div>
        )}
      </div>

      <ConfirmationDialog
        open={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        onConfirm={() => {
          setConfirmOpen(false)
          onUnlock(id)
        }}
        title="Unlock hint?"
        message={
          <span>
            This hint costs <span className="font-semibold text-solar-yellow">{cost} points</span>.
          </span>
        }
        confirmLabel="Unlock"
        loading={isPending}
      />
    </>
  )
}

export function HintsSection({ challengeId, hints }: HintsSectionProps) {
  const { mutate, isPending, variables } = useHintUnlock(challengeId)

  const sorted = [...hints].sort((a, b) => (a.order_index ?? 0) - (b.order_index ?? 0))

  if (sorted.length === 0) {
    return <p className="text-sm text-text-muted italic">No hints available for this challenge.</p>
  }

  return (
    <div className="flex flex-col gap-3">
      {sorted.map((hint) => (
        <HintRow
          key={hint.id}
          hint={hint}
          onUnlock={(id) => mutate(id)}
          isUnlocking={isPending}
          pendingHintId={variables ?? null}
        />
      ))}
    </div>
  )
}
