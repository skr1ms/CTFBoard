import { cn } from '@/shared/lib/utils'
import { Link, useLocation } from 'react-router'

interface SidebarLink {
  href: string
  label: string
  icon: string
}

const ADMIN_LINKS: SidebarLink[] = [
  { href: '/admin/statistics', label: 'Statistics', icon: '📊' },
  { href: '/admin/competition', label: 'Competition', icon: '🏆' },
  { href: '/admin/challenges', label: 'Challenges', icon: '🎯' },
  { href: '/admin/users', label: 'Users', icon: '👥' },
  { href: '/admin/teams', label: 'Teams', icon: '🛡️' },
  { href: '/admin/submissions', label: 'Submissions', icon: '📝' },
  { href: '/admin/unlocks', label: 'Unlocks', icon: '🔓' },
  { href: '/admin/appeals', label: 'Appeals', icon: '⚖️' },
  { href: '/admin/awards', label: 'Awards', icon: '🎖️' },
  { href: '/admin/scoreboard', label: 'Scoreboard', icon: '📋' },
  { href: '/admin/notifications', label: 'Notifications', icon: '🔔' },
  { href: '/admin/pages', label: 'Pages', icon: '📄' },
  { href: '/admin/tags', label: 'Tags', icon: '🏷️' },
  { href: '/admin/brackets', label: 'Brackets', icon: '🗂️' },
  { href: '/admin/fields', label: 'Fields', icon: '🗃️' },
  { href: '/admin/settings', label: 'Settings', icon: '⚙️' },
  { href: '/admin/storage', label: 'Storage', icon: '🗄️' },
  { href: '/admin/backup', label: 'Backup', icon: '💾' },
]

interface AdminSidebarProps {
  collapsed: boolean
  onToggle: () => void
}

export function AdminSidebar({ collapsed, onToggle }: AdminSidebarProps) {
  const { pathname } = useLocation()

  return (
    <aside
      className={cn(
        'flex flex-col h-full border-r border-space-border bg-space-card transition-all duration-200',
        collapsed ? 'w-14' : 'w-56',
      )}
    >
      {/* Header */}
      <div className="flex items-center justify-between h-14 px-3 border-b border-space-border shrink-0">
        {!collapsed && (
          <span
            className="text-sm font-semibold text-cosmic-purple truncate"
            style={{ fontFamily: 'var(--font-display)' }}
          >
            Admin Panel
          </span>
        )}
        <button
          onClick={onToggle}
          aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          className="p-1.5 rounded text-text-muted hover:text-text-primary hover:bg-space-border transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-cosmic-blue ml-auto shrink-0"
        >
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            {collapsed ? (
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M13 5l7 7-7 7M5 5l7 7-7 7"
              />
            ) : (
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M11 19l-7-7 7-7m8 14l-7-7 7-7"
              />
            )}
          </svg>
        </button>
      </div>

      {/* Nav links */}
      <nav className="flex-1 overflow-y-auto py-2">
        <ul className="flex flex-col gap-0.5 px-1.5">
          {ADMIN_LINKS.map((link) => {
            const active = pathname.startsWith(link.href)
            return (
              <li key={link.href}>
                <Link
                  to={link.href}
                  title={collapsed ? link.label : undefined}
                  className={cn(
                    'flex items-center gap-2.5 px-2 py-2 rounded text-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-cosmic-blue',
                    active
                      ? 'bg-cosmic-blue/15 text-cosmic-blue font-medium'
                      : 'text-text-secondary hover:text-text-primary hover:bg-space-border',
                  )}
                >
                  <span className="shrink-0 text-base leading-none">{link.icon}</span>
                  {!collapsed && <span className="truncate">{link.label}</span>}
                </Link>
              </li>
            )
          })}
        </ul>
      </nav>

      {/* Back to site */}
      <div className="p-2 border-t border-space-border shrink-0">
        <Link
          to="/challenges"
          title={collapsed ? 'Back to site' : undefined}
          className="flex items-center gap-2.5 px-2 py-2 rounded text-sm text-text-muted hover:text-text-primary hover:bg-space-border transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-cosmic-blue"
        >
          <svg className="w-4 h-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M10 19l-7-7m0 0l7-7m-7 7h18"
            />
          </svg>
          {!collapsed && <span>Back to site</span>}
        </Link>
      </div>
    </aside>
  )
}
