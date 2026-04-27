import {
  type SubmissionResponse,
  useUserAwards,
  useUserFails,
  useUserProfile,
  useUserSolves,
  useUsers,
} from '@/features/users/useUsers'
import { useDebounce } from '@/shared/lib/hooks/useDebounce'
import { avatarSrc } from '@/shared/lib/utils'
import { Avatar } from '@/shared/ui/avatar'
import { Badge } from '@/shared/ui/badge'
import { EmptyState } from '@/shared/ui/empty-state'
import { Skeleton } from '@/shared/ui/skeleton'
import { VisibilityGate } from '@/shared/ui/visibility-gate'
import { useState } from 'react'
import { useNavigate, useParams } from 'react-router'

// ---------------------------------------------------------------------------
// ---------------------------------------------------------------------------
// UserProfilePage
// ---------------------------------------------------------------------------
function SolvesTable({ userId }: { userId: string }) {
  const { data: solves, isLoading } = useUserSolves(userId)

  if (isLoading) {
    return (
      <div className="flex flex-col gap-2">
        {[0, 1, 2].map((i) => (
          <Skeleton key={i} className="h-10 w-full" />
        ))}
      </div>
    )
  }

  if (!solves || solves.length === 0) {
    return <EmptyState message="No solves yet." />
  }

  return (
    <div className="overflow-x-auto rounded-[var(--radius-lg)] border border-space-border">
      <table className="w-full min-w-[400px]">
        <thead>
          <tr className="border-b border-space-border bg-space-dark/60">
            <th className="px-4 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
              Challenge
            </th>
            <th className="px-4 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide hidden sm:table-cell">
              Category
            </th>
            <th className="px-4 py-2.5 text-right text-xs font-semibold text-text-muted uppercase tracking-wide">
              Points
            </th>
            <th className="px-4 py-2.5 text-right text-xs font-semibold text-text-muted uppercase tracking-wide hidden md:table-cell">
              Solved At
            </th>
          </tr>
        </thead>
        <tbody>
          {solves.map((s) => (
            <tr
              key={s.id}
              className="border-b border-space-border last:border-b-0 hover:bg-space-border/20 transition-colors"
            >
              <td className="px-4 py-2.5 text-sm text-text-primary">{s.challenge_title ?? '-'}</td>
              <td className="px-4 py-2.5 hidden sm:table-cell">
                {s.challenge_category && (
                  <Badge variant="default" className="text-xs">
                    {s.challenge_category}
                  </Badge>
                )}
              </td>
              <td className="px-4 py-2.5 text-right text-sm font-mono font-bold text-cosmic-blue">
                {s.challenge_points ?? '-'}
              </td>
              <td className="px-4 py-2.5 text-right text-xs text-text-muted hidden md:table-cell whitespace-nowrap">
                {s.solved_at
                  ? new Date(s.solved_at).toLocaleString(undefined, {
                      month: 'short',
                      day: 'numeric',
                      hour: '2-digit',
                      minute: '2-digit',
                    })
                  : '-'}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function AwardsTable({ userId }: { userId: string }) {
  const { data: awards, isLoading } = useUserAwards(userId)

  if (isLoading) {
    return <Skeleton className="h-16 w-full" />
  }

  if (!awards || awards.length === 0) {
    return <EmptyState message="No awards yet." />
  }

  return (
    <div className="overflow-x-auto rounded-[var(--radius-lg)] border border-space-border">
      <table className="w-full">
        <thead>
          <tr className="border-b border-space-border bg-space-dark/60">
            <th className="px-4 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
              Description
            </th>
            <th className="px-4 py-2.5 text-right text-xs font-semibold text-text-muted uppercase tracking-wide">
              Value
            </th>
          </tr>
        </thead>
        <tbody>
          {awards.map((a) => (
            <tr
              key={a.id}
              className="border-b border-space-border last:border-b-0 hover:bg-space-border/20 transition-colors"
            >
              <td className="px-4 py-2.5 text-sm text-text-primary">{a.description ?? '-'}</td>
              <td className="px-4 py-2.5 text-right text-sm font-mono font-bold text-solar-yellow">
                {a.value ?? 0}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function FailsTable({ userId }: { userId: string }) {
  const { data: fails, isLoading, error } = useUserFails(userId)

  if (isLoading) {
    return (
      <div className="flex flex-col gap-2">
        {[0, 1, 2].map((i) => (
          <Skeleton key={i} className="h-10 w-full" />
        ))}
      </div>
    )
  }

  if (error) {
    return <p className="text-sm text-text-muted italic py-4">Not available.</p>
  }

  if (!fails || fails.length === 0) {
    return <EmptyState message="No failed attempts." />
  }

  return (
    <div className="overflow-x-auto rounded-[var(--radius-lg)] border border-space-border">
      <table className="w-full min-w-[400px]">
        <thead>
          <tr className="border-b border-space-border bg-space-dark/60">
            <th className="px-4 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
              Challenge
            </th>
            <th className="px-4 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide hidden sm:table-cell">
              Submitted flag
            </th>
            <th className="px-4 py-2.5 text-right text-xs font-semibold text-text-muted uppercase tracking-wide hidden md:table-cell">
              Submitted at
            </th>
          </tr>
        </thead>
        <tbody>
          {fails.map((s: SubmissionResponse) => (
            <tr
              key={s.id}
              className="border-b border-space-border last:border-b-0 hover:bg-space-border/20 transition-colors"
            >
              <td className="px-4 py-2.5 text-sm text-text-primary">{s.challenge_title ?? '-'}</td>
              <td className="px-4 py-2.5 text-xs font-mono text-nova-red hidden sm:table-cell max-w-[200px]">
                <span className="truncate block" title={s.submitted_flag ?? ''}>
                  {s.submitted_flag ?? '-'}
                </span>
              </td>
              <td className="px-4 py-2.5 text-right text-xs text-text-muted hidden md:table-cell whitespace-nowrap">
                {s.created_at
                  ? new Date(s.created_at).toLocaleString(undefined, {
                      month: 'short',
                      day: 'numeric',
                      hour: '2-digit',
                      minute: '2-digit',
                    })
                  : '-'}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export function UserProfilePage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data: profile, isLoading } = useUserProfile(id ?? '')

  if (!id) return null

  return (
    <div className="flex flex-col gap-6 max-w-3xl mx-auto px-4 py-6 sm:px-6">
      <button
        onClick={() => navigate('/users')}
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

      {/* Profile header */}
      {isLoading ? (
        <div className="flex items-center gap-4">
          <Skeleton circle className="w-16 h-16" />
          <div className="flex flex-col gap-2">
            <Skeleton className="h-6 w-32" />
            <Skeleton className="h-4 w-24" />
          </div>
        </div>
      ) : profile ? (
        <div className="flex items-center gap-4">
          <Avatar
            src={avatarSrc(profile.avatar_url)}
            {...(profile.username ? { alt: profile.username } : {})}
            size="lg"
          />
          <div className="flex flex-col gap-1 min-w-0">
            <h1
              className="text-2xl font-black text-text-primary"
              style={{ fontFamily: 'var(--font-display)' }}
            >
              {profile.username ?? 'Unknown user'}
            </h1>
            {profile.created_at && (
              <p className="text-xs text-text-muted">
                Joined{' '}
                {new Date(profile.created_at).toLocaleDateString(undefined, {
                  month: 'long',
                  year: 'numeric',
                })}
              </p>
            )}
          </div>
        </div>
      ) : null}

      {profile && (
        <>
          {/* Solves */}
          <div className="flex flex-col gap-3">
            <h2 className="text-sm font-semibold text-text-muted uppercase tracking-wide">
              Solves
            </h2>
            <SolvesTable userId={id} />
          </div>

          {/* Awards */}
          <div className="flex flex-col gap-3">
            <h2 className="text-sm font-semibold text-text-muted uppercase tracking-wide">
              Awards
            </h2>
            <AwardsTable userId={id} />
          </div>

          {/* Fails */}
          <div className="flex flex-col gap-3">
            <h2 className="text-sm font-semibold text-text-muted uppercase tracking-wide">
              Failed Attempts
            </h2>
            <FailsTable userId={id} />
          </div>
        </>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// UsersListPage
// ---------------------------------------------------------------------------
function UserRow({
  user,
  onClick,
}: {
  user: { id?: string; username?: string; team_id?: string; avatar_url?: string }
  onClick: () => void
}) {
  return (
    <tr
      onClick={onClick}
      className="border-b border-space-border last:border-b-0 hover:bg-space-border/30 cursor-pointer transition-colors group"
    >
      <td className="px-4 py-3">
        <div className="flex items-center gap-3">
          <Avatar
            src={avatarSrc(user.avatar_url)}
            {...(user.username ? { alt: user.username } : {})}
            size="sm"
          />
          <span className="text-sm font-medium text-text-primary group-hover:text-cosmic-blue transition-colors">
            {user.username ?? '-'}
          </span>
        </div>
      </td>
      <td className="px-4 py-3 text-sm text-text-secondary hidden sm:table-cell">
        {user.team_id ? (
          <span className="text-xs text-text-muted font-mono truncate max-w-[120px] block">
            {user.team_id.slice(0, 8)}…
          </span>
        ) : (
          <span className="text-xs text-text-muted italic">No team</span>
        )}
      </td>
    </tr>
  )
}

export function UsersPage() {
  const navigate = useNavigate()
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const debouncedSearch = useDebounce(search, 400)

  const { data, isLoading, isFetching } = useUsers(debouncedSearch, page)
  const users = data?.data ?? []
  const meta = data?.meta

  const handleSearchChange = (val: string) => {
    setSearch(val)
    setPage(1)
  }

  return (
    <VisibilityGate configKey="account_visibility">
      <div className="flex flex-col gap-5 max-w-3xl mx-auto px-4 py-6 sm:px-6">
        {/* Header */}
        <div className="flex items-center justify-between gap-4">
          <h1
            className="text-2xl font-black text-text-primary"
            style={{ fontFamily: 'var(--font-display)' }}
          >
            Users
          </h1>
          {meta?.total !== undefined && (
            <span className="text-sm text-text-muted">{meta.total} total</span>
          )}
        </div>

        {/* Search */}
        <div className="relative">
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
            placeholder="Search by username…"
            className="w-full h-10 pl-9 pr-4 rounded-[var(--radius-md)] border border-space-border bg-space-dark text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/50 focus:border-cosmic-blue"
          />
          {isFetching && !isLoading && (
            <div className="absolute right-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 border-2 border-cosmic-blue border-t-transparent rounded-full animate-spin" />
          )}
        </div>

        {/* Table */}
        <div
          className={`rounded-[var(--radius-lg)] border border-space-border overflow-x-auto transition-opacity ${isFetching ? 'opacity-70' : ''}`}
        >
          <table className="w-full">
            <thead>
              <tr className="border-b border-space-border bg-space-dark/60">
                <th className="px-4 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
                  Username
                </th>
                <th className="px-4 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide hidden sm:table-cell">
                  Team
                </th>
              </tr>
            </thead>
            <tbody>
              {isLoading ? (
                Array.from({ length: 8 }).map((_, i) => (
                  <tr key={i} className="border-b border-space-border last:border-b-0">
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-3">
                        <Skeleton circle className="w-8 h-8" />
                        <Skeleton className="h-4 w-28" />
                      </div>
                    </td>
                    <td className="px-4 py-3 hidden sm:table-cell">
                      <Skeleton className="h-4 w-20" />
                    </td>
                  </tr>
                ))
              ) : users.length === 0 ? (
                <tr>
                  <td colSpan={2} className="px-4 py-12 text-center text-text-muted text-sm">
                    No users found.
                  </td>
                </tr>
              ) : (
                users.map((u) => (
                  <UserRow key={u.id} user={u} onClick={() => u.id && navigate(`/users/${u.id}`)} />
                ))
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        {meta && (meta.total_pages ?? 1) > 1 && (
          <div className="flex items-center justify-between text-sm">
            <button
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page <= 1}
              className="px-3 py-1.5 rounded-[var(--radius-md)] border border-space-border text-text-secondary hover:text-text-primary hover:bg-space-border/50 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
            >
              ‹ Previous
            </button>
            <span className="text-text-muted">
              Page {page} of {meta.total_pages ?? 1}
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
    </VisibilityGate>
  )
}
