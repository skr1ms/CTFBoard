import { api, isApiError } from '@/shared/api/client'
import { useDebounce } from '@/shared/lib/hooks/useDebounce'
import { avatarSrc } from '@/shared/lib/utils'
import { Avatar } from '@/shared/ui/avatar'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { EmptyState } from '@/shared/ui/empty-state'
import { PasswordInput } from '@/shared/ui/password-input'
import { Skeleton } from '@/shared/ui/skeleton'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useNavigate } from 'react-router'
import { toast } from 'sonner'

// ---------------------------------------------------------------------------
// Hooks
// ---------------------------------------------------------------------------
function useAdminUsers(q: string, field: 'username' | 'ip', page: number) {
  return useQuery({
    queryKey: ['admin', 'users', q, field, page],
    queryFn: async () => {
      const query: { q?: string; field?: 'username' | 'ip'; page?: number; per_page?: number } = {
        page,
        per_page: 20,
      }
      if (q) {
        query.q = q
        query.field = field
      }
      const { data, error } = await api.GET('/admin/users', { params: { query } })
      if (error) throw error
      return data ?? { data: [], meta: undefined }
    },
    staleTime: 0,
    placeholderData: (prev) => prev,
  })
}

function useDeleteUser() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      const { error } = await api.DELETE('/admin/users/{ID}', {
        params: { path: { ID: id } },
      })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'users'] })
    },
  })
}

