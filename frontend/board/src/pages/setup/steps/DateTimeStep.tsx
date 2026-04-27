import { type SetupFormData } from '@/features/setup/types'

interface DateTimeStepProps {
  data: SetupFormData
  errors: Partial<Record<keyof SetupFormData, string>>
  onChange: (patch: Partial<SetupFormData>) => void
}

interface DateFieldProps {
  label: string
  hint?: string | undefined
  value: string
  error?: string | undefined
  onChange: (v: string) => void
}

function isValidDateTimeLocal(s: string): boolean {
  if (!s) return true
  const d = new Date(s)
  if (Number.isNaN(d.getTime())) return false
  const [date] = s.split('T')
  if (!date) return false
  const [y, m, day] = date.split('-').map(Number)
  return d.getFullYear() === y && d.getMonth() + 1 === m && d.getDate() === day
}

function DateField({ label, hint, value, error, onChange }: DateFieldProps) {
  const localError = !isValidDateTimeLocal(value) ? 'Invalid date' : undefined
  const shownError = localError ?? error
  return (
    <div className="flex flex-col gap-1.5">
      <label className="text-sm font-medium text-text-secondary">
        {label} <span className="text-text-muted font-normal">(optional)</span>
      </label>
      {hint && <p className="text-xs text-text-muted -mt-0.5">{hint}</p>}
      <input
        type="datetime-local"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full rounded-md border border-border-default bg-deep-space px-3 py-2 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-cosmic-blue focus:border-transparent"
      />
      {shownError && <p className="text-xs text-red-400">{shownError}</p>}
    </div>
  )
}

function getTimezones(): string[] {
  const intlAny = Intl as unknown as { supportedValuesOf?: (k: string) => string[] }
  if (typeof intlAny.supportedValuesOf === 'function') {
    return intlAny.supportedValuesOf('timeZone')
  }
  return [Intl.DateTimeFormat().resolvedOptions().timeZone, 'UTC']
}

export function DateTimeStep({ data, errors, onChange }: DateTimeStepProps) {
  const timezones = getTimezones()
  return (
    <div className="flex flex-col gap-5">
      <DateField
        label="Start time"
        hint="When the competition opens for submissions."
        value={data.start_time}
        onChange={(v) => onChange({ start_time: v })}
      />
      <DateField
        label="End time"
        hint="When the competition closes."
        value={data.end_time}
        error={errors.end_time}
        onChange={(v) => onChange({ end_time: v })}
      />
      <DateField
        label="Freeze time"
        hint="Scoreboard stops updating at this point, but submissions still accepted until end."
        value={data.freeze_time}
        error={errors.freeze_time}
        onChange={(v) => onChange({ freeze_time: v })}
      />

      <div className="flex flex-col gap-1.5">
        <label className="text-sm font-medium text-text-secondary">Timezone</label>
        <select
          value={data.timezone}
          onChange={(e) => onChange({ timezone: e.target.value })}
          className="w-full rounded-md border border-border-default bg-deep-space px-3 py-2 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-cosmic-blue focus:border-transparent"
        >
          {timezones.map((tz) => (
            <option key={tz} value={tz}>
              {tz}
            </option>
          ))}
        </select>
        <p className="text-xs text-text-muted">Display timezone for competition times.</p>
      </div>
    </div>
  )
}
