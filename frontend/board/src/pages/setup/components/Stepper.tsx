import { cn } from '@/shared/lib/utils'
import { type SetupStep, STEP_LABELS } from '@/features/setup/types'

interface StepperProps {
  current: SetupStep
  total: number
}

export function Stepper({ current, total }: StepperProps) {
  const steps = Array.from({ length: total }, (_, i) => i) as SetupStep[]

  return (
    <div
      className="flex items-center justify-center gap-0 mb-8"
      role="list"
      aria-label="Setup steps"
    >
      {steps.map((step, idx) => {
        const isCompleted = step < current
        const isCurrent = step === current
        const isPending = step > current

        return (
          <div key={step} className="flex items-center" role="listitem">
            <div className="flex flex-col items-center gap-1.5">
              <div
                className={cn(
                  'w-8 h-8 rounded-full flex items-center justify-center text-xs font-semibold border-2 transition-all duration-200',
                  isCompleted && 'bg-nebula-purple border-nebula-purple text-white',
                  isCurrent &&
                    'bg-cosmic-blue border-cosmic-blue text-white shadow-[var(--glow-blue-sm)]',
                  isPending && 'bg-transparent border-border-muted text-text-muted',
                )}
                aria-current={isCurrent ? 'step' : undefined}
              >
                {isCompleted ? (
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2.5}
                      d="M5 13l4 4L19 7"
                    />
                  </svg>
                ) : (
                  <span>{step + 1}</span>
                )}
              </div>
              <span
                className={cn(
                  'text-xs font-medium whitespace-nowrap hidden sm:block',
                  isCompleted && 'text-nebula-purple',
                  isCurrent && 'text-cosmic-blue',
                  isPending && 'text-text-muted',
                )}
              >
                {STEP_LABELS[step]}
              </span>
            </div>

            {idx < total - 1 && (
              <div
                className={cn(
                  'h-px w-8 sm:w-12 mx-1 mb-5 transition-colors duration-200',
                  step < current ? 'bg-nebula-purple' : 'bg-border-muted',
                )}
              />
            )}
          </div>
        )
      })}
    </div>
  )
}
