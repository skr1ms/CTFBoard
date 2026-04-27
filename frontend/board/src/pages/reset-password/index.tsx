import { api, isApiError } from '@/shared/api/client'
import { validateEmail, validatePassword } from '@/shared/lib/validation'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { PasswordInput } from '@/shared/ui/password-input'
import { useMutation } from '@tanstack/react-query'
import { type FormEvent, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router'

function errorMessage(err: unknown): string {
  const code = isApiError(err) ? err.code : undefined
  switch (code) {
    case 'TOKEN_EXPIRED':
      return 'This reset link has expired. Please request a new one.'
    case 'TOKEN_NOT_FOUND':
      return 'Invalid reset link. Please request a new one.'
    case 'RATE_LIMIT_EXCEEDED':
      return 'Too many attempts. Please wait and try again.'
    default:
      return isApiError(err) ? err.message : 'Something went wrong.'
  }
}

// ---------------------------------------------------------------------------
// Step 1 - enter email
// ---------------------------------------------------------------------------

function RequestStep() {
  const [email, setEmail] = useState('')
  const [emailError, setEmailError] = useState<string | null>(null)
  const [sent, setSent] = useState(false)

  const { mutate, isPending, error } = useMutation({
    mutationFn: async () => {
      const { error } = await api.POST('/auth/forgot-password', { body: { email } })
      if (error) throw error
    },
    onSuccess: () => setSent(true),
  })

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault()
    const err = validateEmail(email)
    if (err) {
      setEmailError(err)
      return
    }
    setEmailError(null)
    mutate()
  }

  if (sent) {
    return (
      <div className="text-center flex flex-col items-center gap-4">
        <div className="text-5xl">📬</div>
        <h2 className="text-lg font-semibold text-text-primary">Check your inbox</h2>
        <p className="text-sm text-text-muted max-w-xs">
          If an account exists for <span className="text-text-secondary">{email}</span>, you'll
          receive a password reset link shortly.
        </p>
        <Link to="/login" className="text-sm text-cosmic-blue hover:underline">
          Back to Sign in
        </Link>
      </div>
    )
  }

  return (
    <form onSubmit={handleSubmit} noValidate className="flex flex-col gap-4">
      <Input
        label="Email"
        type="email"
        autoComplete="email"
        data-testid="reset-email"
        value={email}
        onChange={(e) => {
          setEmail(e.target.value)
          setEmailError(null)
        }}
        {...(emailError ? { error: emailError } : {})}
        placeholder="you@example.com"
      />
      {error && <p className="text-sm text-red-400">{errorMessage(error)}</p>}
      <Button
        type="submit"
        data-testid="reset-submit"
        variant="primary"
        loading={isPending}
        className="w-full"
      >
        Send reset link
      </Button>
      <Link
        to="/login"
        className="text-center text-sm text-text-muted hover:text-cosmic-blue transition-colors"
      >
        Back to Sign in
      </Link>
    </form>
  )
}

// ---------------------------------------------------------------------------
// Step 2 - set new password (token from URL)
// ---------------------------------------------------------------------------

function ResetStep({ token }: { token: string }) {
  const navigate = useNavigate()
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [passwordError, setPasswordError] = useState<string | null>(null)
  const [confirmError, setConfirmError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)

  const { mutate, isPending, error } = useMutation({
    mutationFn: async () => {
      const { error } = await api.POST('/auth/reset-password', {
        body: { token, new_password: password },
      })
      if (error) throw error
    },
    onSuccess: () => {
      setSuccess(true)
      setTimeout(() => navigate('/login'), 2500)
    },
  })

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault()
    let valid = true
    const pwErr = validatePassword(password)
    if (pwErr) {
      setPasswordError(pwErr)
      valid = false
    } else setPasswordError(null)
    if (!confirm) {
      setConfirmError('Please confirm your password.')
      valid = false
    } else if (confirm !== password) {
      setConfirmError('Passwords do not match.')
      valid = false
    } else setConfirmError(null)
    if (!valid) return
    mutate()
  }

  if (success) {
    return (
      <div className="text-center flex flex-col items-center gap-4">
        <div className="text-5xl">✅</div>
        <h2 className="text-lg font-semibold text-text-primary">Password updated</h2>
        <p className="text-sm text-text-muted">Redirecting to sign in…</p>
      </div>
    )
  }

  return (
    <form onSubmit={handleSubmit} noValidate className="flex flex-col gap-4">
      <PasswordInput
        label="New password"
        autoComplete="new-password"
        data-testid="reset-password"
        value={password}
        onChange={(e) => {
          setPassword(e.target.value)
          setPasswordError(null)
        }}
        {...(passwordError ? { error: passwordError } : {})}
        placeholder="Min. 8 characters"
      />
      <PasswordInput
        label="Confirm password"
        autoComplete="new-password"
        data-testid="reset-confirm"
        value={confirm}
        onChange={(e) => {
          setConfirm(e.target.value)
          setConfirmError(null)
        }}
        {...(confirmError ? { error: confirmError } : {})}
        placeholder="Repeat password"
      />
      {error && (
        <div className="rounded-[var(--radius-md)] border border-red-500/30 bg-red-500/10 px-3 py-2.5 text-sm text-red-400">
          {errorMessage(error)}
          {isApiError(error) &&
            (error.code === 'TOKEN_EXPIRED' || error.code === 'TOKEN_NOT_FOUND') && (
              <Link to="/reset-password" className="ml-1 underline">
                Request new link
              </Link>
            )}
        </div>
      )}
      <Button
        type="submit"
        data-testid="reset-submit-new"
        variant="primary"
        loading={isPending}
        className="w-full"
      >
        Set new password
      </Button>
    </form>
  )
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

export function ResetPasswordPage() {
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token')

  return (
    <div className="flex min-h-[calc(100vh-3.5rem)] items-center justify-center px-4 py-12">
      <div className="w-full max-w-sm">
        <div className="text-center mb-8">
          <div className="text-4xl mb-3">🔑</div>
          <h1
            className="text-2xl font-bold text-text-primary"
            style={{ fontFamily: 'var(--font-display)' }}
          >
            {token ? 'Set new password' : 'Reset password'}
          </h1>
          {!token && (
            <p className="text-sm text-text-muted mt-1">Enter your email to receive a reset link</p>
          )}
        </div>
        {token ? <ResetStep token={token} /> : <RequestStep />}
      </div>
    </div>
  )
}
