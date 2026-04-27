import {
  type NotificationResponse,
  type UserNotificationResponse,
  useGlobalNotifications,
  useMarkAsRead,
  usePersonalNotifications,
} from '@/features/notifications/useNotifications'
import { cn } from '@/shared/lib/utils'
import { EmptyState } from '@/shared/ui/empty-state'
import { Skeleton } from '@/shared/ui/skeleton'
import { useState } from 'react'

// ---------------------------------------------------------------------------
// Level styling
// ---------------------------------------------------------------------------

type NotifLevel = 'info' | 'warning' | 'error' | 'success'

const LEVEL_STYLES: Record<NotifLevel, { border: string; dot: string; bg: string }> = {
  info: { border: 'border-cosmic-blue/60', dot: 'bg-cosmic-blue', bg: 'bg-cosmic-blue/5' },
  warning: { border: 'border-solar-yellow/60', dot: 'bg-solar-yellow', bg: 'bg-solar-yellow/5' },
  error: { border: 'border-nova-red/60', dot: 'bg-nova-red', bg: 'bg-nova-red/5' },
  success: { border: 'border-nebula-green/60', dot: 'bg-nebula-green', bg: 'bg-nebula-green/5' },
}

function levelStyles(type?: string, unread = false) {
  const key = (type ?? 'info') as NotifLevel
  const s = LEVEL_STYLES[key] ?? LEVEL_STYLES.info
  return {
    border: unread ? s.border : 'border-space-border',
    dot: s.dot,
    bg: unread ? s.bg : '',
  }
}

// ---------------------------------------------------------------------------
// Global notification card (read-only)
// ---------------------------------------------------------------------------

