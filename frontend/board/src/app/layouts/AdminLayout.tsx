import { cn } from '@/shared/lib/utils'
import { AdminSidebar } from '@/widgets/sidebar/AdminSidebar'
import { type ReactNode, useEffect, useRef, useState } from 'react'

interface AdminLayoutProps {
  children: ReactNode
}

// Breakpoints (must match Tailwind config - md=768, lg=1024)
const TABLET_BP = 768
const DESKTOP_BP = 1024

function useAdminSidebarState() {
  const [mobileOpen, setMobileOpen] = useState(false)
  // On tablet the sidebar is always collapsed (icons only); on desktop it starts expanded.
  // We track user's manual toggle as an override on desktop.
  const [desktopCollapsed, setDesktopCollapsed] = useState(false)

  const width = useWindowWidth()

  const isMobile = width < TABLET_BP
  const isTablet = width >= TABLET_BP && width < DESKTOP_BP
  const isDesktop = width >= DESKTOP_BP

  // Collapsed state to pass to sidebar:
  // - mobile: always collapsed (sidebar shown as overlay, full-width)
  // - tablet: always collapsed (icons only, no toggle shown)
  // - desktop: user controls via desktopCollapsed
  const collapsed = isDesktop ? desktopCollapsed : true

  const handleToggle = () => {
    if (isDesktop) setDesktopCollapsed((v) => !v)
    // On mobile the toggle button lives in the topbar (opens/closes overlay)
  }

  return { isMobile, isTablet, isDesktop, mobileOpen, setMobileOpen, collapsed, handleToggle }
}

function useWindowWidth() {
  const [width, setWidth] = useState(() =>
    typeof window !== 'undefined' ? window.innerWidth : DESKTOP_BP,
  )
  const rafRef = useRef<number | null>(null)

  useEffect(() => {
    const handler = () => {
      if (rafRef.current !== null) cancelAnimationFrame(rafRef.current)
      rafRef.current = requestAnimationFrame(() => setWidth(window.innerWidth))
    }
    window.addEventListener('resize', handler)
    return () => {
      window.removeEventListener('resize', handler)
      if (rafRef.current !== null) cancelAnimationFrame(rafRef.current)
    }
  }, [])

  return width
}

export function AdminLayout({ children }: AdminLayoutProps) {
  const { isMobile, isTablet, mobileOpen, setMobileOpen, collapsed, handleToggle } =
    useAdminSidebarState()

  // Show toggle button inside sidebar header only on desktop.
  // On tablet the sidebar is always icons-only (no toggle).
  // On mobile the sidebar is an overlay controlled by topbar hamburger.

  return (
    <div className="min-h-screen bg-space-dark flex">
      {/* Mobile overlay backdrop */}
      {isMobile && mobileOpen && (
        <div
          className="fixed inset-0 z-20 bg-black/50"
          onClick={() => setMobileOpen(false)}
          aria-hidden="true"
        />
      )}

      {/* Sidebar */}
      {isMobile ? (
        // Mobile: fixed overlay, slide in/out
        <div
          className={cn(
            'fixed left-0 top-0 z-30 h-full transition-transform duration-200',
            mobileOpen ? 'translate-x-0' : '-translate-x-full',
          )}
        >
          <AdminSidebar collapsed={false} onToggle={() => setMobileOpen(false)} />
        </div>
      ) : (
        // Tablet / Desktop: part of the flex layout
        <div className="shrink-0 h-screen sticky top-0">
          <AdminSidebar collapsed={collapsed} onToggle={isTablet ? () => {} : handleToggle} />
        </div>
      )}

      {/* Main content */}
      <div className="flex-1 flex flex-col min-w-0 min-h-screen">
        {/* Mobile topbar with hamburger */}
        {isMobile && (
          <div className="flex items-center gap-3 h-14 px-4 border-b border-space-border bg-space-dark/80 backdrop-blur-md sticky top-0 z-10">
            <button
              onClick={() => setMobileOpen(true)}
              aria-label="Open admin menu"
              className="p-1.5 rounded text-text-secondary hover:text-text-primary hover:bg-space-card focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-cosmic-blue"
            >
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M4 6h16M4 12h16M4 18h16"
                />
              </svg>
            </button>
            <span
              className="text-sm font-semibold text-cosmic-purple"
              style={{ fontFamily: 'var(--font-display)' }}
            >
              Admin Panel
            </span>
          </div>
        )}

        <main className="flex-1 p-4 sm:p-6 overflow-auto">{children}</main>
      </div>
    </div>
  )
}
