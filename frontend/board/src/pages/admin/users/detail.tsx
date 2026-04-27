import { api, isApiError } from '@/shared/api/client'
import type { components } from '@/shared/api/schema.d'
import { avatarSrc } from '@/shared/lib/utils'
import { Avatar } from '@/shared/ui/avatar'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { EmptyState } from '@/shared/ui/empty-state'
import { PasswordInput } from '@/shared/ui/password-input'
import { Skeleton } from '@/shared/ui/skeleton'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { useLocation, useNavigate, useParams } from 'react-router'
import { toast } from 'sonner'

type AdminUser = components['schemas']['AdminUserResponse']
type TrackingEntry = components['schemas']['TrackingEntry']

// ---------------------------------------------------------------------------
// Hooks
// ---------------------------------------------------------------------------
function useAdminUser(id: string, initialUser: AdminUser | null) {
  return useQuery({
    queryKey: ['admin', 'user', id],
    queryFn: async () => {
      // No dedicated GET /admin/users/{ID} endpoint - search by username is unreliable
      // from a URL without username context. Scan pages until found.
      let page = 1
      while (true) {
        const { data, error } = await api.GET('/admin/users', {
          params: { query: { page, per_page: 50 } },
        })
        if (error) throw error
        const rows = data?.data ?? []
        const found = rows.find((u) => u.id === id)
        if (found) return found
        const totalPages = data?.meta?.total_pages ?? 1
        if (page >= totalPages || rows.length === 0) return null
        page++
      }
    },
    // Use router state as placeholder to avoid loading flash when navigating from list
    placeholderData: initialUser ?? null,
    staleTime: 0,
  })
}

function useAdminUserTracking(id: string, page: number) {
  return useQuery({
    queryKey: ['admin', 'user', id, 'tracking', page],
    queryFn: async () => {
      const { data, error } = await api.GET('/admin/users/{ID}/tracking', {
        params: { path: { ID: id }, query: { page, per_page: 20 } },
      })
      if (error) throw error
      return data ?? { data: [], meta: undefined }
    },
    staleTime: 0,
  })
}

function useAdminUserSubmissions(id: string, page: number) {
  return useQuery({
    queryKey: ['admin', 'user', id, 'submissions', page],
    queryFn: async () => {
      const { data, error } = await api.GET('/admin/submissions/user/{userID}', {
        params: { path: { userID: id }, query: { page, per_page: 20 } },
      })
      if (error) throw error
      return data ?? { data: [], meta: undefined }
    },
    staleTime: 0,
  })
}

function useAdminUserMissing(id: string) {
  return useQuery({
    queryKey: ['admin', 'user', id, 'missing'],
    queryFn: async () => {
      const { data, error } = await api.GET('/admin/users/{ID}/missing-challenges', {
        params: { path: { ID: id } },
      })
      if (error) throw error
      return data ?? []
    },
    staleTime: 0,
  })
}

function useUpdateUser(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (body: {
      username?: string
      email?: string
      role?: string
      is_verified?: boolean
      password?: string
    }) => {
      const { data, error } = await api.PATCH('/admin/users/{ID}', {
        params: { path: { ID: id } },
        body,
      })
      if (error) throw error
      return data
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'user', id] })
      void qc.invalidateQueries({ queryKey: ['admin', 'users'] })
    },
  })
}

function useBanUser(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (reason: string) => {
      const { error } = await api.POST('/admin/users/{ID}/ban', {
        params: { path: { ID: id } },
        body: { reason },
      })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'user', id] })
      void qc.invalidateQueries({ queryKey: ['admin', 'users'] })
    },
  })
}

function useUnbanUser(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async () => {
      const { error } = await api.DELETE('/admin/users/{ID}/ban', {
        params: { path: { ID: id } },
      })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'user', id] })
      void qc.invalidateQueries({ queryKey: ['admin', 'users'] })
    },
  })
}

