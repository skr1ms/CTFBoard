import { usePublicConfigs } from '@/features/auth/usePublicConfigs'
import { Spinner } from '@/shared/ui/spinner'
import { type ReactNode } from 'react'
import { Navigate, Outlet } from 'react-router'

/**
 * Redirects all unauthenticated traffic to /setup while setup_complete is false.
 * Once setup is complete this guard becomes a transparent passthrough.
 * Also used inversely on the /setup route to redirect away when already set up.
 */
export function SetupGuard() {
  const { data, isLoading } = usePublicConfigs()

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-space-dark">
        <Spinner size="lg" />
      </div>
    )
  }

  const isComplete = data?.setup_complete === true

  if (!isComplete) {
    return <Navigate to="/setup" replace />
  }

  return <Outlet />
}

/** Companion guard placed on the /setup route itself - redirects to /admin if already done. */
export function SetupCompleteGuard({ children }: { children: ReactNode }) {
  const { data, isLoading } = usePublicConfigs()

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-space-dark">
        <Spinner size="lg" />
      </div>
    )
  }

  const isComplete = data?.setup_complete === true

  if (isComplete) {
    return <Navigate to="/admin" replace />
  }

  return <>{children}</>
}
