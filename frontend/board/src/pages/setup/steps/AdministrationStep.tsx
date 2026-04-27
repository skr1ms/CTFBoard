import { Input } from '@/shared/ui/input'
import { PasswordInput } from '@/shared/ui/password-input'
import { type SetupFormData } from '@/features/setup/types'

interface AdministrationStepProps {
  data: SetupFormData
  errors: Partial<Record<keyof SetupFormData, string>>
  onChange: (patch: Partial<SetupFormData>) => void
}

export function AdministrationStep({ data, errors, onChange }: AdministrationStepProps) {
  return (
    <div className="flex flex-col gap-5">
      <Input
        label="Admin username"
        value={data.admin_username}
        onChange={(e) => onChange({ admin_username: e.target.value })}
        error={errors.admin_username}
        placeholder="admin"
        autoComplete="username"
      />
      <Input
        label="Admin email"
        type="email"
        value={data.admin_email}
        onChange={(e) => onChange({ admin_email: e.target.value })}
        error={errors.admin_email}
        placeholder="admin@example.com"
        autoComplete="email"
      />
      <PasswordInput
        label="Admin password"
        value={data.admin_password}
        onChange={(e) => onChange({ admin_password: e.target.value })}
        error={errors.admin_password}
        hint="Minimum 8 characters."
        autoComplete="new-password"
      />
    </div>
  )
}
