import { env } from '@/shared/config/env'
import { type ReactNode } from 'react'

interface PublicSetupLayoutProps {
  children: ReactNode
}

export function PublicSetupLayout({ children }: PublicSetupLayoutProps) {
  return (
    <div className="min-h-screen bg-space-dark flex flex-col items-center justify-start py-10 px-4">
      <header className="mb-8 flex flex-col items-center gap-2">
        <span className="text-4xl text-cosmic-blue select-none">✦</span>
        <h1
          className="text-2xl font-bold text-text-primary tracking-tight"
          style={{ fontFamily: 'var(--font-display)' }}
        >
          {env.appName}
        </h1>
        <p className="text-sm text-text-muted">Initial Setup</p>
      </header>
      <div className="w-full max-w-2xl">{children}</div>
    </div>
  )
}
