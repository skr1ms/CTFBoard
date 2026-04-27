import { useAuthStore } from '@/shared/stores/authStore'
import { Spinner } from '@/shared/ui/spinner'
import { type ReactNode } from 'react'
import { Navigate } from 'react-router'

export function TeamGuard({ children }: { children: ReactNode }) {
  const hydrating = useAuthStore((s) => s.hydrating)
  const user = useAuthStore((s) => s.user)

  if (hydrating) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <Spinner size="lg" />
      </div>
    )
  }

  if (!user?.team_id) {
    return <Navigate to="/team/enroll" replace />
  }

  return <>{children}</>
}