function GlobalCard({ n }: { n: NotificationResponse }) {
  const s = levelStyles(n.type)

  return (
    <div
      className={cn(
        'flex gap-3 rounded-[var(--radius-md)] border p-4 transition-colors',
        s.border,
        s.bg,
      )}
    >
      <span className={cn('mt-1.5 h-2 w-2 flex-shrink-0 rounded-full', s.dot)} />
      <div className="flex flex-col gap-0.5 min-w-0">
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-sm font-semibold text-text-primary">
            {n.title ?? 'Notification'}
          </span>
          {n.is_pinned && (
            <span className="text-[10px] font-semibold uppercase tracking-wide text-solar-yellow bg-solar-yellow/10 px-1.5 py-0.5 rounded">
              Pinned
            </span>
          )}
        </div>
        {n.content && (
          <p className="text-sm text-text-secondary whitespace-pre-wrap break-words">{n.content}</p>
        )}
        {n.created_at && (
          <p className="text-xs text-text-muted mt-1">
            {new Date(n.created_at).toLocaleString(undefined, {
              month: 'short',
              day: 'numeric',
              hour: '2-digit',
              minute: '2-digit',
            })}
          </p>
        )}
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Personal notification card (clickable -> mark read)
// ---------------------------------------------------------------------------

function PersonalCard({ n }: { n: UserNotificationResponse }) {
  const markAsRead = useMarkAsRead()
  const unread = !n.is_read
  const s = levelStyles(n.type, unread)

  const handleClick = () => {
    if (unread && n.id) {
      markAsRead.mutate(n.id)
    }
  }

  return (
    <div
      data-testid={`notification-item-${n.id}`}
      onClick={handleClick}
      className={cn(
        'flex gap-3 rounded-[var(--radius-md)] border p-4 transition-colors',
        s.border,
        s.bg,
        unread ? 'cursor-pointer hover:brightness-110' : 'opacity-70',
      )}
    >
      <span
        className={cn('mt-1.5 h-2 w-2 flex-shrink-0 rounded-full', s.dot, !unread && 'opacity-40')}
      />
      <div className="flex flex-col gap-0.5 min-w-0 flex-1">
        <div className="flex items-center justify-between gap-2">
          <span
            className={cn(
              'text-sm font-semibold',
              unread ? 'text-text-primary' : 'text-text-muted',
            )}
          >
            {n.title ?? 'Notification'}
          </span>
          {unread && (
            <span
              className="h-2 w-2 flex-shrink-0 rounded-full bg-cosmic-blue"
              aria-label="Unread"
            />
          )}
        </div>
        {n.content && (
          <p className="text-sm text-text-secondary whitespace-pre-wrap break-words">{n.content}</p>
        )}
        {n.created_at && (
          <p className="text-xs text-text-muted mt-1">
            {new Date(n.created_at).toLocaleString(undefined, {
              month: 'short',
              day: 'numeric',
              hour: '2-digit',
              minute: '2-digit',
            })}
          </p>
        )}
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Skeleton list
// ---------------------------------------------------------------------------

function SkeletonList() {
  return (
    <div className="flex flex-col gap-3">
      {[0, 1, 2].map((i) => (
        <div
          key={i}
          className="flex gap-3 rounded-[var(--radius-md)] border border-space-border p-4"
        >
          <Skeleton circle className="w-2 h-2 mt-1.5 flex-shrink-0" />
          <div className="flex flex-col gap-2 flex-1">
            <Skeleton className="h-4 w-40" />
            <Skeleton className="h-3 w-full" />
          </div>
        </div>
      ))}
    </div>
  )
}

// ---------------------------------------------------------------------------
// NotificationsPage
// ---------------------------------------------------------------------------

type Tab = 'personal' | 'global'

export function NotificationsPage() {
  const [tab, setTab] = useState<Tab>('personal')

  const { data: personal, isLoading: personalLoading } = usePersonalNotifications()
  const { data: global, isLoading: globalLoading } = useGlobalNotifications()

  const unreadCount = personal?.filter((n) => !n.is_read).length ?? 0

  // pinned first, then by date desc
  const sortedGlobal = [...(global ?? [])].sort((a, b) => {
    if (a.is_pinned && !b.is_pinned) return -1
    if (!a.is_pinned && b.is_pinned) return 1
    return (b.created_at ?? '').localeCompare(a.created_at ?? '')
  })

  const sortedPersonal = [...(personal ?? [])].sort((a, b) =>
    (b.created_at ?? '').localeCompare(a.created_at ?? ''),
  )

  return (
    <div className="flex flex-col gap-5 max-w-2xl mx-auto px-4 py-6 sm:px-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <h1
          className="text-2xl font-black text-text-primary"
          style={{ fontFamily: 'var(--font-display)' }}
        >
          Notifications
        </h1>
        {unreadCount > 0 && (
          <span className="flex items-center justify-center h-5 min-w-5 px-1.5 rounded-full bg-cosmic-blue text-[11px] font-bold text-white">
            {unreadCount}
          </span>
        )}
      </div>

      {/* Tabs */}
      <div className="flex border-b border-space-border">
        {(['personal', 'global'] as Tab[]).map((t) => (
          <button
            key={t}
            data-testid={`notifications-tab-${t}`}
            onClick={() => setTab(t)}
            className={cn(
              'px-4 py-2.5 text-sm font-medium capitalize transition-colors border-b-2 -mb-px',
              tab === t
                ? 'border-cosmic-blue text-text-primary'
                : 'border-transparent text-text-muted hover:text-text-secondary',
            )}
          >
            {t}
            {t === 'personal' && unreadCount > 0 && (
              <span className="ml-2 inline-flex items-center justify-center h-4 min-w-4 px-1 rounded-full bg-cosmic-blue text-[10px] font-bold text-white">
                {unreadCount}
              </span>
            )}
          </button>
        ))}
      </div>

      {/* Content */}
      {tab === 'personal' && (
        <div className="flex flex-col gap-3">
          {personalLoading ? (
            <SkeletonList />
          ) : sortedPersonal.length === 0 ? (
            <EmptyState message="No personal notifications." />
          ) : (
            sortedPersonal.map((n) => <PersonalCard key={n.id} n={n} />)
          )}
        </div>
      )}

      {tab === 'global' && (
        <div className="flex flex-col gap-3">
          {globalLoading ? (
            <SkeletonList />
          ) : sortedGlobal.length === 0 ? (
            <EmptyState message="No global notifications." />
          ) : (
            sortedGlobal.map((n) => <GlobalCard key={n.id} n={n} />)
          )}
        </div>
      )}
    </div>
  )
}
