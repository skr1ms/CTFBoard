import { Footer } from '@/widgets/footer'
import { Navbar } from '@/widgets/navbar'
import { type ReactNode } from 'react'

interface PublicLayoutProps {
  children: ReactNode
}

export function PublicLayout({ children }: PublicLayoutProps) {
  return (
    <div className="min-h-screen bg-space-dark flex flex-col">
      <Navbar />
      <main className="flex-1 w-full max-w-7xl mx-auto px-4 sm:px-6 py-6">{children}</main>
      <Footer />
    </div>
  )
}
