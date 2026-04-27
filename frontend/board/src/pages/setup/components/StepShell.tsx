import { Button } from '@/shared/ui/button'
import { type ReactNode } from 'react'

interface StepShellProps {
  title: string
  subtitle?: string
  children: ReactNode
  step: number
  totalSteps: number
  isLastStep: boolean
  isSubmitting: boolean
  onBack: () => void
  onNext: () => void
}

export function StepShell({
  title,
  subtitle,
  children,
  step,
  totalSteps,
  isLastStep,
  isSubmitting,
  onBack,
  onNext,
}: StepShellProps) {
  return (
    <div className="bg-surface-card border border-border-default rounded-xl shadow-lg overflow-hidden">
      <div className="px-6 pt-6 pb-4 border-b border-border-subtle">
        <h2
          className="text-lg font-semibold text-text-primary"
          style={{ fontFamily: 'var(--font-display)' }}
        >
          {title}
        </h2>
        {subtitle && <p className="text-sm text-text-muted mt-1">{subtitle}</p>}
      </div>

      <div className="px-6 py-6">{children}</div>

      <div className="px-6 pb-6 flex items-center justify-between gap-3">
        <Button variant="ghost" onClick={onBack} disabled={step === 0 || isSubmitting} size="md">
          Back
        </Button>

        <div className="flex items-center gap-2">
          <span className="text-xs text-text-muted">
            {step + 1} / {totalSteps}
          </span>
          <Button
            variant="primary"
            onClick={onNext}
            loading={isSubmitting && isLastStep}
            disabled={isSubmitting}
            size="md"
          >
            {isLastStep ? 'Finish' : 'Next'}
          </Button>
        </div>
      </div>
    </div>
  )
}
