import { type ReactNode } from 'react'

interface ErrorLayoutProps {
  children: ReactNode
}

export function ErrorLayout({ children }: ErrorLayoutProps) {
  return (
    <div className="min-h-screen bg-space-dark flex flex-col items-center justify-center px-4">
      {children}
    </div>
  )
}
