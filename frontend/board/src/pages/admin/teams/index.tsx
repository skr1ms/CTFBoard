import { api, isApiError } from '@/shared/api/client'
import type { components } from '@/shared/api/schema.d'
import { useDebounce } from '@/shared/lib/hooks/useDebounce'
import { avatarSrc } from '@/shared/lib/utils'
import { Avatar } from '@/shared/ui/avatar'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { EmptyState } from '@/shared/ui/empty-state'
import { Skeleton } from '@/shared/ui/skeleton'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Link } from 'react-router'
import { toast } from 'sonner'

type AdminTeam = components['schemas']['AdminTeamResponse']
type BracketResponse = components['schemas']['BracketResponse']

// ---------------------------------------------------------------------------
// Hooks - list
// ---------------------------------------------------------------------------
function useAdminTeams(q: string, page: number) {
  return useQuery({
    queryKey: ['admin', 'teams', q, page],
    queryFn: async () => {
      const query: { q?: string; page?: number; per_page?: number } = { page, per_page: 20 }
      if (q) query.q = q
      const { data, error } = await api.GET('/admin/teams', { params: { query } })
      if (error) throw error
      return data ?? { data: [], meta: undefined }
    },
    staleTime: 0,
    placeholderData: (prev) => prev,
  })
}

function useDeleteTeam() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      const { error } = await api.DELETE('/admin/teams/{ID}', { params: { path: { ID: id } } })
      if (error) throw error
    },
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['admin', 'teams'] }),
  })
}

// ---------------------------------------------------------------------------
// Hooks - detail
// ---------------------------------------------------------------------------
function useAdminTeamMembers(id: string) {
  return useQuery({
    queryKey: ['admin', 'team', id, 'members'],
    queryFn: async () => {
      const { data, error } = await api.GET('/admin/teams/{ID}/members', {
        params: { path: { ID: id } },
      })
      if (error) throw error
      return (data ?? []) as components['schemas']['AdminUserResponse'][]
    },
    staleTime: 0,
    enabled: !!id,
  })
}

function useBrackets() {
  return useQuery({
    queryKey: ['brackets'],
    queryFn: async () => {
      const { data, error } = await api.GET('/brackets')
      if (error) throw error
      return (data ?? []) as BracketResponse[]
    },
    staleTime: 60_000,
  })
}

function useUpdateTeam(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (body: {
      name?: string
      captain_id?: string
      bracket_id?: string
      is_hidden?: boolean
    }) => {
      const { data, error } = await api.PATCH('/admin/teams/{ID}', {
        params: { path: { ID: id } },
        body,
      })
      if (error) throw error
      return data
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'teams'] })
      void qc.invalidateQueries({ queryKey: ['admin', 'team', id] })
    },
  })
}

function useBanTeam(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (body: { reason: string; ban_members: boolean }) => {
      const { error } = await api.POST('/admin/teams/{ID}/ban', {
        params: { path: { ID: id } },
        body,
      })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'teams'] })
    },
  })
}

function useUnbanTeam(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async () => {
      const { error } = await api.DELETE('/admin/teams/{ID}/ban', {
        params: { path: { ID: id } },
      })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'teams'] })
    },
  })
}

function useAddMember(teamId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (userId: string) => {
      const { error } = await api.POST('/admin/teams/{ID}/members', {
        params: { path: { ID: teamId } },
        body: { user_id: userId },
      })
      if (error) throw error
    },
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['admin', 'team', teamId, 'members'] }),
  })
}

function useRemoveMember(teamId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (userId: string) => {
      const { error } = await api.DELETE('/admin/teams/{ID}/members/{userID}', {
        params: { path: { ID: teamId, userID: userId } },
      })
      if (error) throw error
    },
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['admin', 'team', teamId, 'members'] }),
  })
}

function useSetBracket(teamId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (bracketId: string | null) => {
      const body: { bracket_id?: string | null } = { bracket_id: bracketId }
      const { error } = await api.PATCH('/admin/teams/{ID}/bracket', {
        params: { path: { ID: teamId } },
        body,
      })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'teams'] })
    },
  })
}