function useDeleteUserAvatar(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async () => {
      const { error } = await api.DELETE('/admin/users/{ID}/avatar', {
        params: { path: { ID: id } },
      })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'user', id] })
      void qc.invalidateQueries({ queryKey: ['admin', 'users'] })
      toast.success('Avatar removed')
    },
    onError: () => toast.error('Failed to remove avatar'),
  })
}

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

function SectionCard({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="rounded-[var(--radius-lg)] border border-space-border bg-space-card p-5 flex flex-col gap-4">
      <h2 className="text-sm font-semibold text-text-muted uppercase tracking-wide">{title}</h2>
      {children}
    </div>
  )
}

// Edit form
function EditForm({ user, userId }: { user: AdminUser; userId: string }) {
  const { mutateAsync: updateUser, isPending } = useUpdateUser(userId)
  const [username, setUsername] = useState(user.username ?? '')
  const [email, setEmail] = useState(user.email ?? '')
  const [role, setRole] = useState(user.role ?? 'user')
  const [isVerified, setIsVerified] = useState(user.is_verified ?? false)
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    const body: Parameters<typeof updateUser>[0] = {
      username,
      email,
      role,
      is_verified: isVerified,
    }
    if (password) body.password = password
    try {
      await updateUser(body)
      toast.success('User updated')
      setPassword('')
    } catch (err) {
      setError(isApiError(err) ? err.message : 'Update failed')
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-3">
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div className="flex flex-col gap-1.5">
          <label className="text-xs font-semibold text-text-muted uppercase tracking-wide">
            Username
          </label>
          <input
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <label className="text-xs font-semibold text-text-muted uppercase tracking-wide">
            Email
          </label>
          <input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <label className="text-xs font-semibold text-text-muted uppercase tracking-wide">
            Role
          </label>
          <select
            value={role}
            onChange={(e) => setRole(e.target.value)}
            className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
          >
            <option value="user">user</option>
            <option value="admin">admin</option>
          </select>
        </div>
        <PasswordInput
          label="New Password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="Leave blank to keep current"
          autoComplete="new-password"
        />
      </div>
      <label className="flex items-center gap-2 cursor-pointer w-fit">
        <input
          type="checkbox"
          checked={isVerified}
          onChange={(e) => setIsVerified(e.target.checked)}
          className="rounded border-space-border accent-cosmic-blue"
        />
        <span className="text-sm text-text-secondary">Email verified</span>
      </label>
      {error && <p className="text-xs text-nova-red bg-nova-red/10 rounded px-3 py-2">{error}</p>}
      <div>
        <Button type="submit" variant="primary" size="sm" loading={isPending}>
          Save changes
        </Button>
      </div>
    </form>
  )
}

