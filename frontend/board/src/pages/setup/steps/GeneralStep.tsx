import { Input } from '@/shared/ui/input'
import { type SetupFormData } from '@/features/setup/types'

interface GeneralStepProps {
  data: SetupFormData
  errors: Partial<Record<keyof SetupFormData, string>>
  onChange: (patch: Partial<SetupFormData>) => void
}

export function GeneralStep({ data, errors, onChange }: GeneralStepProps) {
  return (
    <div className="flex flex-col gap-5">
      <Input
        label="CTF Name"
        value={data.ctf_name}
        onChange={(e) => onChange({ ctf_name: e.target.value })}
        error={errors.ctf_name}
        placeholder="e.g. AstroCTF 2026"
        maxLength={100}
        required
      />
      <div className="flex flex-col gap-1.5">
        <label htmlFor="ctf-description" className="text-sm font-medium text-text-secondary">
          Description <span className="text-text-muted font-normal">(optional)</span>
        </label>
        <textarea
          id="ctf-description"
          value={data.ctf_description}
          onChange={(e) => onChange({ ctf_description: e.target.value })}
          placeholder="A short description of your competition…"
          rows={4}
          className="w-full rounded-md border border-border-default bg-deep-space px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue focus:border-transparent resize-none"
        />
      </div>
    </div>
  )
}
