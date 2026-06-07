import { type PublicConfigs, usePublicConfigs } from '@/features/auth/usePublicConfigs'
import {
  normalizeCompetitionMode,
  participantCopy,
} from '@/features/competition/participation'
import { useCompetitionStatus } from '@/features/competition/useCompetitionStatus'
import { useUnreadCount } from '@/features/notifications/useNotifications'
import { env } from '@/shared/config/env'
import { avatarSrc, cn } from '@/shared/lib/utils'
import { useAuthStore } from '@/shared/stores/authStore'
import { useUiStore } from '@/shared/stores/uiStore'
import { Avatar } from '@/shared/ui/avatar'
import { Button } from '@/shared/ui/button'
import { useEffect, useState } from 'react'
import { Link, NavLink, useLocation, useNavigate } from 'react-router'

interface NavLinkItem {
  label: string
  href: string
}

function getNavLinks(
  role: 'guest' | 'user' | 'admin',
  configs: PublicConfigs | undefined,
  mode?: string,
): NavLinkItem[] {
  const isPublic = (key: keyof PublicConfigs) => (configs?.[key] ?? 'public') === 'public'
  const links: NavLinkItem[] = []
  const labels = participantCopy(mode)

  if (role !== 'guest') {
    links.push({ label: 'Challenges', href: '/challenges' })
  }
  if (role !== 'guest' || isPublic('score_visibility')) {
    links.push({ label: 'Scoreboard', href: '/scoreboard' })
  }
  if (role !== 'guest' || isPublic('account_visibility')) {
    links.push({ label: labels.plural, href: '/teams' })
  }
  if (role === 'admin') {
    links.push({ label: 'Users', href: '/users' })
    links.push({ label: 'Activity', href: '/activity' })
  }
  if (role === 'user') {
    links.push({ label: 'Users', href: '/users' })
    links.push({ label: 'Activity', href: '/activity' })
  }

  return links
}

