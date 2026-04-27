import { ErrorLayout } from '@/app/layouts/ErrorLayout'
import { useAuthStore } from '@/shared/stores/authStore'
import { Spinner } from '@/shared/ui/spinner'
import { type ReactNode } from 'react'
import { Link } from 'react-router'

function Forbidden() {
  return (
    <ErrorLayout>
      <div className="text-center flex flex-col items-center gap-4">
        <span
          className="text-6xl font-bold text-cosmic-purple"
          style={{ fontFamily: 'var(--font-display)' }}
        >
          403
        </span>
        <h1 className="text-xl font-semibold text-text-primary">Access Denied</h1>
        <p className="text-text-muted">You do not have permission to view this page.</p>
        <Link to="/challenges" className="text-cosmic-blue hover:underline text-sm">
          Back to Challenges
        </Link>
      </div>
    </ErrorLayout>
  )
}

export function AdminGuard({ children }: { children: ReactNode }) {
  const hydrating = useAuthStore((s) => s.hydrating)
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const isAdmin = useAuthStore((s) => s.isAdmin)

  if (hydrating) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <Spinner size="lg" />
      </div>
    )
  }

  if (!isAuthenticated || !isAdmin) return <Forbidden />
  return <>{children}</>
}
