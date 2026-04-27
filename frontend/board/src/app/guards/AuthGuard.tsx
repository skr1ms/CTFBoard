import { useAuthStore } from '@/shared/stores/authStore'
import { Spinner } from '@/shared/ui/spinner'
import { type ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router'

export function AuthGuard({ children }: { children: ReactNode }) {
  const hydrating = useAuthStore((s) => s.hydrating)
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const isBanned = useAuthStore((s) => s.isBanned)
  const location = useLocation()

  if (hydrating) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <Spinner size="lg" />
      </div>
    )
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" state={{ from: location.pathname }} replace />
  }

  if (isBanned && location.pathname !== '/banned') {
    return <Navigate to="/banned" replace />
  }

  return <>{children}</>
}
