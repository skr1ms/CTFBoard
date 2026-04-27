import { api, isApiError } from '@/shared/api/client'
import type { components } from '@/shared/api/schema.d'
import { EmptyState } from '@/shared/ui/empty-state'
import { Modal } from '@/shared/ui/modal'
import { Skeleton } from '@/shared/ui/skeleton'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { toast } from 'sonner'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------
type Notification = components['schemas']['NotificationResponse']
type CreateBody = components['schemas']['CreateNotificationRequest']
type NotifType = CreateBody['type']

// ---------------------------------------------------------------------------
// Hooks
// ---------------------------------------------------------------------------
function useNotifications() {
  return useQuery({
    queryKey: ['admin', 'notifications'],
    queryFn: async () => {
      const { data, error } = await api.GET('/notifications', {
        params: { query: { per_page: 100 } },
      })
      if (error) throw error
      return (data ?? []) as Notification[]
    },
    staleTime: 0,
  })
}

function useCreateNotification() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (body: CreateBody) => {
      const { error } = await api.POST('/admin/notifications', { body })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'notifications'] })
      toast.success('Notification created')
    },
    onError: (err) => toast.error(isApiError(err) ? err.message : 'Create failed'),
  })
}

function useCreatePersonalNotification() {
  return useMutation({
    mutationFn: async ({
      userID,
      body,
    }: {
      userID: string
      body: components['schemas']['CreateUserNotificationRequest']
    }) => {
      const { error } = await api.POST('/admin/notifications/user/{userID}', {
        params: { path: { userID } },
        body,
      })
      if (error) throw error
    },
    onSuccess: () => toast.success('Personal notification sent'),
    onError: (err) => toast.error(isApiError(err) ? err.message : 'Send failed'),
  })
}

function useUpdateNotification() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async ({
      id,
      body,
    }: {
      id: string
      body: { title: string; content: string; type?: NotifType; is_pinned?: boolean }
    }) => {
      const { error } = await api.PUT('/admin/notifications/{ID}', {
        params: { path: { ID: id } },
        body,
      })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'notifications'] })
      toast.success('Notification updated')
    },
    onError: (err) => toast.error(isApiError(err) ? err.message : 'Update failed'),
  })
}

function useDeleteNotification() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      const { error } = await api.DELETE('/admin/notifications/{ID}', {
        params: { path: { ID: id } },
      })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'notifications'] })
      toast.success('Deleted')
    },
    onError: (err) => toast.error(isApiError(err) ? err.message : 'Delete failed'),
  })
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
const TYPE_COLORS: Record<NotifType, string> = {
  info: 'bg-cosmic-blue/20 text-cosmic-blue',
  warning: 'bg-nova-yellow/20 text-nova-yellow',
  success: 'bg-nebula-green/20 text-nebula-green',
  error: 'bg-nova-red/20 text-nova-red',
}

// ---------------------------------------------------------------------------
// Global create form
// ---------------------------------------------------------------------------
function CreateNotificationForm() {
  const { mutateAsync: create, isPending } = useCreateNotification()
  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')
  const [type, setType] = useState<NotifType>('info')
  const [isPinned, setIsPinned] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!title.trim() || !content.trim()) return
    try {
      await create({ title: title.trim(), content: content.trim(), type, is_pinned: isPinned })
      setTitle('')
      setContent('')
      setType('info')
      setIsPinned(false)
    } catch {
      // error already toasted
    }
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="flex flex-col gap-3 p-4 rounded-[var(--radius-lg)] border border-space-border bg-space-card"
    >
      <h2 className="text-sm font-semibold text-text-primary">New Global Notification</h2>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-text-muted">Title *</label>
          <input
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="Notification title"
            required
            className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
          />
        </div>

        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-text-muted">Type</label>
          <select
            value={type}
            onChange={(e) => setType(e.target.value as NotifType)}
            className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-2 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
          >
            <option value="info">info</option>
            <option value="warning">warning</option>
            <option value="success">success</option>
            <option value="error">error</option>
          </select>
        </div>
      </div>

      <div className="flex flex-col gap-1">
        <label className="text-xs font-medium text-text-muted">Content *</label>
        <textarea
          value={content}
          onChange={(e) => setContent(e.target.value)}
          placeholder="Notification message…"
          required
          rows={3}
          className="rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40 resize-y"
        />
      </div>

      <div className="flex items-center justify-between">
        <label className="flex items-center gap-2 cursor-pointer select-none">
          <input
            type="checkbox"
            checked={isPinned}
            onChange={(e) => setIsPinned(e.target.checked)}
            className="w-4 h-4 rounded accent-cosmic-blue"
          />
          <span className="text-sm text-text-secondary">Pin notification</span>
        </label>
        <button
          type="submit"
          disabled={isPending || !title.trim() || !content.trim()}
          className="px-4 py-1.5 text-sm rounded-[var(--radius-md)] bg-cosmic-blue hover:bg-cosmic-blue/90 text-white transition-colors disabled:opacity-50"
        >
          {isPending ? 'Sending…' : 'Send notification'}
        </button>
      </div>
    </form>
  )
}

