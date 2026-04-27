import { useUserFields, type FieldResponse } from '@/features/auth/useFields'
import { usePublicConfigs } from '@/features/auth/usePublicConfigs'
import { api, isApiError } from '@/shared/api/client'
import { validateEmail, validatePassword, validateUsername } from '@/shared/lib/validation'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { PasswordInput } from '@/shared/ui/password-input'
import { Spinner } from '@/shared/ui/spinner'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router'

// ---------------------------------------------------------------------------
// Pure validation (exported for unit tests)
// ---------------------------------------------------------------------------

// eslint-disable-next-line react-refresh/only-export-components
export function validateRegisterFields(
  username: string,
  email: string,
  password: string,
  confirm: string,
  fields: FieldResponse[],
  customValues: Record<string, string>,
): Record<string, string> {
  const errs: Record<string, string> = {}
  const usernameErr = validateUsername(username)
  if (usernameErr) errs['username'] = usernameErr
  const emailErr = validateEmail(email)
  if (emailErr) errs['email'] = emailErr
  const passwordErr = validatePassword(password)
  if (passwordErr) errs['password'] = passwordErr
  if (!confirm) errs['confirm'] = 'Please confirm your password.'
  else if (confirm !== password) errs['confirm'] = 'Passwords do not match.'
  fields.forEach((f) => {
    if (f.required && f.id && !customValues[f.id]) {
      errs[f.id] = `${f.name ?? 'Field'} is required.`
    }
  })
  return errs
}

// ---------------------------------------------------------------------------
// Dynamic field renderer
// ---------------------------------------------------------------------------

