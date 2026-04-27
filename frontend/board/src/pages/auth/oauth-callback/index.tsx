import { api, baseClient } from '@/shared/api/client'
import { useAuthStore } from '@/shared/stores/authStore'
import { Spinner } from '@/shared/ui/spinner'
import { useEffect, useRef } from 'react'
import { useNavigate, useSearchParams } from 'react-router'
import { toast } from 'sonner'

const ERROR_MESSAGES: Record<string, string> = {
  OAUTH_STATE_MISSING: 'OAuth login failed: session expired, please try again',
  OAUTH_STATE_MISMATCH: 'OAuth login failed: invalid state',
  OAUTH_PROVIDER_DISABLED: 'This OAuth provider is disabled',
  OAUTH_ACCOUNT_ALREADY_LINKED: 'This account is already linked to another user',
  OAUTH_EMAIL_TAKEN: 'An account with this email already exists',
  OAUTH_LINK_REQUIRES_AUTH: 'You must be logged in to link an OAuth account',
  access_denied: 'OAuth login was cancelled',
  INTERNAL_ERROR: 'OAuth login failed: internal error, please try again',
}

export function OAuthCallbackPage() {
  const navigate = useNavigate()
  // Backend redirects to /auth/callback?code=<one-time> on success, or ?error=<code> on failure.
  const [searchParams] = useSearchParams()
  const login = useAuthStore((s) => s.login)
  const setAccessToken = useAuthStore((s) => s.setAccessToken)
  const processed = useRef(false)

  useEffect(() => {
    if (processed.current) return
    processed.current = true

    const queryError = searchParams.get('error')
    if (queryError) {
      const message = ERROR_MESSAGES[queryError] ?? `OAuth login failed: ${queryError}`
      toast.error(message)
      void navigate('/login', { replace: true })
      return
    }

    const code = searchParams.get('code')
    if (!code) {
      toast.error('OAuth login failed: missing exchange code')
      void navigate('/login', { replace: true })
      return
    }

    const finalize = async () => {
      const { data: tokenData, error: exchangeError } = await baseClient.POST(
        '/auth/oauth/exchange',
        { body: { code } },
      )

      if (exchangeError || !tokenData?.access_token) {
        toast.error('OAuth login failed: code expired or already used')
        void navigate('/login', { replace: true })
        return
      }

      const accessToken = tokenData.access_token

      // Temporarily set the access token so the API client can authenticate the /auth/me call.
      setAccessToken(accessToken)

      const { data: me, error: meError } = await api.GET('/auth/me')

      if (meError || !me) {
        useAuthStore.setState({ accessToken: null })
        toast.error('OAuth login failed: could not fetch user info')
        void navigate('/login', { replace: true })
        return
      }

      login(accessToken, me)
      void navigate('/challenges', { replace: true })
    }

    void finalize()
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div className="flex items-center justify-center min-h-screen bg-space-bg">
      <div className="flex flex-col items-center gap-4">
        <Spinner size="lg" />
        <p className="text-sm text-text-muted">Completing sign-in…</p>
      </div>
    </div>
  )
}