// Ban section
function BanSection({ user, userId }: { user: AdminUser; userId: string }) {
  const { mutateAsync: banUser, isPending: banning } = useBanUser(userId)
  const { mutateAsync: unbanUser, isPending: unbanning } = useUnbanUser(userId)
  const [showBanForm, setShowBanForm] = useState(false)
  const [showUnbanConfirm, setShowUnbanConfirm] = useState(false)
  const [reason, setReason] = useState('')
  const [error, setError] = useState('')

  const handleBan = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    try {
      await banUser(reason)
      toast.success(`User banned`)
      setShowBanForm(false)
      setReason('')
    } catch (err) {
      setError(isApiError(err) ? err.message : 'Ban failed')
    }
  }

  const handleUnban = async () => {
    try {
      await unbanUser()
      toast.success('User unbanned')
      setShowUnbanConfirm(false)
    } catch (err) {
      toast.error(isApiError(err) ? err.message : 'Unban failed')
    }
  }

  if (user.is_banned) {
    return (
      <div className="flex flex-col gap-3">
        <div className="flex items-center gap-3 p-3 rounded-[var(--radius-md)] bg-nova-red/10 border border-nova-red/20">
          <Badge variant="red">Banned</Badge>
          <div className="flex-1 min-w-0">
            {user.banned_reason && (
              <p className="text-xs text-text-secondary truncate">Reason: {user.banned_reason}</p>
            )}
            {user.banned_at && (
              <p className="text-xs text-text-muted">{new Date(user.banned_at).toLocaleString()}</p>
            )}
          </div>
        </div>
        {showUnbanConfirm ? (
          <div className="flex items-center gap-2 text-sm">
            <span className="text-text-secondary">Confirm unban?</span>
            <Button size="sm" variant="primary" loading={unbanning} onClick={handleUnban}>
              Unban
            </Button>
            <Button size="sm" variant="ghost" onClick={() => setShowUnbanConfirm(false)}>
              Cancel
            </Button>
          </div>
        ) : (
          <Button size="sm" variant="ghost" onClick={() => setShowUnbanConfirm(true)}>
            Unban user
          </Button>
        )}
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-3">
      {!showBanForm ? (
        <Button size="sm" variant="danger" onClick={() => setShowBanForm(true)}>
          Ban user
        </Button>
      ) : (
        <form onSubmit={handleBan} className="flex flex-col gap-3">
          <div className="flex flex-col gap-1.5">
            <label className="text-xs font-semibold text-text-muted uppercase tracking-wide">
              Ban reason
            </label>
            <input
              type="text"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              required
              placeholder="Violation of rules…"
              className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-nova-red/40"
            />
          </div>
          {error && <p className="text-xs text-nova-red">{error}</p>}
          <div className="flex gap-2">
            <Button type="submit" variant="danger" size="sm" loading={banning}>
              Confirm ban
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => {
                setShowBanForm(false)
                setReason('')
              }}
            >
              Cancel
            </Button>
          </div>
        </form>
      )}
    </div>
  )
}

// Avatar section
function AvatarSection({ user, userId }: { user: AdminUser; userId: string }) {
  const qc = useQueryClient()
  const { mutateAsync: deleteAvatar, isPending: deleting } = useDeleteUserAvatar(userId)
  const [preview, setPreview] = useState<string | null>(null)
  const [uploading, setUploading] = useState(false)

  useEffect(() => {
    return () => {
      if (preview) URL.revokeObjectURL(preview)
    }
  }, [preview])

  const handleFile = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    setPreview(URL.createObjectURL(file))
    setUploading(true)
    try {
      const form = new FormData()
      form.append('file', file)
      const { error: uploadError } = await api.PUT('/admin/users/{ID}/avatar', {
        params: { path: { ID: userId } },
        body: form as never,
      })
      if (uploadError) {
        toast.error(isApiError(uploadError) ? uploadError.message : 'Upload failed')
        setPreview(null)
        return
      }
      void qc.invalidateQueries({ queryKey: ['admin', 'user', userId] })
      void qc.invalidateQueries({ queryKey: ['admin', 'users'] })
      toast.success('Avatar updated')
    } catch {
      toast.error('Upload failed')
      setPreview(null)
    } finally {
      setUploading(false)
      e.target.value = ''
    }
  }

  const currentUrl = preview ?? avatarSrc(user.avatar_url) ?? null

  return (
    <div className="flex items-center gap-4">
      <Avatar src={currentUrl} alt={user.username ?? ''} size="lg" />
      <div className="flex flex-col gap-2">
        <label
          className={`cursor-pointer text-xs font-medium px-3 py-1.5 rounded-[var(--radius-md)] border border-space-border text-text-secondary hover:text-text-primary hover:bg-space-border/50 transition-colors ${uploading ? 'opacity-50 cursor-not-allowed' : ''}`}
        >
          {uploading ? 'Uploading…' : 'Upload avatar'}
          <input
            type="file"
            accept="image/*"
            className="sr-only"
            onChange={handleFile}
            disabled={uploading}
          />
        </label>
        {currentUrl && (
          <button
            onClick={() => deleteAvatar()}
            disabled={deleting}
            className="text-xs text-nova-red hover:underline disabled:opacity-40 text-left"
          >
            Remove avatar
          </button>
        )}
      </div>
    </div>
  )
}