// ---------------------------------------------------------------------------
// Personal notification form
// ---------------------------------------------------------------------------
function PersonalNotificationForm() {
  const { mutateAsync: send, isPending } = useCreatePersonalNotification()
  const [userID, setUserID] = useState('')
  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')
  const [type, setType] = useState<NotifType>('info')
  const [open, setOpen] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!userID.trim() || !title.trim() || !content.trim()) return
    try {
      await send({
        userID: userID.trim(),
        body: { title: title.trim(), content: content.trim(), type },
      })
      setUserID('')
      setTitle('')
      setContent('')
      setType('info')
      setOpen(false)
    } catch {
      // error already toasted
    }
  }

  if (!open) {
    return (
      <button
        onClick={() => setOpen(true)}
        className="self-start text-xs text-text-muted hover:text-text-primary border border-space-border rounded-[var(--radius-md)] px-3 py-1.5 transition-colors"
      >
        + Send personal notification
      </button>
    )
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="flex flex-col gap-3 p-4 rounded-[var(--radius-lg)] border border-space-border bg-space-card"
    >
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold text-text-primary">Personal Notification</h2>
        <button
          type="button"
          onClick={() => setOpen(false)}
          className="text-xs text-text-muted hover:text-text-primary"
        >
          Cancel
        </button>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-text-muted">User ID *</label>
          <input
            type="text"
            value={userID}
            onChange={(e) => setUserID(e.target.value)}
            placeholder="UUID"
            required
            className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm font-mono text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
          />
        </div>
        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-text-muted">Title *</label>
          <input
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            required
            className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
          />
        </div>
        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-text-muted">Type</label>
          <select
            value={type}
            onChange={(e) => setType(e.target.value as NotifType)}
            className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-2 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
          >
            <option value="info">info</option>
            <option value="warning">warning</option>
            <option value="success">success</option>
            <option value="error">error</option>
          </select>
        </div>
      </div>

      <div className="flex flex-col gap-1">
        <label className="text-xs font-medium text-text-muted">Content *</label>
        <textarea
          value={content}
          onChange={(e) => setContent(e.target.value)}
          rows={2}
          required
          className="rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40 resize-y"
        />
      </div>

      <div className="flex justify-end">
        <button
          type="submit"
          disabled={isPending || !userID.trim() || !title.trim() || !content.trim()}
          className="px-4 py-1.5 text-sm rounded-[var(--radius-md)] bg-cosmic-blue hover:bg-cosmic-blue/90 text-white transition-colors disabled:opacity-50"
        >
          {isPending ? 'Sending…' : 'Send'}
        </button>
      </div>
    </form>
  )
}

// ---------------------------------------------------------------------------
// Edit modal
// ---------------------------------------------------------------------------
const NOTIF_TYPES: NotifType[] = ['info', 'warning', 'success', 'error']

const inputCls =
  'h-9 w-full rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40'
const textareaCls =
  'w-full rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40 resize-y'

