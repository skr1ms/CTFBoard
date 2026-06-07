import { useCompleteSetup } from '@/features/setup/useCompleteSetup'
import {
  SETUP_DEFAULTS,
  STEP_LABELS,
  type SetupFormData,
  type SetupStep,
} from '@/features/setup/types'
import { validateStep } from '@/features/setup/schema'
import { toast } from '@/shared/ui/toast'
import { useReducer } from 'react'
import { useNavigate } from 'react-router'
import { Stepper } from './components/Stepper'
import { StepShell } from './components/StepShell'
import { GeneralStep } from './steps/GeneralStep'
import { ModeStep } from './steps/ModeStep'
import { SettingsStep } from './steps/SettingsStep'
import { AdministrationStep } from './steps/AdministrationStep'
import { StyleStep } from './steps/StyleStep'
import { DateTimeStep } from './steps/DateTimeStep'

const TOTAL_STEPS = 6

interface WizardState {
  step: SetupStep
  data: SetupFormData
  errors: Partial<Record<keyof SetupFormData, string>>
}

type WizardAction =
  | { type: 'patch'; payload: Partial<SetupFormData> }
  | { type: 'next'; errors: Partial<Record<keyof SetupFormData, string>> }
  | { type: 'back' }
  | { type: 'clearErrors' }

function wizardReducer(state: WizardState, action: WizardAction): WizardState {
  switch (action.type) {
    case 'patch':
      return { ...state, data: { ...state.data, ...action.payload }, errors: {} }
    case 'next':
      if (Object.keys(action.errors).length > 0) {
        return { ...state, errors: action.errors }
      }
      return {
        ...state,
        step: Math.min(state.step + 1, TOTAL_STEPS - 1) as SetupStep,
        errors: {},
      }
    case 'back':
      return {
        ...state,
        step: Math.max(state.step - 1, 0) as SetupStep,
        errors: {},
      }
    case 'clearErrors':
      return { ...state, errors: {} }
  }
}

export function SetupPage() {
  const navigate = useNavigate()
  const mutation = useCompleteSetup()

  const [state, dispatch] = useReducer(wizardReducer, {
    step: 0,
    data: SETUP_DEFAULTS,
    errors: {},
  })

  const handleChange = (patch: Partial<SetupFormData>) => {
    dispatch({ type: 'patch', payload: patch })
  }

  const handleNext = () => {
    const errors = validateStep(state.step, state.data)
    dispatch({ type: 'next', errors })
  }

  const handleBack = () => {
    dispatch({ type: 'back' })
  }

  const handleFinish = () => {
    const errors = validateStep(state.step, state.data)
    if (Object.keys(errors).length > 0) {
      dispatch({ type: 'next', errors })
      return
    }

    mutation.mutate(state.data, {
      onSuccess: () => {
        void navigate('/admin', { replace: true })
      },
      onError: (err) => {
        const msg = err instanceof Error ? err.message : 'Setup failed. Please try again.'
        toast.error(msg)
      },
    })
  }

  const isLastStep = state.step === ((TOTAL_STEPS - 1) as SetupStep)
  const onNext = isLastStep ? handleFinish : handleNext

  return (
    <div className="flex flex-col gap-0">
      <Stepper current={state.step} total={TOTAL_STEPS} />

      <StepShell
        title={STEP_LABELS[state.step]}
        subtitle={getStepSubtitle(state.step)}
        step={state.step}
        totalSteps={TOTAL_STEPS}
        isLastStep={isLastStep}
        isSubmitting={mutation.isPending}
        onBack={handleBack}
        onNext={onNext}
      >
        {state.step === 0 && (
          <GeneralStep data={state.data} errors={state.errors} onChange={handleChange} />
        )}
        {state.step === 1 && (
          <ModeStep data={state.data} errors={state.errors} onChange={handleChange} />
        )}
        {state.step === 2 && (
          <SettingsStep data={state.data} errors={state.errors} onChange={handleChange} />
        )}
        {state.step === 3 && (
          <AdministrationStep data={state.data} errors={state.errors} onChange={handleChange} />
        )}
        {state.step === 4 && <StyleStep />}
        {state.step === 5 && (
          <DateTimeStep data={state.data} errors={state.errors} onChange={handleChange} />
        )}
      </StepShell>
    </div>
  )
}

function getStepSubtitle(step: SetupStep): string {
  switch (step) {
    case 0:
      return 'Name your competition and add a short description.'
    case 1:
      return 'Choose the primary competition format.'
    case 2:
      return 'Configure visibility and access defaults for your platform.'
    case 3:
      return 'Create the initial administrator account.'
    case 4:
      return 'Customise the look of your platform.'
    case 5:
      return 'Set competition start, end, and optional scoreboard freeze times.'
  }
}
