import { useAuthStore } from '@/shared/stores/authStore'
import { Spinner } from '@/shared/ui/spinner'
import { type ReactNode } from 'react'
import { Navigate } from 'react-router'

export function GuestGuard({ children }: { children: ReactNode }) {
  const hydrating = useAuthStore((s) => s.hydrating)
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)

  if (hydrating) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <Spinner size="lg" />
      </div>
    )
  }

  if (isAuthenticated) return <Navigate to="/challenges" replace />
  return <>{children}</>
}