function EditNotificationModal({ notif, onClose }: { notif: Notification; onClose: () => void }) {
  const { mutateAsync: update, isPending } = useUpdateNotification()
  const [title, setTitle] = useState(notif.title ?? '')
  const [content, setContent] = useState(notif.content ?? '')
  const [type, setType] = useState<NotifType>((notif.type as NotifType) ?? 'info')
  const [isPinned, setIsPinned] = useState(notif.is_pinned ?? false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    await update({ id: notif.id ?? '', body: { title, content, type, is_pinned: isPinned } })
    onClose()
  }

  return (
    <Modal open onClose={onClose} title="Edit Notification" maxWidth="max-w-lg">
      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-text-muted uppercase tracking-wide">
            Title
          </label>
          <input
            className={inputCls}
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            required
          />
        </div>
        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-text-muted uppercase tracking-wide">
            Content
          </label>
          <textarea
            className={textareaCls}
            rows={3}
            value={content}
            onChange={(e) => setContent(e.target.value)}
            required
          />
        </div>
        <div className="flex gap-4 flex-wrap items-center">
          <div className="flex flex-col gap-1 flex-1 min-w-[140px]">
            <label className="text-xs font-medium text-text-muted uppercase tracking-wide">
              Type
            </label>
            <select
              className={inputCls}
              value={type}
              onChange={(e) => setType(e.target.value as NotifType)}
            >
              {NOTIF_TYPES.map((t) => (
                <option key={t} value={t}>
                  {t}
                </option>
              ))}
            </select>
          </div>
          <label className="flex items-center gap-2 cursor-pointer mt-4">
            <input
              type="checkbox"
              checked={isPinned}
              onChange={(e) => setIsPinned(e.target.checked)}
              className="rounded border-space-border"
            />
            <span className="text-sm text-text-secondary">Pinned</span>
          </label>
        </div>
        <div className="flex justify-end gap-2 pt-1">
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-2 text-sm text-text-muted hover:text-text-primary transition-colors"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={isPending}
            className="px-4 py-2 text-sm rounded-[var(--radius-md)] bg-cosmic-blue text-white hover:bg-cosmic-blue/90 disabled:opacity-50 transition-colors"
          >
            {isPending ? 'Saving…' : 'Save'}
          </button>
        </div>
      </form>
    </Modal>
  )
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------
export function AdminNotificationsPage() {
  const { data: notifications = [], isLoading } = useNotifications()
  const { mutate: deleteNotif, isPending: deleting } = useDeleteNotification()
  const [editingNotif, setEditingNotif] = useState<Notification | null>(null)

  const sorted = [...notifications].sort((a, b) =>
    (b.created_at ?? '').localeCompare(a.created_at ?? ''),
  )

  return (
    <div className="flex flex-col gap-5">
      <h1
        className="text-2xl font-black text-text-primary"
        style={{ fontFamily: 'var(--font-display)' }}
      >
        Notifications
      </h1>

      <CreateNotificationForm />
      <PersonalNotificationForm />

      <div className="overflow-x-auto rounded-[var(--radius-lg)] border border-space-border">
        <table className="w-full min-w-[520px]">
          <thead>
            <tr className="border-b border-space-border bg-space-dark/60">
              <th className="px-3 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
                Title
              </th>
              <th className="px-3 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide hidden sm:table-cell">
                Content
              </th>
              <th className="px-3 py-2.5 text-center text-xs font-semibold text-text-muted uppercase tracking-wide">
                Type
              </th>
              <th className="px-3 py-2.5 text-center text-xs font-semibold text-text-muted uppercase tracking-wide">
                Pinned
              </th>
              <th className="px-3 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide hidden md:table-cell">
                Created
              </th>
              <th className="px-3 py-2.5 text-right text-xs font-semibold text-text-muted uppercase tracking-wide">
                Actions
              </th>
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              Array.from({ length: 4 }).map((_, i) => (
                <tr key={i} className="border-b border-space-border">
                  {Array.from({ length: 6 }).map((__, j) => (
                    <td key={j} className="px-3 py-2.5">
                      <Skeleton className="h-4 w-full" />
                    </td>
                  ))}
                </tr>
              ))
            ) : sorted.length === 0 ? (
              <tr>
                <td colSpan={6}>
                  <EmptyState message="No notifications yet." />
                </td>
              </tr>
            ) : (
              sorted.map((n) => (
                <tr
                  key={n.id}
                  data-testid={`admin-row-${n.id}`}
                  className="border-b border-space-border last:border-b-0 hover:bg-space-border/20 transition-colors"
                >
                  <td className="px-3 py-2.5 text-sm text-text-primary font-medium max-w-[160px]">
                    <span className="truncate block">{n.title ?? '-'}</span>
                  </td>
                  <td className="px-3 py-2.5 text-sm text-text-muted hidden sm:table-cell max-w-[220px]">
                    <span className="truncate block">{n.content ?? '-'}</span>
                  </td>
                  <td className="px-3 py-2.5 text-center">
                    <span
                      className={`inline-block px-1.5 py-0.5 text-[10px] rounded font-semibold ${TYPE_COLORS[n.type as NotifType] ?? TYPE_COLORS.info}`}
                    >
                      {n.type ?? 'info'}
                    </span>
                  </td>
                  <td className="px-3 py-2.5 text-center text-xs">
                    {n.is_pinned ? (
                      <span className="text-nova-yellow">📌</span>
                    ) : (
                      <span className="text-text-muted">-</span>
                    )}
                  </td>
                  <td className="px-3 py-2.5 text-xs text-text-muted hidden md:table-cell whitespace-nowrap">
                    {n.created_at ? new Date(n.created_at).toLocaleString() : '-'}
                  </td>
                  <td className="px-3 py-2.5 text-right">
                    <div className="flex items-center justify-end gap-2">
                      <button
                        data-testid={`admin-edit-${n.id}`}
                        onClick={() => setEditingNotif(n)}
                        className="text-xs text-text-muted hover:text-cosmic-blue transition-colors"
                      >
                        Edit
                      </button>
                      <button
                        data-testid={`admin-delete-${n.id}`}
                        onClick={() => n.id && deleteNotif(n.id)}
                        disabled={deleting}
                        className="text-xs text-nova-red hover:underline disabled:opacity-40"
                      >
                        Delete
                      </button>
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      <p className="text-xs text-text-muted">
        {notifications.length} notification{notifications.length !== 1 ? 's' : ''}
      </p>

      {editingNotif && (
        <EditNotificationModal notif={editingNotif} onClose={() => setEditingNotif(null)} />
      )}
    </div>
  )
}
