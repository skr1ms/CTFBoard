import { type CompetitionMode, type SetupFormData } from '@/features/setup/types'
import { cn } from '@/shared/lib/utils'
import { Input } from '@/shared/ui/input'

interface ModeStepProps {
  data: SetupFormData
  errors: Partial<Record<keyof SetupFormData, string>>
  onChange: (patch: Partial<SetupFormData>) => void
}

const modes: { value: CompetitionMode; label: string; description: string }[] = [
  {
    value: 'teams_only',
    label: 'Teams only',
    description: 'Participants must form or join a team to compete.',
  },
  {
    value: 'solo_only',
    label: 'Solo only',
    description: 'Participants compete individually, no teams allowed.',
  },
  {
    value: 'flexible',
    label: 'Flexible',
    description: 'Participants can compete solo or in a team - their choice.',
  },
]

export function ModeStep({ data, errors, onChange }: ModeStepProps) {
  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-col gap-3">
        {modes.map((m) => (
          <label
            key={m.value}
            className={cn(
              'flex items-start gap-3 p-4 rounded-lg border-2 cursor-pointer transition-colors duration-150',
              data.mode === m.value
                ? 'border-cosmic-blue bg-cosmic-blue/5'
                : 'border-border-default hover:border-border-muted',
            )}
          >
            <input
              type="radio"
              name="mode"
              value={m.value}
              checked={data.mode === m.value}
              onChange={() => onChange({ mode: m.value })}
              className="mt-0.5 accent-cosmic-blue"
            />
            <div>
              <div className="text-sm font-semibold text-text-primary">{m.label}</div>
              <div className="text-xs text-text-muted mt-0.5">{m.description}</div>
            </div>
          </label>
        ))}
        {errors.mode && <p className="text-xs text-red-400">{errors.mode}</p>}
      </div>

      {data.mode !== 'solo_only' && (
        <Input
          label="Max team size"
          type="text"
          inputMode="numeric"
          value={data.max_team_size}
          onChange={(e) => {
            const v = e.target.value
            if (v === '' || /^\d{1,3}$/.test(v)) onChange({ max_team_size: v })
          }}
          error={errors.max_team_size}
          hint="How many members can be in a single team."
        />
      )}
    </div>
  )
}
