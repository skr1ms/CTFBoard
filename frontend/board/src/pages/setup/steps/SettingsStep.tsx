import { type SetupFormData, type Visibility } from '@/features/setup/types'

interface SettingsStepProps {
  data: SetupFormData
  errors: Partial<Record<keyof SetupFormData, string>>
  onChange: (patch: Partial<SetupFormData>) => void
}

const visibilityOptions: { value: Visibility; label: string }[] = [
  { value: 'public', label: 'Public - visible to everyone' },
  { value: 'private', label: 'Private - visible after login' },
  { value: 'hidden', label: 'Hidden - not listed anywhere' },
  { value: 'admins', label: 'Admins only' },
]

interface VisibilitySelectProps {
  label: string
  hint: string
  value: Visibility
  onChange: (v: Visibility) => void
}

function VisibilitySelect({ label, hint, value, onChange }: VisibilitySelectProps) {
  return (
    <div className="flex flex-col gap-1.5">
      <label className="text-sm font-medium text-text-secondary">{label}</label>
      {hint && <p className="text-xs text-text-muted -mt-0.5">{hint}</p>}
      <select
        value={value}
        onChange={(e) => onChange(e.target.value as Visibility)}
        className="w-full rounded-md border border-border-default bg-deep-space px-3 py-2 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-cosmic-blue focus:border-transparent"
      >
        {visibilityOptions.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </select>
    </div>
  )
}

export function SettingsStep({ data, onChange }: SettingsStepProps) {
  return (
    <div className="flex flex-col gap-5">
      <VisibilitySelect
        label="Challenge visibility"
        hint="Who can see the challenge list."
        value={data.challenge_visibility}
        onChange={(v) => onChange({ challenge_visibility: v })}
      />
      <VisibilitySelect
        label="Score visibility"
        hint="Who can view the scoreboard."
        value={data.score_visibility}
        onChange={(v) => onChange({ score_visibility: v })}
      />
      <VisibilitySelect
        label="Account visibility"
        hint="Who can browse user and team profiles."
        value={data.account_visibility}
        onChange={(v) => onChange({ account_visibility: v })}
      />
      <VisibilitySelect
        label="Registration visibility"
        hint="Whether new registrations are open to the public."
        value={data.registration_visibility}
        onChange={(v) => onChange({ registration_visibility: v })}
      />

      <label className="flex items-center justify-between p-4 rounded-lg border border-border-default bg-deep-space/50 cursor-pointer">
        <div>
          <p className="text-sm font-medium text-text-primary">Email verification</p>
          <p className="text-xs text-text-muted mt-0.5">
            Require users to verify their email on signup.
          </p>
        </div>
        <input
          type="checkbox"
          checked={data.email_verification_required}
          onChange={(e) => onChange({ email_verification_required: e.target.checked })}
          className="accent-cosmic-blue h-5 w-9"
        />
      </label>
    </div>
  )
}