function useCreateUser() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (body: {
      username: string
      email: string
      password: string
      role?: string
    }) => {
      const { data, error } = await api.POST('/admin/users', { body })
      if (error) throw error
      return data
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'users'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Badges
// ---------------------------------------------------------------------------
function RoleBadge({ role }: { role?: string | undefined }) {
  if (role === 'admin') return <Badge variant="purple">admin</Badge>
  return <Badge variant="default">user</Badge>
}

function BannedBadge() {
  return <Badge variant="red">Banned</Badge>
}

// ---------------------------------------------------------------------------
// Create user modal
// ---------------------------------------------------------------------------
function CreateUserModal({ onClose }: { onClose: () => void }) {
  const { mutateAsync: createUser, isPending } = useCreateUser()
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [role, setRole] = useState<'user' | 'admin'>('user')
  const [error, setError] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    try {
      await createUser({ username, email, password, role })
      toast.success(`User "${username}" created`)
      onClose()
    } catch (err) {
      setError(isApiError(err) ? err.message : 'Failed to create user')
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
    >
      <div className="w-full max-w-md rounded-[var(--radius-lg)] border border-space-border bg-space-card p-6 flex flex-col gap-4 shadow-2xl">
        <div className="flex items-center justify-between">
          <h2
            className="text-lg font-bold text-text-primary"
            style={{ fontFamily: 'var(--font-display)' }}
          >
            Create User
          </h2>
          <button
            onClick={onClose}
            className="text-text-muted hover:text-text-primary transition-colors p-1 rounded hover:bg-space-border/50"
          >
            <svg className="w-5 h-5" viewBox="0 0 16 16" fill="currentColor">
              <path d="M3.72 3.72a.75.75 0 0 1 1.06 0L8 6.94l3.22-3.22a.749.749 0 0 1 1.275.326.749.749 0 0 1-.215.734L9.06 8l3.22 3.22a.749.749 0 0 1-.326 1.275.749.749 0 0 1-.734-.215L8 9.06l-3.22 3.22a.751.751 0 0 1-1.042-.018.751.751 0 0 1-.018-1.042L6.94 8 3.72 4.78a.75.75 0 0 1 0-1.06Z" />
            </svg>
          </button>
        </div>

        <form onSubmit={handleSubmit} className="flex flex-col gap-3">
          <div className="flex flex-col gap-1.5">
            <label className="text-xs font-semibold text-text-muted uppercase tracking-wide">
              Username
            </label>
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              required
              className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
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
              required
              className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
            />
          </div>
          <PasswordInput
            label="Password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            autoComplete="new-password"
          />
          <div className="flex flex-col gap-1.5">
            <label className="text-xs font-semibold text-text-muted uppercase tracking-wide">
              Role
            </label>
            <select
              value={role}
              onChange={(e) => setRole(e.target.value as 'user' | 'admin')}
              className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
            >
              <option value="user">user</option>
              <option value="admin">admin</option>
            </select>
          </div>

          {error && (
            <p className="text-xs text-nova-red bg-nova-red/10 rounded px-3 py-2">{error}</p>
          )}

          <div className="flex gap-2 pt-1">
            <Button
              type="submit"
              variant="primary"
              size="sm"
              loading={isPending}
              className="flex-1"
            >
              Create
            </Button>
            <Button type="button" variant="ghost" size="sm" onClick={onClose} disabled={isPending}>
              Cancel
            </Button>
          </div>
        </form>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Bulk confirm bar
// ---------------------------------------------------------------------------
function BulkDeleteBar({
  count,
  onConfirm,
  onCancel,
  loading,
}: {
  count: number
  onConfirm: () => void
  onCancel: () => void
  loading: boolean
}) {
  return (
    <div className="flex items-center gap-3 px-4 py-2.5 rounded-[var(--radius-md)] bg-nova-red/10 border border-nova-red/30 text-sm">
      <span className="flex-1 text-text-secondary">
        Delete <strong className="text-nova-red">{count}</strong> user{count !== 1 ? 's' : ''}? This
        cannot be undone.
      </span>
      <Button size="sm" variant="danger" loading={loading} onClick={onConfirm}>
        Delete all
      </Button>
      <Button size="sm" variant="ghost" disabled={loading} onClick={onCancel}>
        Cancel
      </Button>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------
export function AdminUsersPage() {
  const navigate = useNavigate()
  const [search, setSearch] = useState('')
  const [searchField, setSearchField] = useState<'username' | 'ip'>('username')
  const [page, setPage] = useState(1)
  const debouncedSearch = useDebounce(search, 300)

  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [confirmBulk, setConfirmBulk] = useState(false)
  const [showCreate, setShowCreate] = useState(false)

  const { data, isLoading, isFetching } = useAdminUsers(debouncedSearch, searchField, page)
  const { mutateAsync: deleteUser, isPending: deleting } = useDeleteUser()

  const users = data?.data ?? []
  const meta = data?.meta

  const handleSearchChange = (val: string) => {
    setSearch(val)
    setPage(1)
    setSelected(new Set())
  }

  const handleFieldChange = (val: 'username' | 'ip') => {
    setSearchField(val)
    setPage(1)
    setSelected(new Set())
  }

  // Selection
  const pageIds = users.map((u) => u.id ?? '').filter(Boolean)
  const allSelected = pageIds.length > 0 && pageIds.every((id) => selected.has(id))

  const toggleAll = () => {
    if (allSelected) {
      setSelected((prev) => {
        const next = new Set(prev)
        pageIds.forEach((id) => next.delete(id))
        return next
      })
    } else {
      setSelected((prev) => new Set([...prev, ...pageIds]))
    }
  }

  const toggleOne = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const handleBulkDelete = async () => {
    const ids = Array.from(selected).filter(Boolean)
    let failed = 0
    await Promise.all(
      ids.map((id) =>
        deleteUser(id).catch((err) => {
          failed++
          console.error('Delete failed', id, err)
        }),
      ),
    )
    if (failed === 0) {
      toast.success(`Deleted ${ids.length} user${ids.length !== 1 ? 's' : ''}`)
    } else {
      toast.error(`${failed} deletion${failed !== 1 ? 's' : ''} failed`)
    }
    setSelected(new Set())
    setConfirmBulk(false)
  }

  const handleSingleDelete = async (id: string, username: string) => {
    try {
      await deleteUser(id)
      toast.success(`Deleted "${username}"`)
      setSelected((prev) => {
        const next = new Set(prev)
        next.delete(id)
        return next
      })
    } catch (err) {
      toast.error(isApiError(err) ? err.message : 'Delete failed')
    }
  }

  const selectedCount = selected.size

  return (
    <>
      {showCreate && <CreateUserModal onClose={() => setShowCreate(false)} />}

      <div className="flex flex-col gap-4">
        {/* Header */}
        <div className="flex items-center justify-between gap-3 flex-wrap">
          <h1
            className="text-2xl font-black text-text-primary"
            style={{ fontFamily: 'var(--font-display)' }}
          >
            Users
          </h1>
          <div className="flex items-center gap-2">
            {selectedCount > 0 && !confirmBulk && (
              <Button
                data-testid="admin-bulk-delete"
                variant="danger"
                size="sm"
                onClick={() => setConfirmBulk(true)}
              >
                Delete {selectedCount} selected
              </Button>
            )}
            <Button variant="primary" size="sm" onClick={() => setShowCreate(true)}>
              + Create User
            </Button>
          </div>
        </div>

        {/* Bulk confirm */}
        {confirmBulk && (
          <BulkDeleteBar
            count={selectedCount}
            onConfirm={handleBulkDelete}
            onCancel={() => setConfirmBulk(false)}
            loading={deleting}
          />
        )}

        {/* Search row */}
        <div className="flex gap-2 flex-wrap">
          <div className="relative flex-1 min-w-[200px] max-w-sm">
            <svg
              className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-muted pointer-events-none"
              viewBox="0 0 16 16"
              fill="currentColor"
            >
              <path d="M10.68 11.74a6 6 0 0 1-7.922-8.982 6 6 0 0 1 8.982 7.922l3.04 3.04a.749.749 0 0 1-.326 1.275.749.749 0 0 1-.734-.215ZM11.5 7a4.499 4.499 0 1 0-8.997 0A4.499 4.499 0 0 0 11.5 7Z" />
            </svg>
            <input
              type="text"
              value={search}
              onChange={(e) => handleSearchChange(e.target.value)}
              placeholder={searchField === 'ip' ? 'Search by IP address…' : 'Search by username…'}
              className="w-full h-9 pl-9 pr-3 rounded-[var(--radius-md)] border border-space-border bg-space-dark text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
            />
            {isFetching && !isLoading && (
              <div className="absolute right-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 border-2 border-cosmic-blue border-t-transparent rounded-full animate-spin" />
            )}
          </div>
          <select
            value={searchField}
            onChange={(e) => handleFieldChange(e.target.value as 'username' | 'ip')}
            className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
          >
            <option value="username">Username</option>
            <option value="ip">IP address</option>
          </select>
        </div>

        {/* Table */}
        <div
          className={`overflow-x-auto rounded-[var(--radius-lg)] border border-space-border transition-opacity ${isFetching && !isLoading ? 'opacity-70' : ''}`}
        >
          <table className="w-full min-w-[700px] text-sm">
            <thead>
              <tr className="border-b border-space-border bg-space-dark/60">
                <th className="w-10 px-3 py-3">
                  <input
                    type="checkbox"
                    checked={allSelected}
                    onChange={toggleAll}
                    className="rounded border-space-border accent-cosmic-blue"
                    aria-label="Select all"
                  />
                </th>
                <th className="px-4 py-3 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
                  User
                </th>
                <th className="px-4 py-3 text-left text-xs font-semibold text-text-muted uppercase tracking-wide hidden md:table-cell">
                  Email
                </th>
                <th className="px-4 py-3 text-center text-xs font-semibold text-text-muted uppercase tracking-wide">
                  Role
                </th>
                <th className="px-4 py-3 text-center text-xs font-semibold text-text-muted uppercase tracking-wide hidden sm:table-cell">
                  Verified
                </th>
                <th className="px-4 py-3 text-center text-xs font-semibold text-text-muted uppercase tracking-wide">
                  Status
                </th>
                <th className="px-4 py-3 text-left text-xs font-semibold text-text-muted uppercase tracking-wide hidden lg:table-cell">
                  Team
                </th>
                <th className="w-16 px-3 py-3" />
              </tr>
            </thead>
            <tbody>
              {isLoading ? (
                Array.from({ length: 8 }).map((_, i) => (
                  <tr key={i} className="border-b border-space-border last:border-b-0">
                    <td className="px-3 py-3">
                      <Skeleton className="h-4 w-4" />
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-3">
                        <Skeleton circle className="w-7 h-7" />
                        <Skeleton className="h-4 w-28" />
                      </div>
                    </td>
                    <td className="px-4 py-3 hidden md:table-cell">
                      <Skeleton className="h-4 w-40" />
                    </td>
                    <td className="px-4 py-3 text-center">
                      <Skeleton className="h-5 w-14 mx-auto" />
                    </td>
                    <td className="px-4 py-3 text-center hidden sm:table-cell">
                      <Skeleton className="h-4 w-8 mx-auto" />
                    </td>
                    <td className="px-4 py-3 text-center">
                      <Skeleton className="h-5 w-16 mx-auto" />
                    </td>
                    <td className="px-4 py-3 hidden lg:table-cell">
                      <Skeleton className="h-4 w-24" />
                    </td>
                    <td className="px-3 py-3" />
                  </tr>
                ))
              ) : users.length === 0 ? (
                <tr>
                  <td colSpan={8}>
                    <EmptyState
                      message={debouncedSearch ? 'No users match your search.' : 'No users yet.'}
                    />
                  </td>
                </tr>
              ) : (
                users.map((u) => {
                  const id = u.id ?? ''
                  const isSelected = selected.has(id)
                  return (
                    <tr
                      key={id}
                      data-testid={`admin-row-${id}`}
                      onClick={() => id && navigate(`/admin/users/${id}`, { state: { user: u } })}
                      className={`border-b border-space-border last:border-b-0 transition-colors cursor-pointer
                        ${isSelected ? 'bg-cosmic-blue/5' : 'hover:bg-space-border/20'}`}
                    >
                      <td className="px-3 py-3" onClick={(e) => e.stopPropagation()}>
                        <input
                          type="checkbox"
                          checked={isSelected}
                          onChange={() => toggleOne(id)}
                          className="rounded border-space-border accent-cosmic-blue"
                          aria-label={`Select ${u.username}`}
                        />
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-3">
                          <Avatar
                            src={avatarSrc(u.avatar_url)}
                            {...(u.username ? { alt: u.username } : {})}
                            size="sm"
                          />
                          <span className="font-medium text-text-primary truncate max-w-[140px] group-hover:text-cosmic-blue transition-colors">
                            {u.username ?? '-'}
                          </span>
                        </div>
                      </td>
                      <td className="px-4 py-3 text-text-secondary hidden md:table-cell truncate max-w-[200px]">
                        {u.email ?? '-'}
                      </td>
                      <td className="px-4 py-3 text-center">
                        <RoleBadge role={u.role} />
                      </td>
                      <td className="px-4 py-3 text-center text-text-muted hidden sm:table-cell">
                        {u.is_verified ? (
                          <span className="text-nebula-green text-base">✓</span>
                        ) : (
                          <span className="text-text-muted text-base">-</span>
                        )}
                      </td>
                      <td className="px-4 py-3 text-center">
                        {u.is_banned ? (
                          <BannedBadge />
                        ) : (
                          <span className="text-xs text-text-muted">Active</span>
                        )}
                      </td>
                      <td className="px-4 py-3 text-text-muted font-mono text-xs hidden lg:table-cell">
                        {u.team_id ? `${u.team_id.slice(0, 8)}…` : '-'}
                      </td>
                      <td className="px-3 py-3 text-right" onClick={(e) => e.stopPropagation()}>
                        <button
                          data-testid={`admin-delete-${id}`}
                          onClick={() => handleSingleDelete(id, u.username ?? 'this user')}
                          disabled={deleting}
                          className="text-xs text-text-muted hover:text-nova-red transition-colors px-1.5 py-1 rounded hover:bg-nova-red/10 disabled:opacity-40"
                        >
                          Delete
                        </button>
                      </td>
                    </tr>
                  )
                })
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        {meta && (meta.total_pages ?? 1) > 1 && (
          <div className="flex items-center justify-between text-sm">
            <button
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page <= 1 || isFetching}
              className="px-3 py-1.5 rounded-[var(--radius-md)] border border-space-border text-text-secondary hover:text-text-primary hover:bg-space-border/50 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
            >
              ‹ Previous
            </button>
            <span className="text-text-muted">
              Page {page} of {meta.total_pages ?? 1}
              {meta.total !== undefined && <span className="ml-2">({meta.total} total)</span>}
            </span>
            <button
              onClick={() => setPage((p) => p + 1)}
              disabled={page >= (meta.total_pages ?? 1) || isFetching}
              className="px-3 py-1.5 rounded-[var(--radius-md)] border border-space-border text-text-secondary hover:text-text-primary hover:bg-space-border/50 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
            >
              Next ›
            </button>
          </div>
        )}
      </div>
    </>
  )
}