// Tracking tab
function TrackingTab({ userId }: { userId: string }) {
  const [page, setPage] = useState(1)
  const { data, isLoading } = useAdminUserTracking(userId, page)
  const entries: TrackingEntry[] = data?.data ?? []
  const meta = data?.meta

  return (
    <div className="flex flex-col gap-3">
      <div className="overflow-x-auto rounded-[var(--radius-lg)] border border-space-border">
        <table className="w-full min-w-[500px] text-sm">
          <thead>
            <tr className="border-b border-space-border bg-space-dark/60">
              <th className="px-4 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
                IP Address
              </th>
              <th className="px-4 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide hidden sm:table-cell">
                User Agent
              </th>
              <th className="px-4 py-2.5 text-right text-xs font-semibold text-text-muted uppercase tracking-wide whitespace-nowrap">
                Tracked At
              </th>
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              Array.from({ length: 5 }).map((_, i) => (
                <tr key={i} className="border-b border-space-border last:border-b-0">
                  <td className="px-4 py-2.5">
                    <Skeleton className="h-4 w-28" />
                  </td>
                  <td className="px-4 py-2.5 hidden sm:table-cell">
                    <Skeleton className="h-4 w-48" />
                  </td>
                  <td className="px-4 py-2.5 text-right">
                    <Skeleton className="h-4 w-28 ml-auto" />
                  </td>
                </tr>
              ))
            ) : entries.length === 0 ? (
              <tr>
                <td colSpan={3}>
                  <EmptyState message="No tracking data." />
                </td>
              </tr>
            ) : (
              entries.map((e) => (
                <tr
                  key={e.id}
                  className="border-b border-space-border last:border-b-0 hover:bg-space-border/20 transition-colors"
                >
                  <td className="px-4 py-2.5 font-mono text-xs text-text-primary">{e.ip ?? '-'}</td>
                  <td className="px-4 py-2.5 text-xs text-text-muted truncate max-w-[260px] hidden sm:table-cell">
                    {e.user_agent ?? '-'}
                  </td>
                  <td className="px-4 py-2.5 text-right text-xs text-text-muted whitespace-nowrap">
                    {e.tracked_at ? new Date(e.tracked_at).toLocaleString() : '-'}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
      {meta && (meta.total_pages ?? 1) > 1 && (
        <div className="flex items-center justify-between text-sm">
          <button
            onClick={() => setPage((p) => Math.max(1, p - 1))}
            disabled={page <= 1}
            className="px-3 py-1.5 rounded-[var(--radius-md)] border border-space-border text-text-secondary hover:text-text-primary hover:bg-space-border/50 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
          >
            ‹ Prev
          </button>
          <span className="text-text-muted">
            Page {page} / {meta.total_pages ?? 1}
          </span>
          <button
            onClick={() => setPage((p) => p + 1)}
            disabled={page >= (meta.total_pages ?? 1)}
            className="px-3 py-1.5 rounded-[var(--radius-md)] border border-space-border text-text-secondary hover:text-text-primary hover:bg-space-border/50 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
          >
            Next ›
          </button>
        </div>
      )}
    </div>
  )
}

// Missing challenges tab
function MissingTab({ userId }: { userId: string }) {
  const { data: challenges, isLoading } = useAdminUserMissing(userId)

  if (isLoading) {
    return (
      <div className="flex flex-col gap-2">
        {[0, 1, 2].map((i) => (
          <Skeleton key={i} className="h-10 w-full" />
        ))}
      </div>
    )
  }

  if (!challenges || challenges.length === 0) {
    return <EmptyState message="All challenges solved!" />
  }

  return (
    <div className="overflow-x-auto rounded-[var(--radius-lg)] border border-space-border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-space-border bg-space-dark/60">
            <th className="px-4 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
              Title
            </th>
            <th className="px-4 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide hidden sm:table-cell">
              Category
            </th>
            <th className="px-4 py-2.5 text-right text-xs font-semibold text-text-muted uppercase tracking-wide">
              Points
            </th>
          </tr>
        </thead>
        <tbody>
          {challenges.map((c) => (
            <tr
              key={c.id}
              className="border-b border-space-border last:border-b-0 hover:bg-space-border/20 transition-colors"
            >
              <td className="px-4 py-2.5 text-text-primary">{c.title ?? '-'}</td>
              <td className="px-4 py-2.5 text-text-secondary hidden sm:table-cell text-xs">
                {c.category ?? '-'}
              </td>
              <td className="px-4 py-2.5 text-right font-mono font-bold text-cosmic-blue">
                {c.points ?? 0}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

// Submissions tab
function SubmissionsTab({ userId }: { userId: string }) {
  const [page, setPage] = useState(1)
  const { data, isLoading } = useAdminUserSubmissions(userId, page)
  const rows = data?.data ?? []
  const meta = data?.meta

  if (isLoading) {
    return (
      <div className="flex flex-col gap-2">
        {[0, 1, 2].map((i) => (
          <Skeleton key={i} className="h-10 w-full" />
        ))}
      </div>
    )
  }

  if (rows.length === 0) {
    return <EmptyState message="No submissions found." />
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="overflow-x-auto rounded-[var(--radius-lg)] border border-space-border">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-space-border bg-space-dark/60">
              <th className="px-4 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
                Challenge
              </th>
              <th className="px-4 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide hidden sm:table-cell">
                Flag
              </th>
              <th className="px-4 py-2.5 text-center text-xs font-semibold text-text-muted uppercase tracking-wide">
                Result
              </th>
              <th className="px-4 py-2.5 text-right text-xs font-semibold text-text-muted uppercase tracking-wide hidden md:table-cell">
                Time
              </th>
            </tr>
          </thead>
          <tbody>
            {rows.map((s) => (
              <tr
                key={s.id}
                className="border-b border-space-border last:border-b-0 hover:bg-space-border/20 transition-colors"
              >
                <td className="px-4 py-2.5 text-text-primary">
                  {s.challenge_title ?? s.challenge_id ?? '-'}
                </td>
                <td className="px-4 py-2.5 font-mono text-xs text-text-muted hidden sm:table-cell truncate max-w-[180px]">
                  {s.submitted_flag ?? '-'}
                </td>
                <td className="px-4 py-2.5 text-center">
                  <span
                    className={`px-1.5 py-0.5 rounded text-xs font-medium ${s.is_correct ? 'bg-nebula-green/10 text-nebula-green' : 'bg-nova-red/10 text-nova-red'}`}
                  >
                    {s.is_correct ? 'Correct' : 'Wrong'}
                  </span>
                </td>
                <td className="px-4 py-2.5 text-right text-text-muted text-xs hidden md:table-cell">
                  {s.created_at ? new Date(s.created_at).toLocaleString() : '-'}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {meta && (meta.total_pages ?? 1) > 1 && (
        <div className="flex items-center justify-between text-sm text-text-muted">
          <button
            disabled={page <= 1}
            onClick={() => setPage((p) => p - 1)}
            className="px-3 py-1 rounded border border-space-border hover:bg-space-border/30 disabled:opacity-40 transition-colors"
          >
            Prev
          </button>
          <span>
            Page {page} / {meta.total_pages ?? 1}
          </span>
          <button
            disabled={page >= (meta.total_pages ?? 1)}
            onClick={() => setPage((p) => p + 1)}
            className="px-3 py-1 rounded border border-space-border hover:bg-space-border/30 disabled:opacity-40 transition-colors"
          >
            Next
          </button>
        </div>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------
type Tab = 'edit' | 'tracking' | 'missing' | 'submissions'

const TABS: Array<{ id: Tab; label: string }> = [
  { id: 'edit', label: 'Profile' },
  { id: 'tracking', label: 'Tracking' },
  { id: 'missing', label: 'Missing' },
  { id: 'submissions', label: 'Submissions' },
]

export function AdminUserDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const stateUser = (location.state as { user?: AdminUser } | null)?.user ?? null
  const [tab, setTab] = useState<Tab>('edit')

  const { data: user, isLoading } = useAdminUser(id ?? '', stateUser)

  if (!id) return null

  return (
    <div className="flex flex-col gap-5 max-w-3xl">
      {/* Back */}
      <button
        onClick={() => navigate('/admin/users')}
        className="flex items-center gap-1.5 text-sm text-text-muted hover:text-text-primary transition-colors w-fit"
      >
        <svg className="w-4 h-4" viewBox="0 0 16 16" fill="currentColor">
          <path
            fillRule="evenodd"
            d="M9.78 4.22a.75.75 0 0 1 0 1.06L7.06 8l2.72 2.72a.749.749 0 0 1-.326 1.275.749.749 0 0 1-.734-.215L5.97 8.53a.75.75 0 0 1 0-1.06l2.75-2.75a.75.75 0 0 1 1.06 0Z"
            clipRule="evenodd"
          />
        </svg>
        Back to users
      </button>

      {/* Header */}
      {isLoading ? (
        <div className="flex items-center gap-4">
          <Skeleton circle className="w-14 h-14" />
          <div className="flex flex-col gap-2">
            <Skeleton className="h-6 w-36" />
            <Skeleton className="h-4 w-24" />
          </div>
        </div>
      ) : user ? (
        <div className="flex items-center gap-4 flex-wrap">
          <Avatar src={avatarSrc(user.avatar_url)} alt={user.username ?? ''} size="lg" />
          <div className="flex flex-col gap-1 min-w-0">
            <h1
              className="text-2xl font-black text-text-primary"
              style={{ fontFamily: 'var(--font-display)' }}
            >
              {user.username ?? 'Unknown user'}
            </h1>
            <div className="flex items-center gap-2 flex-wrap">
              {user.role === 'admin' ? (
                <Badge variant="purple">admin</Badge>
              ) : (
                <Badge variant="default">user</Badge>
              )}
              {user.is_banned && <Badge variant="red">Banned</Badge>}
              {user.is_verified && <Badge variant="green">Verified</Badge>}
              <span className="text-xs text-text-muted font-mono">{user.email}</span>
            </div>
          </div>
        </div>
      ) : (
        <p className="text-text-muted">User not found.</p>
      )}

      {user && (
        <>
          {/* Avatar management */}
          <SectionCard title="Avatar">
            <AvatarSection user={user} userId={id} />
          </SectionCard>

          {/* Ban/Unban */}
          <SectionCard title="Ban management">
            <BanSection user={user} userId={id} />
          </SectionCard>

          {/* Tabs: Profile / Tracking / Missing */}
          <div className="flex flex-col gap-0">
            <div className="flex gap-0 border-b border-space-border">
              {TABS.map((t) => (
                <button
                  key={t.id}
                  onClick={() => setTab(t.id)}
                  className={`px-4 py-2 text-sm font-medium border-b-2 -mb-px transition-colors focus-visible:outline-none
                    ${
                      tab === t.id
                        ? 'border-cosmic-blue text-cosmic-blue'
                        : 'border-transparent text-text-secondary hover:text-text-primary hover:border-space-border'
                    }`}
                >
                  {t.label}
                </button>
              ))}
            </div>
            <div className="pt-4">
              {tab === 'edit' && <EditForm user={user} userId={id} />}
              {tab === 'tracking' && <TrackingTab userId={id} />}
              {tab === 'missing' && <MissingTab userId={id} />}
              {tab === 'submissions' && <SubmissionsTab userId={id} />}
            </div>
          </div>
        </>
      )}
    </div>
  )
}