function DynamicField({
  field,
  value,
  onChange,
  error,
}: {
  field: FieldResponse
  value: string
  onChange: (v: string) => void
  error?: string
}) {
  const label = field.name ?? ''
  const required = field.required ?? false

  if (field.field_type === 'boolean') {
    return (
      <label className="flex items-center gap-2 text-sm text-text-secondary cursor-pointer">
        <input
          type="checkbox"
          checked={value === 'true'}
          onChange={(e) => onChange(e.target.checked ? 'true' : 'false')}
          className="w-4 h-4 rounded border-space-border bg-space-dark text-cosmic-blue focus:ring-cosmic-blue"
        />
        {label}
        {required && <span className="text-nova-red">*</span>}
        {error && <span className="text-xs text-nova-red ml-1">{error}</span>}
      </label>
    )
  }

  if (field.field_type === 'select' && field.options?.length) {
    return (
      <div className="flex flex-col gap-1.5">
        <label className="text-sm font-medium text-text-secondary">
          {label}
          {required && <span className="text-nova-red ml-0.5">*</span>}
        </label>
        <select
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className="h-10 w-full rounded-md border border-space-border bg-space-card px-3 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-cosmic-blue/50 focus:border-cosmic-blue"
        >
          <option value="">Select…</option>
          {field.options.map((opt) => (
            <option key={opt} value={opt}>
              {opt}
            </option>
          ))}
        </select>
        {error && <p className="text-xs text-nova-red">{error}</p>}
      </div>
    )
  }

  return (
    <Input
      label={`${label}${required ? ' *' : ''}`}
      type={field.field_type === 'number' ? 'number' : 'text'}
      value={value}
      onChange={(e) => onChange(e.target.value)}
      {...(error ? { error } : {})}
    />
  )
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

export function RegisterPage() {
  const navigate = useNavigate()
  const qc = useQueryClient()
  const { data: configs, isLoading: configsLoading } = usePublicConfigs()
  const { data: fields = [], isLoading: fieldsLoading } = useUserFields()

  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [customValues, setCustomValues] = useState<Record<string, string>>({})
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})
  const [successMsg, setSuccessMsg] = useState<string | null>(null)

  const clearFieldError = (key: string) =>
    setFieldErrors((prev) => {
      const n = { ...prev }
      delete n[key]
      return n
    })

  const {
    mutate,
    isPending,
    error: mutError,
  } = useMutation({
    mutationFn: async () => {
      const custom_fields: Record<string, string> = {}
      fields.forEach((f) => {
        const val = f.id ? customValues[f.id] : undefined
        if (f.id && val) custom_fields[f.id] = val
      })
      const { data, error } = await api.POST('/auth/register', {
        body: {
          username,
          email,
          password,
          ...(Object.keys(custom_fields).length > 0 ? { custom_fields } : {}),
        },
      })
      if (error) throw error
      return data
    },
    onSuccess: () => {
      setSuccessMsg('Registration successful! Check your email to verify your account.')
      setTimeout(() => navigate('/login', { state: { registered: true } }), 2500)
    },
    onError: (err) => {
      const code = isApiError(err) ? err.code : undefined
      if (code === 'USERNAME_TAKEN') {
        setFieldErrors((p) => ({ ...p, username: 'Username is already taken.' }))
      } else if (code === 'USER_ALREADY_EXISTS') {
        setFieldErrors((p) => ({ ...p, email: 'An account with this email already exists.' }))
      } else if (code === 'REGISTRATION_DISABLED') {
        // Admin disabled registration while user had the form open - refresh the
        // public configs so the page transitions to the "Registration Closed" view.
        void qc.invalidateQueries({ queryKey: ['configs', 'public'] })
      }
    },
  })

  const validate = (): boolean => {
    const errs = validateRegisterFields(username, email, password, confirm, fields, customValues)
    setFieldErrors(errs)
    return Object.keys(errs).length === 0
  }

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault()
    if (!validate()) return
    mutate()
  }

  const apiErr = isApiError(mutError) ? mutError : null
  const globalError = (() => {
    switch (apiErr?.code) {
      case 'USERNAME_TAKEN':
      case 'USER_ALREADY_EXISTS':
        return null // shown inline
      case 'RATE_LIMIT_EXCEEDED':
        return 'Too many attempts. Please wait and try again.'
      default:
        return apiErr ? (apiErr.message ?? 'Registration failed.') : null
    }
  })()

  if (configsLoading) {
    return (
      <div className="flex min-h-[calc(100vh-3.5rem)] items-center justify-center">
        <Spinner size="lg" />
      </div>
    )
  }

  if (configs?.registration_enabled === false) {
    return (
      <div className="flex min-h-[calc(100vh-3.5rem)] items-center justify-center px-4">
        <div className="text-center max-w-sm">
          <div className="text-5xl mb-4">🔒</div>
          <h1 className="text-xl font-bold text-text-primary mb-2">Registration Closed</h1>
          <p className="text-text-muted text-sm">
            Registration is currently disabled. Please check back later.
          </p>
          <Link to="/login" className="mt-4 inline-block text-sm text-cosmic-blue hover:underline">
            Back to Sign in
          </Link>
        </div>
      </div>
    )
  }

  return (
    <div className="flex min-h-[calc(100vh-3.5rem)] items-center justify-center px-4 py-12">
      <div className="w-full max-w-sm">
        <div className="text-center mb-8">
          <div className="text-4xl mb-3">✦</div>
          <h1
            className="text-2xl font-bold text-text-primary"
            style={{ fontFamily: 'var(--font-display)' }}
          >
            Create account
          </h1>
          <p className="text-sm text-text-muted mt-1">Join the competition</p>
        </div>

        {successMsg ? (
          <div className="rounded-[var(--radius-md)] border border-green-500/30 bg-green-500/10 px-4 py-3 text-sm text-green-400 text-center">
            {successMsg}
          </div>
        ) : (
          <form onSubmit={handleSubmit} noValidate className="flex flex-col gap-4">
            <Input
              label="Username"
              autoComplete="username"
              data-testid="register-username"
              value={username}
              onChange={(e) => {
                setUsername(e.target.value)
                clearFieldError('username')
              }}
              {...(fieldErrors.username ? { error: fieldErrors.username } : {})}
              placeholder="player1"
            />

            <Input
              label="Email"
              type="email"
              autoComplete="email"
              data-testid="register-email"
              value={email}
              onChange={(e) => {
                setEmail(e.target.value)
                clearFieldError('email')
              }}
              {...(fieldErrors.email ? { error: fieldErrors.email } : {})}
              placeholder="you@example.com"
            />

            <PasswordInput
              label="Password"
              autoComplete="new-password"
              data-testid="register-password"
              value={password}
              onChange={(e) => {
                setPassword(e.target.value)
                clearFieldError('password')
              }}
              {...(fieldErrors.password ? { error: fieldErrors.password } : {})}
              placeholder="Min. 8 characters"
            />

            <PasswordInput
              label="Confirm password"
              autoComplete="new-password"
              data-testid="register-confirm"
              value={confirm}
              onChange={(e) => {
                setConfirm(e.target.value)
                clearFieldError('confirm')
              }}
              {...(fieldErrors.confirm ? { error: fieldErrors.confirm } : {})}
              placeholder="Repeat password"
            />

            {/* Dynamic fields */}
            {!fieldsLoading &&
              fields.map((f) => {
                const fid = f.id ?? ''
                const ferr = fid ? fieldErrors[fid] : undefined
                return (
                  <DynamicField
                    key={fid || f.name}
                    field={f}
                    value={fid ? (customValues[fid] ?? '') : ''}
                    onChange={(v) => {
                      if (fid) {
                        setCustomValues((prev) => ({ ...prev, [fid]: v }))
                        clearFieldError(fid)
                      }
                    }}
                    {...(ferr ? { error: ferr } : {})}
                  />
                )
              })}

            {globalError && (
              <div
                data-testid="register-error"
                className="rounded-[var(--radius-md)] border border-red-500/30 bg-red-500/10 px-3 py-2.5 text-sm text-red-400"
              >
                {globalError}
              </div>
            )}

            <Button
              type="submit"
              data-testid="register-submit"
              variant="primary"
              loading={isPending}
              className="w-full mt-1"
            >
              Create account
            </Button>
          </form>
        )}

        <p className="text-center text-sm text-text-muted mt-6">
          Already have an account?{' '}
          <Link to="/login" className="text-cosmic-blue hover:underline">
            Sign in
          </Link>
        </p>
      </div>
    </div>
  )
}
