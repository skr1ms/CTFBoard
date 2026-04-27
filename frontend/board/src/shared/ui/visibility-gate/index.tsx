import { usePublicConfigs, type Visibility } from '@/features/auth/usePublicConfigs'
import { useAuthStore } from '@/shared/stores/authStore'
import { Link } from 'react-router'

interface VisibilityGateProps {
  configKey: 'challenge_visibility' | 'score_visibility' | 'account_visibility'
  children: React.ReactNode
}

function SignInCard({ section }: { section: string }) {
  return (
    <div className="flex flex-col items-center justify-center gap-4 py-20 text-center">
      <div className="w-14 h-14 rounded-full bg-space-card border border-space-border flex items-center justify-center">
        <svg
          className="w-7 h-7 text-text-muted"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={1.5}
            d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"
          />
        </svg>
      </div>
      <div className="flex flex-col gap-1">
        <p className="text-base font-semibold text-text-primary">Sign in to view {section}</p>
        <p className="text-sm text-text-muted">
          This section is only available to registered participants.
        </p>
      </div>
      <div className="flex gap-3">
        <Link
          to="/login"
          className="px-5 py-2 text-sm rounded-[var(--radius-md)] bg-cosmic-blue hover:bg-cosmic-blue/90 text-white transition-colors"
        >
          Sign in
        </Link>
        <Link
          to="/register"
          className="px-5 py-2 text-sm rounded-[var(--radius-md)] border border-space-border text-text-secondary hover:text-text-primary hover:border-cosmic-blue/40 transition-colors"
        >
          Register
        </Link>
      </div>
    </div>
  )
}

function RestrictedCard() {
  return (
    <div className="flex flex-col items-center justify-center gap-3 py-20 text-center">
      <div className="w-14 h-14 rounded-full bg-space-card border border-space-border flex items-center justify-center">
        <svg
          className="w-7 h-7 text-nova-red/60"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={1.5}
            d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
          />
        </svg>
      </div>
      <p className="text-base font-semibold text-text-primary">Not available</p>
      <p className="text-sm text-text-muted">This section has been restricted by the organizers.</p>
    </div>
  )
}

const SECTION_LABELS: Record<string, string> = {
  challenge_visibility: 'challenges',
  score_visibility: 'scoreboard',
  account_visibility: 'user & team profiles',
}

export function VisibilityGate({ configKey, children }: VisibilityGateProps) {
  const { data: configs, isLoading } = usePublicConfigs()
  const { isAuthenticated, user } = useAuthStore()

  if (isLoading) return null

  const visibility: Visibility = (configs?.[configKey] as Visibility) ?? 'public'
  const isAdmin = user?.role === 'admin'

  if (isAdmin) return <>{children}</>

  if (visibility === 'public') return <>{children}</>

  if (visibility === 'private') {
    if (!isAuthenticated) {
      return <SignInCard section={SECTION_LABELS[configKey] ?? configKey} />
    }
    return <>{children}</>
  }

  // 'hidden' | 'admins' - non-admins see restricted card
  return <RestrictedCard />
}