export function Navbar() {
  const { user, isAuthenticated, isAdmin, logout } = useAuthStore()
  const { mobileMenuOpen, toggleMobileMenu, setMobileMenuOpen } = useUiStore()
  const { data: publicConfigs } = usePublicConfigs()
  const { data: competitionStatus } = useCompetitionStatus()
  const navLinks = getNavLinks(
    isAdmin ? 'admin' : isAuthenticated ? 'user' : 'guest',
    publicConfigs,
    competitionStatus?.mode,
  )
  const [userMenuOpen, setUserMenuOpen] = useState(false)
  const navigate = useNavigate()
  const location = useLocation()
  const unreadCountRaw = useUnreadCount()
  const unreadCount = isAuthenticated ? unreadCountRaw : 0
  const mode = normalizeCompetitionMode(competitionStatus?.mode)
  const myParticipationLabel = mode === 'solo_only' ? 'My Profile' : 'My Team'
  const enrollLabel = mode === 'solo_only' ? 'Start Solo' : 'Join / Create Team'

  useEffect(() => {
    setMobileMenuOpen(false)
  }, [location.pathname, setMobileMenuOpen])

  const handleLogout = () => {
    setUserMenuOpen(false)
    void logout()
      .catch(() => undefined)
      .then(() => navigate('/login', { replace: true }))
  }

  return (
    <header className="sticky top-0 z-40 w-full border-b border-space-border bg-space-dark/80 backdrop-blur-md">
      <nav className="mx-auto flex h-14 max-w-7xl items-center justify-between px-4 sm:px-6">
        {/* Logo */}
        <Link
          to="/"
          className="flex items-center gap-2 text-lg font-bold text-cosmic-blue focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cosmic-blue rounded"
          style={{ fontFamily: 'var(--font-display)' }}
        >
          <span className="text-2xl">✦</span>
          <span className="hidden sm:inline">{env.appName}</span>
        </Link>

        {/* Desktop links */}
        <ul className="hidden md:flex items-center gap-1">
          {navLinks.map((link) => (
            <li key={link.href}>
              <NavLink
                to={link.href}
                className={({ isActive }) =>
                  cn(
                    'px-3 py-1.5 rounded text-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-cosmic-blue',
                    isActive
                      ? 'text-text-primary'
                      : 'text-text-secondary hover:text-text-primary hover:bg-space-card',
                  )
                }
              >
                {link.label}
              </NavLink>
            </li>
          ))}
          {isAdmin && (
            <li>
              <NavLink
                to="/admin"
                className={({ isActive }) =>
                  cn(
                    'px-3 py-1.5 rounded text-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-cosmic-purple',
                    isActive
                      ? 'text-text-primary'
                      : 'text-cosmic-purple hover:text-text-primary hover:bg-space-card',
                  )
                }
              >
                Admin
              </NavLink>
            </li>
          )}
        </ul>

        {/* Right side */}
        <div className="flex items-center gap-2">
          {isAuthenticated && (
            <button
              data-testid="navbar-notifications-bell"
              onClick={() => navigate('/notifications')}
              aria-label={`Notifications${unreadCount > 0 ? ` (${unreadCount} unread)` : ''}`}
              className="relative p-1.5 rounded text-text-secondary hover:text-text-primary hover:bg-space-card transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-cosmic-blue"
            >
              <svg
                className="w-5 h-5"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth={2}
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M15 17h5l-1.405-1.405A2.032 2.032 0 0 1 18 14.158V11a6 6 0 1 0-12 0v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0a3 3 0 1 1-6 0m6 0H9"
                />
              </svg>
              {unreadCount > 0 && (
                <span
                  data-testid="navbar-unread-badge"
                  className="absolute -top-0.5 -right-0.5 flex items-center justify-center h-4 min-w-4 px-1 rounded-full bg-cosmic-blue text-[10px] font-bold text-white leading-none"
                >
                  {unreadCount > 99 ? '99+' : unreadCount}
                </span>
              )}
            </button>
          )}
          {isAuthenticated ? (
            <div className="relative">
              <button
                data-testid="navbar-user-menu-trigger"
                onClick={() => setUserMenuOpen((v) => !v)}
                className="flex items-center gap-2 rounded-full focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cosmic-blue"
                aria-label="User menu"
                aria-expanded={userMenuOpen}
              >
                <Avatar src={avatarSrc(user?.avatar_url)} alt={user?.username ?? ''} size="sm" />
              </button>

              {userMenuOpen && (
                <>
                  <div
                    className="fixed inset-0 z-10"
                    onClick={() => setUserMenuOpen(false)}
                    aria-hidden="true"
                  />
                  <div className="absolute right-0 top-10 z-20 w-48 rounded-[var(--radius-md)] border border-space-border bg-space-card shadow-lg">
                    <div className="px-3 py-2 border-b border-space-border">
                      <p className="text-sm font-medium text-text-primary truncate">
                        {user?.username}
                      </p>
                      <p className="text-xs text-text-muted truncate">{user?.email}</p>
                    </div>
                    <ul className="py-1">
                      <li>
                        <Link
                          to="/settings"
                          onClick={() => setUserMenuOpen(false)}
                          className="block px-3 py-2 text-sm text-text-secondary hover:text-text-primary hover:bg-space-border transition-colors"
                        >
                          Settings
                        </Link>
                      </li>
                      {user?.team_id ? (
                        <li>
                          <Link
                            to="/team"
                            onClick={() => setUserMenuOpen(false)}
                            className="block px-3 py-2 text-sm text-text-secondary hover:text-text-primary hover:bg-space-border transition-colors"
                          >
                            {myParticipationLabel}
                          </Link>
                        </li>
                      ) : (
                        <li>
                          <Link
                            to="/team/enroll"
                            onClick={() => setUserMenuOpen(false)}
                            className="block px-3 py-2 text-sm text-text-secondary hover:text-text-primary hover:bg-space-border transition-colors"
                          >
                            {enrollLabel}
                          </Link>
                        </li>
                      )}
                      <li>
                        <button
                          data-testid="navbar-logout"
                          onClick={handleLogout}
                          className="w-full text-left px-3 py-2 text-sm text-nova-red hover:bg-space-border transition-colors"
                        >
                          Sign out
                        </button>
                      </li>
                    </ul>
                  </div>
                </>
              )}
            </div>
          ) : (
            <div className="hidden md:flex items-center gap-2">
              <Button variant="ghost" size="sm" onClick={() => navigate('/login')}>
                Sign in
              </Button>
              <Button variant="primary" size="sm" onClick={() => navigate('/register')}>
                Register
              </Button>
            </div>
          )}

          {/* Mobile menu button */}
          <button
            className="md:hidden p-2 rounded text-text-secondary hover:text-text-primary hover:bg-space-card focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-cosmic-blue"
            onClick={toggleMobileMenu}
            aria-label="Toggle menu"
            aria-expanded={mobileMenuOpen}
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              {mobileMenuOpen ? (
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M6 18L18 6M6 6l12 12"
                />
              ) : (
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M4 6h16M4 12h16M4 18h16"
                />
              )}
            </svg>
          </button>
        </div>
      </nav>

      {/* Mobile drawer */}
      <div
        className={cn(
          'md:hidden border-t border-space-border bg-space-dark overflow-hidden transition-all duration-200',
          mobileMenuOpen ? 'max-h-96' : 'max-h-0',
        )}
      >
        <ul className="px-4 py-2 flex flex-col gap-1">
          {navLinks.map((link) => (
            <li key={link.href}>
              <NavLink
                to={link.href}
                onClick={() => setMobileMenuOpen(false)}
                className={({ isActive }) =>
                  cn(
                    'block px-3 py-2 rounded text-sm transition-colors',
                    isActive
                      ? 'text-text-primary'
                      : 'text-text-secondary hover:text-text-primary hover:bg-space-card',
                  )
                }
              >
                {link.label}
              </NavLink>
            </li>
          ))}
          {isAdmin && (
            <li>
              <NavLink
                to="/admin"
                onClick={() => setMobileMenuOpen(false)}
                className="block px-3 py-2 rounded text-sm text-cosmic-purple hover:bg-space-card transition-colors"
              >
                Admin
              </NavLink>
            </li>
          )}
          {!isAuthenticated && (
            <>
              <li className="pt-1 border-t border-space-border">
                <Link
                  to="/login"
                  onClick={() => setMobileMenuOpen(false)}
                  className="block px-3 py-2 rounded text-sm text-text-secondary hover:text-text-primary hover:bg-space-card transition-colors"
                >
                  Sign in
                </Link>
              </li>
              <li>
                <Link
                  to="/register"
                  onClick={() => setMobileMenuOpen(false)}
                  className="block px-3 py-2 rounded text-sm text-cosmic-blue hover:bg-space-card transition-colors"
                >
                  Register
                </Link>
              </li>
            </>
          )}
        </ul>
      </div>
    </header>
  )
}