function useSetHidden(teamId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (hidden: boolean) => {
      const { error } = await api.PATCH('/admin/teams/{ID}/hidden', {
        params: { path: { ID: teamId } },
        body: { hidden },
      })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'teams'] })
    },
  })
}

function useDeleteTeamAvatar(teamId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async () => {
      const { error } = await api.DELETE('/admin/teams/{ID}/avatar', {
        params: { path: { ID: teamId } },
      })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'teams'] })
      toast.success('Avatar removed')
    },
    onError: () => toast.error('Failed to remove avatar'),
  })
}

// ---------------------------------------------------------------------------
// Detail panel components
// ---------------------------------------------------------------------------

function EditTeamForm({ team, brackets }: { team: AdminTeam; brackets: BracketResponse[] }) {
  const { mutateAsync: update, isPending } = useUpdateTeam(team.id ?? '')
  const { mutateAsync: setBracket, isPending: bracketPending } = useSetBracket(team.id ?? '')
  const { mutateAsync: setHidden, isPending: hiddenPending } = useSetHidden(team.id ?? '')

  const [prevTeam, setPrevTeam] = useState(team)
  const [name, setName] = useState(team.name ?? '')
  const [captainId, setCaptainId] = useState(team.captain_id ?? '')
  const [bracketId, setBracketId] = useState(team.bracket_id ?? '')
  const [isHidden, setIsHidden] = useState(team.is_hidden ?? false)
  const [error, setError] = useState('')

  if (prevTeam !== team) {
    setPrevTeam(team)
    setName(team.name ?? '')
    setCaptainId(team.captain_id ?? '')
    setBracketId(team.bracket_id ?? '')
    setIsHidden(team.is_hidden ?? false)
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    try {
      const body: Parameters<typeof update>[0] = { name }
      if (captainId) body.captain_id = captainId
      await update(body)

      // bracket separately
      if (bracketId !== (team.bracket_id ?? '')) {
        await setBracket(bracketId || null)
      }
      // hidden separately
      if (isHidden !== (team.is_hidden ?? false)) {
        await setHidden(isHidden)
      }

      toast.success('Team updated')
    } catch (err) {
      setError(isApiError(err) ? err.message : 'Update failed')
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-3">
      <div className="flex flex-col gap-1.5">
        <label className="text-xs font-semibold text-text-muted uppercase tracking-wide">
          Name
        </label>
        <input
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <label className="text-xs font-semibold text-text-muted uppercase tracking-wide">
          Captain ID (UUID)
        </label>
        <input
          type="text"
          value={captainId}
          onChange={(e) => setCaptainId(e.target.value)}
          placeholder="Leave blank to keep current"
          className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary placeholder:text-text-muted font-mono focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <label className="text-xs font-semibold text-text-muted uppercase tracking-wide">
          Bracket
        </label>
        <select
          value={bracketId}
          onChange={(e) => setBracketId(e.target.value)}
          className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
        >
          <option value="">- No bracket -</option>
          {brackets.map((b) => (
            <option key={b.id} value={b.id ?? ''}>
              {b.name ?? b.id}
            </option>
          ))}
        </select>
      </div>
      <label className="flex items-center gap-2 cursor-pointer w-fit">
        <input
          type="checkbox"
          checked={isHidden}
          onChange={(e) => setIsHidden(e.target.checked)}
          className="rounded border-space-border accent-cosmic-blue"
        />
        <span className="text-sm text-text-secondary">Hidden from scoreboard</span>
      </label>
      {error && <p className="text-xs text-nova-red bg-nova-red/10 rounded px-3 py-2">{error}</p>}
      <div>
        <Button
          type="submit"
          variant="primary"
          size="sm"
          loading={isPending || bracketPending || hiddenPending}
        >
          Save changes
        </Button>
      </div>
    </form>
  )
}

function MembersTab({ team }: { team: AdminTeam }) {
  const { data: members, isLoading } = useAdminTeamMembers(team.id ?? '')
  const { mutateAsync: addMember, isPending: adding } = useAddMember(team.id ?? '')
  const { mutateAsync: removeMember, isPending: removing } = useRemoveMember(team.id ?? '')
  const [addUserId, setAddUserId] = useState('')
  const [addError, setAddError] = useState('')

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault()
    setAddError('')
    try {
      await addMember(addUserId.trim())
      toast.success('Member added')
      setAddUserId('')
    } catch (err) {
      setAddError(isApiError(err) ? err.message : 'Failed to add member')
    }
  }

  const handleRemove = async (userId: string, username: string) => {
    try {
      await removeMember(userId)
      toast.success(`Removed ${username}`)
    } catch (err) {
      toast.error(isApiError(err) ? err.message : 'Failed to remove member')
    }
  }

  return (
    <div className="flex flex-col gap-4">
      {/* Add member form */}
      <form onSubmit={handleAdd} className="flex gap-2">
        <input
          type="text"
          value={addUserId}
          onChange={(e) => setAddUserId(e.target.value)}
          placeholder="User UUID…"
          className="flex-1 h-8 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-xs text-text-primary font-mono placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
        />
        <Button
          type="submit"
          size="sm"
          variant="primary"
          loading={adding}
          disabled={!addUserId.trim()}
        >
          Add
        </Button>
      </form>
      {addError && <p className="text-xs text-nova-red">{addError}</p>}

      {/* Members list */}
      {isLoading ? (
        <div className="flex flex-col gap-2">
          {[0, 1, 2].map((i) => (
            <Skeleton key={i} className="h-9 w-full" />
          ))}
        </div>
      ) : !members || members.length === 0 ? (
        <EmptyState message="No members." />
      ) : (
        <div className="flex flex-col gap-1">
          {members.map((m) => (
            <div
              key={m.id}
              className="flex items-center gap-3 px-3 py-2 rounded-[var(--radius-md)] hover:bg-space-border/20 group"
            >
              <Avatar src={avatarSrc(m.avatar_url)} alt={m.username ?? ''} size="sm" />
              <span className="flex-1 text-sm text-text-primary truncate">
                {m.username ?? m.id}
              </span>
              {m.id === team.captain_id && <Badge variant="yellow">Captain</Badge>}
              <button
                onClick={() => handleRemove(m.id ?? '', m.username ?? 'user')}
                disabled={removing}
                className="text-xs text-text-muted hover:text-nova-red transition-colors opacity-0 group-hover:opacity-100 px-1.5 py-1 rounded hover:bg-nova-red/10 disabled:opacity-40"
              >
                Remove
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

function BanSection({ team }: { team: AdminTeam }) {
  const { mutateAsync: banTeam, isPending: banning } = useBanTeam(team.id ?? '')
  const { mutateAsync: unbanTeam, isPending: unbanning } = useUnbanTeam(team.id ?? '')
  const [showBanForm, setShowBanForm] = useState(false)
  const [showUnbanConfirm, setShowUnbanConfirm] = useState(false)
  const [reason, setReason] = useState('')
  const [banMembers, setBanMembers] = useState(false)
  const [error, setError] = useState('')

  const handleBan = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    try {
      await banTeam({ reason, ban_members: banMembers })
      toast.success('Team banned')
      setShowBanForm(false)
      setReason('')
    } catch (err) {
      setError(isApiError(err) ? err.message : 'Ban failed')
    }
  }

  const handleUnban = async () => {
    try {
      await unbanTeam()
      toast.success('Team unbanned')
      setShowUnbanConfirm(false)
    } catch (err) {
      toast.error(isApiError(err) ? err.message : 'Unban failed')
    }
  }

  if (team.is_banned) {
    return (
      <div className="flex flex-col gap-3">
        <div className="flex items-center gap-3 p-3 rounded-[var(--radius-md)] bg-nova-red/10 border border-nova-red/20">
          <Badge variant="red">Banned</Badge>
          <div className="flex-1 min-w-0">
            {team.banned_reason && (
              <p className="text-xs text-text-secondary truncate">Reason: {team.banned_reason}</p>
            )}
            {team.banned_at && (
              <p className="text-xs text-text-muted">{new Date(team.banned_at).toLocaleString()}</p>
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
            Unban team
          </Button>
        )}
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-3">
      {!showBanForm ? (
        <Button size="sm" variant="danger" onClick={() => setShowBanForm(true)}>
          Ban team
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
          <label className="flex items-center gap-2 cursor-pointer w-fit">
            <input
              type="checkbox"
              checked={banMembers}
              onChange={(e) => setBanMembers(e.target.checked)}
              className="rounded border-space-border accent-nova-red"
            />
            <span className="text-sm text-text-secondary">Also ban all members</span>
          </label>
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

function AvatarSection({ team }: { team: AdminTeam }) {
  const qc = useQueryClient()
  const { mutateAsync: deleteAvatar, isPending: deleting } = useDeleteTeamAvatar(team.id ?? '')
  const [preview, setPreview] = useState<string | null>(null)
  const [uploading, setUploading] = useState(false)

  const handleFile = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    setPreview(URL.createObjectURL(file))
    setUploading(true)
    try {
      const form = new FormData()
      form.append('file', file)
      const { error: uploadError } = await api.PUT('/admin/teams/{ID}/avatar', {
        params: { path: { ID: team.id ?? '' } },
        body: form as never,
      })
      if (uploadError) {
        toast.error(isApiError(uploadError) ? uploadError.message : 'Upload failed')
        setPreview(null)
        return
      }
      void qc.invalidateQueries({ queryKey: ['admin', 'teams'] })
      toast.success('Avatar updated')
    } catch {
      toast.error('Upload failed')
      setPreview(null)
    } finally {
      setUploading(false)
      e.target.value = ''
    }
  }

  const currentUrl = preview ?? avatarSrc(team.avatar_url) ?? null

  return (
    <div className="flex items-center gap-4">
      <Avatar src={currentUrl} alt={team.name ?? ''} size="lg" />
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

// ---------------------------------------------------------------------------
// Detail panel
// ---------------------------------------------------------------------------
type DetailTab = 'edit' | 'members' | 'ban'

const DETAIL_TABS: Array<{ id: DetailTab; label: string }> = [
  { id: 'edit', label: 'Details' },
  { id: 'members', label: 'Members' },
  { id: 'ban', label: 'Ban' },
]

function TeamDetailPanel({
  team,
  brackets,
  onClose,
}: {
  team: AdminTeam
  brackets: BracketResponse[]
  onClose: () => void
}) {
  const [tab, setTab] = useState<DetailTab>('edit')

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center gap-3 p-4 border-b border-space-border">
        <Avatar src={avatarSrc(team.avatar_url)} alt={team.name ?? ''} size="md" />
        <div className="flex-1 min-w-0">
          <p className="font-semibold text-text-primary truncate">{team.name ?? '-'}</p>
          <div className="flex items-center gap-1.5 flex-wrap mt-0.5">
            {team.is_banned && <Badge variant="red">Banned</Badge>}
            {team.is_hidden && <Badge variant="default">Hidden</Badge>}
            {team.is_solo && <Badge variant="cyan">Solo</Badge>}
          </div>
        </div>
        <button
          onClick={onClose}
          className="text-text-muted hover:text-text-primary transition-colors p-1 rounded hover:bg-space-border/50"
        >
          <svg className="w-5 h-5" viewBox="0 0 16 16" fill="currentColor">
            <path d="M3.72 3.72a.75.75 0 0 1 1.06 0L8 6.94l3.22-3.22a.749.749 0 0 1 1.275.326.749.749 0 0 1-.215.734L9.06 8l3.22 3.22a.749.749 0 0 1-.326 1.275.749.749 0 0 1-.734-.215L8 9.06l-3.22 3.22a.751.751 0 0 1-1.042-.018.751.751 0 0 1-.018-1.042L6.94 8 3.72 4.78a.75.75 0 0 1 0-1.06Z" />
          </svg>
        </button>
      </div>

      {/* Avatar */}
      <div className="px-4 pt-4">
        <AvatarSection key={team.id} team={team} />
      </div>

      {/* Tabs */}
      <div className="flex border-b border-space-border mt-4 px-4">
        {DETAIL_TABS.map((t) => (
          <button
            key={t.id}
            onClick={() => setTab(t.id)}
            className={`px-3 py-2 text-sm font-medium border-b-2 -mb-px transition-colors focus-visible:outline-none
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

      {/* Tab content */}
      <div className="flex-1 overflow-y-auto p-4">
        {tab === 'edit' && <EditTeamForm team={team} brackets={brackets} />}
        {tab === 'members' && <MembersTab team={team} />}
        {tab === 'ban' && <BanSection team={team} />}
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------
export function AdminTeamsPage() {
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [selectedTeam, setSelectedTeam] = useState<AdminTeam | null>(null)

  const { mutateAsync: deleteTeam, isPending: deleting } = useDeleteTeam()
  const { data: brackets = [] } = useBrackets()

  const debouncedSearch = useDebounce(search, 300)

  const { data, isLoading: loading, isFetching: fetching } = useAdminTeams(debouncedSearch, page)
  const displayTeams = data?.data ?? []
  const displayMeta = data?.meta

  const handleDelete = async (id: string, name: string, e: React.MouseEvent) => {
    e.stopPropagation()
    try {
      await deleteTeam(id)
      toast.success(`Deleted "${name}"`)
      if (selectedTeam?.id === id) setSelectedTeam(null)
    } catch (err) {
      toast.error(isApiError(err) ? err.message : 'Delete failed')
    }
  }

  return (
    <div className="flex gap-4 h-full">
      {/* List */}
      <div
        className={`flex flex-col gap-4 flex-1 min-w-0 transition-all ${selectedTeam ? 'hidden lg:flex' : 'flex'}`}
      >
        {/* Header */}
        <div className="flex items-center justify-between gap-3 flex-wrap">
          <h1
            className="text-2xl font-black text-text-primary"
            style={{ fontFamily: 'var(--font-display)' }}
          >
            Teams
          </h1>
          {displayMeta?.total !== undefined && (
            <span className="text-sm text-text-muted">{displayMeta.total} total</span>
          )}
        </div>

        {/* Search */}
        <div className="relative max-w-sm">
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
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search by name…"
            className="w-full h-9 pl-9 pr-3 rounded-[var(--radius-md)] border border-space-border bg-space-dark text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
          />
          {fetching && !loading && (
            <div className="absolute right-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 border-2 border-cosmic-blue border-t-transparent rounded-full animate-spin" />
          )}
        </div>

        {/* Table */}
        <div
          className={`overflow-x-auto rounded-[var(--radius-lg)] border border-space-border transition-opacity ${fetching && !loading ? 'opacity-70' : ''}`}
        >
          <table className="w-full min-w-[560px] text-sm">
            <thead>
              <tr className="border-b border-space-border bg-space-dark/60">
                <th className="px-4 py-3 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
                  Team
                </th>
                <th className="px-4 py-3 text-center text-xs font-semibold text-text-muted uppercase tracking-wide hidden sm:table-cell">
                  Members
                </th>
                <th className="px-4 py-3 text-center text-xs font-semibold text-text-muted uppercase tracking-wide hidden md:table-cell">
                  Bracket
                </th>
                <th className="px-4 py-3 text-center text-xs font-semibold text-text-muted uppercase tracking-wide">
                  Status
                </th>
                <th className="w-16 px-3 py-3" />
              </tr>
            </thead>
            <tbody>
              {loading ? (
                Array.from({ length: 8 }).map((_, i) => (
                  <tr key={i} className="border-b border-space-border last:border-b-0">
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-3">
                        <Skeleton circle className="w-7 h-7" />
                        <Skeleton className="h-4 w-32" />
                      </div>
                    </td>
                    <td className="px-4 py-3 text-center hidden sm:table-cell">
                      <Skeleton className="h-4 w-8 mx-auto" />
                    </td>
                    <td className="px-4 py-3 text-center hidden md:table-cell">
                      <Skeleton className="h-5 w-20 mx-auto" />
                    </td>
                    <td className="px-4 py-3 text-center">
                      <Skeleton className="h-5 w-16 mx-auto" />
                    </td>
                    <td className="px-3 py-3" />
                  </tr>
                ))
              ) : displayTeams.length === 0 ? (
                <tr>
                  <td colSpan={5}>
                    <EmptyState
                      message={debouncedSearch ? 'No teams match your search.' : 'No teams yet.'}
                    />
                  </td>
                </tr>
              ) : (
                displayTeams.map((t) => {
                  const bracketName = t.bracket_id
                    ? (brackets.find((b) => b.id === t.bracket_id)?.name ??
                      t.bracket_id.slice(0, 8))
                    : null
                  const isActive = selectedTeam?.id === t.id
                  return (
                    <tr
                      key={t.id}
                      data-testid={`admin-row-${t.id}`}
                      onClick={() => setSelectedTeam(t)}
                      className={`border-b border-space-border last:border-b-0 transition-colors cursor-pointer
                        ${isActive ? 'bg-cosmic-blue/10' : 'hover:bg-space-border/20'}`}
                    >
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-3">
                          <Avatar src={avatarSrc(t.avatar_url)} alt={t.name ?? ''} size="sm" />
                          <span
                            className={`font-medium truncate max-w-[160px] ${isActive ? 'text-cosmic-blue' : 'text-text-primary'}`}
                          >
                            {t.name ?? '-'}
                          </span>
                        </div>
                      </td>
                      <td className="px-4 py-3 text-center tabular-nums text-text-secondary hidden sm:table-cell">
                        {t.member_count ?? 0}
                      </td>
                      <td className="px-4 py-3 text-center hidden md:table-cell">
                        {bracketName ? (
                          <Badge variant="blue">{bracketName}</Badge>
                        ) : (
                          <span className="text-xs text-text-muted">-</span>
                        )}
                      </td>
                      <td className="px-4 py-3 text-center">
                        {t.is_banned ? (
                          <Badge variant="red">Banned</Badge>
                        ) : t.is_hidden ? (
                          <Badge variant="default">Hidden</Badge>
                        ) : (
                          <span className="text-xs text-nebula-green">Active</span>
                        )}
                      </td>
                      <td className="px-3 py-3 text-right" onClick={(e) => e.stopPropagation()}>
                        <div className="flex items-center justify-end gap-1">
                          <Link
                            to={`/admin/teams/${t.id}`}
                            onClick={(e) => e.stopPropagation()}
                            className="text-xs text-text-muted hover:text-cosmic-blue transition-colors px-1.5 py-1 rounded hover:bg-cosmic-blue/10"
                          >
                            View
                          </Link>
                          <button
                            data-testid={`admin-delete-${t.id}`}
                            onClick={(e) => handleDelete(t.id ?? '', t.name ?? 'this team', e)}
                            disabled={deleting}
                            className="text-xs text-text-muted hover:text-nova-red transition-colors px-1.5 py-1 rounded hover:bg-nova-red/10 disabled:opacity-40"
                          >
                            Delete
                          </button>
                        </div>
                      </td>
                    </tr>
                  )
                })
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        {displayMeta && (displayMeta.total_pages ?? 1) > 1 && (
          <div className="flex items-center justify-between text-sm">
            <button
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page <= 1 || fetching}
              className="px-3 py-1.5 rounded-[var(--radius-md)] border border-space-border text-text-secondary hover:text-text-primary hover:bg-space-border/50 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
            >
              ‹ Previous
            </button>
            <span className="text-text-muted">
              Page {page} of {displayMeta.total_pages ?? 1}
            </span>
            <button
              onClick={() => setPage((p) => p + 1)}
              disabled={page >= (displayMeta.total_pages ?? 1) || fetching}
              className="px-3 py-1.5 rounded-[var(--radius-md)] border border-space-border text-text-secondary hover:text-text-primary hover:bg-space-border/50 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
            >
              Next ›
            </button>
          </div>
        )}
      </div>

      {/* Detail panel */}
      {selectedTeam && (
        <div className="w-full lg:w-[400px] flex-shrink-0 rounded-[var(--radius-lg)] border border-space-border bg-space-card flex flex-col overflow-hidden">
          <TeamDetailPanel
            key={selectedTeam.id}
            team={selectedTeam}
            brackets={brackets}
            onClose={() => setSelectedTeam(null)}
          />
        </div>
      )}
    </div>
  )
}
