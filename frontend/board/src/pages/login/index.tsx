import { useOAuthProviders } from '@/features/auth/useOAuthProviders'
import { api, isApiError } from '@/shared/api/client'
import { env } from '@/shared/config/env'
import { validateEmail } from '@/shared/lib/validation'
import { useAuthStore } from '@/shared/stores/authStore'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { PasswordInput } from '@/shared/ui/password-input'
import { useMutation } from '@tanstack/react-query'
import { useState, type FormEvent } from 'react'
import { Link, useLocation, useNavigate } from 'react-router'
import { toast } from 'sonner'

// eslint-disable-next-line react-refresh/only-export-components
export function validateLoginFields(
  email: string,
  password: string,
): { emailError: string | null; passwordError: string | null } {
  const emailError = validateEmail(email)
  const passwordError = !password ? 'Password is required.' : null
  return { emailError, passwordError }
}

function OAuthButton({ provider, label, icon }: { provider: string; label: string; icon: string }) {
  const handleClick = () => {
    window.location.href = `${env.apiBaseUrl}/auth/oauth/${provider}`
  }
  return (
    <button
      type="button"
      onClick={handleClick}
      className="flex items-center justify-center gap-2 w-full px-4 py-2.5 rounded-[var(--radius-md)] border border-space-border bg-space-card text-sm text-text-secondary hover:text-text-primary hover:border-cosmic-blue/50 hover:bg-space-dark transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cosmic-blue"
    >
      <span>{icon}</span>
      <span>Continue with {label}</span>
    </button>
  )
}

export function LoginPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const login = useAuthStore((s) => s.login)
  const setAccessToken = useAuthStore((s) => s.setAccessToken)
  const { data: oauthProviders } = useOAuthProviders()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [emailError, setEmailError] = useState<string | null>(null)
  const [passwordError, setPasswordError] = useState<string | null>(null)

  const from = (location.state as { from?: string } | null)?.from ?? '/challenges'

  const {
    mutate,
    reset: resetMutation,
    isPending,
    error: mutError,
  } = useMutation({
    mutationFn: async () => {
      const { data, error } = await api.POST('/auth/login', {
        body: { email, password },
      })
      if (error) throw error
      return data
    },
    onSuccess: async (data) => {
      if (!data?.access_token) return
      // Set access token before /me so the auth middleware can attach it.
      setAccessToken(data.access_token)
      const { data: me, error: meErr } = await api.GET('/auth/me')
      if (meErr || !me) {
        toast.error('Login succeeded but failed to load profile. Please try again.')
        return
      }
      login(data.access_token, me)
      navigate(from, { replace: true })
    },
    onError: (err) => {
      if (isApiError(err) && err.code === 'RATE_LIMIT_EXCEEDED') {
        toast.error('Too many attempts. Please wait a minute and try again.', { duration: 10_000 })
      }
    },
  })

  const apiErr = isApiError(mutError) ? mutError : null
  const errorMessage = (() => {
    switch (apiErr?.code) {
      case 'INVALID_CREDENTIALS':
        return 'Invalid email or password.'
      case 'USER_BANNED':
        return apiErr.message ?? 'Your account has been banned.'
      case 'EMAIL_NOT_VERIFIED':
        return 'Please verify your email. Check your inbox for a verification link.'
      case 'RATE_LIMIT_EXCEEDED':
        return 'Too many attempts. Please wait and try again.'
      default:
        return apiErr ? (apiErr.message ?? 'Something went wrong.') : null
    }
  })()

  const validate = () => {
    const { emailError, passwordError } = validateLoginFields(email, password)
    setEmailError(emailError)
    setPasswordError(passwordError)
    return !emailError && !passwordError
  }

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault()
    if (!validate()) return
    mutate()
  }

  const showGithub = oauthProviders?.github === true
  const showGoogle = oauthProviders?.google === true
  const showOAuth = showGithub || showGoogle

  return (
    <div className="flex min-h-[calc(100vh-3.5rem)] items-center justify-center px-4 py-12">
      <div className="w-full max-w-sm">
        {/* Header */}
        <div className="text-center mb-8">
          <div className="text-4xl mb-3">✦</div>
          <h1
            className="text-2xl font-bold text-text-primary"
            style={{ fontFamily: 'var(--font-display)' }}
          >
            Sign in
          </h1>
          <p className="text-sm text-text-muted mt-1">Enter your credentials to continue</p>
        </div>

        {/* OAuth */}
        {showOAuth && (
          <div className="flex flex-col gap-2 mb-6">
            {showGithub && <OAuthButton provider="github" label="GitHub" icon="⬡" />}
            {showGoogle && <OAuthButton provider="google" label="Google" icon="G" />}
            <div className="flex items-center gap-3 my-1">
              <div className="flex-1 h-px bg-space-border" />
              <span className="text-xs text-text-muted">or</span>
              <div className="flex-1 h-px bg-space-border" />
            </div>
          </div>
        )}

        {/* Form */}
        <form onSubmit={handleSubmit} noValidate className="flex flex-col gap-4">
          <Input
            label="Email"
            type="email"
            autoComplete="email"
            data-testid="login-email"
            value={email}
            onChange={(e) => {
              setEmail(e.target.value)
              setEmailError(null)
              resetMutation()
            }}
            {...(emailError ? { error: emailError } : {})}
            placeholder="you@example.com"
          />

          <div className="flex flex-col gap-1">
            <PasswordInput
              label="Password"
              autoComplete="current-password"
              data-testid="login-password"
              value={password}
              onChange={(e) => {
                setPassword(e.target.value)
                setPasswordError(null)
                resetMutation()
              }}
              {...(passwordError ? { error: passwordError } : {})}
              placeholder="••••••••"
            />
            <div className="flex justify-end">
              <Link
                to="/reset-password"
                className="text-xs text-text-muted hover:text-cosmic-blue transition-colors"
              >
                Forgot password?
              </Link>
            </div>
          </div>

          {/* API error */}
          {errorMessage && (
            <div
              data-testid="login-error"
              className="rounded-[var(--radius-md)] border border-red-500/30 bg-red-500/10 px-3 py-2.5 text-sm text-red-400"
            >
              {errorMessage}
            </div>
          )}

          <Button
            type="submit"
            data-testid="login-submit"
            variant="primary"
            loading={isPending}
            className="w-full mt-1"
          >
            Sign in
          </Button>
        </form>

        {/* Footer */}
        <p className="text-center text-sm text-text-muted mt-6">
          Don&apos;t have an account?{' '}
          <Link to="/register" className="text-cosmic-blue hover:underline">
            Register
          </Link>
        </p>
      </div>
    </div>
  )
}
