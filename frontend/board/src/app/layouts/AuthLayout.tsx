import { useWsStore } from '@/shared/stores/wsStore'
import { CompetitionBanner } from '@/widgets/competition-banner'
import { Navbar } from '@/widgets/navbar'
import { type ReactNode } from 'react'

interface AuthLayoutProps {
  children: ReactNode
}

function WsReconnectBanner() {
  const connected = useWsStore((s) => s.connected)
  const attempt = useWsStore((s) => s.reconnectAttempt)
  const usingSse = useWsStore((s) => s.usingSse)

  if (connected || usingSse || attempt === 0) return null

  return (
    <div
      role="status"
      aria-live="polite"
      className="w-full bg-yellow-500/10 border-b border-yellow-500/30 px-4 py-2 text-center text-xs text-yellow-400"
    >
      Connection lost - reconnecting…
    </div>
  )
}

export function AuthLayout({ children }: AuthLayoutProps) {
  return (
    <div className="min-h-screen bg-space-dark flex flex-col">
      <Navbar />
      <CompetitionBanner />
      <WsReconnectBanner />
      <div className="flex flex-1 w-full max-w-7xl mx-auto px-4 sm:px-6">
        <main className="flex-1 py-6">{children}</main>
      </div>
    </div>
  )
}
