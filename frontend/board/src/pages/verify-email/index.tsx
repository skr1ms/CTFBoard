import { api, apiErrorMessage, isApiError } from '@/shared/api/client'
import { Spinner } from '@/shared/ui/spinner'
import { useMutation } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router'

export function VerifyEmailPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token')
  const [status, setStatus] = useState<'idle' | 'verifying' | 'success' | 'error'>(
    token ? 'idle' : 'error',
  )
  const [errMsg, setErrMsg] = useState<string | null>(null)

  const { mutate: verify } = useMutation({
    mutationFn: async (t: string) => {
      const { error } = await api.POST('/auth/verify-email', { body: { token: t } })
      if (error) throw error
    },
    onMutate: () => setStatus('verifying'),
    onSuccess: () => {
      setStatus('success')
      setTimeout(() => navigate('/login', { state: { verified: true } }), 3000)
    },
    onError: (err) => {
      const code = isApiError(err) ? err.code : undefined
      setStatus('error')
      switch (code) {
        case 'TOKEN_EXPIRED':
          setErrMsg('This verification link has expired.')
          break
        case 'TOKEN_NOT_FOUND':
          setErrMsg('Invalid or already used verification link.')
          break
        default:
          setErrMsg(apiErrorMessage(err, 'Verification failed.'))
      }
    },
  })

  useEffect(() => {
    if (token) {
      // Remove token from URL and browser history before making the request
      window.history.replaceState(null, '', window.location.pathname)
      verify(token)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return (
    <div className="flex min-h-[calc(100vh-3.5rem)] items-center justify-center px-4 py-12">
      <div className="w-full max-w-sm text-center flex flex-col items-center gap-6">
        {status === 'idle' || status === 'verifying' ? (
          <>
            <Spinner size="lg" />
            <p className="text-text-muted text-sm">Verifying your email…</p>
          </>
        ) : status === 'success' ? (
          <>
            <div className="text-6xl">✅</div>
            <h1 className="text-xl font-semibold text-text-primary">Email verified!</h1>
            <p className="text-sm text-text-muted">Redirecting to sign in…</p>
          </>
        ) : (
          <>
            <div className="text-6xl">❌</div>
            <h1 className="text-xl font-semibold text-text-primary">Verification failed</h1>
            <p className="text-sm text-red-400">{errMsg}</p>

            <p className="text-sm text-text-muted">
              Please request a new verification link after signing in.
            </p>

            <Link
              to="/login"
              className="text-sm text-text-muted hover:text-cosmic-blue transition-colors"
            >
              Back to Sign in
            </Link>
          </>
        )}
      </div>
    </div>
  )
}
