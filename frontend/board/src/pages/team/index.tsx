import {
  useDisbandTeam,
  useKickMember,
  useLeaveTeam,
  useMyTeam,
  useRegenerateInvite,
  useTeamInvite,
  useTransferCaptain,
  useUpdateTeamName,
} from '@/features/team/useMyTeam'
import { api, apiErrorMessage, isApiError } from '@/shared/api/client'
import { useCopyToClipboard } from '@/shared/lib/hooks/useCopyToClipboard'
import { avatarSrc } from '@/shared/lib/utils'
import { useAuthStore } from '@/shared/stores/authStore'
import { Avatar } from '@/shared/ui/avatar'
import { Button } from '@/shared/ui/button'
import { EmptyState } from '@/shared/ui/empty-state'
import { FileUpload } from '@/shared/ui/file-upload'
import { Skeleton } from '@/shared/ui/skeleton'
import { useQueryClient } from '@tanstack/react-query'
import { type FormEvent, useEffect, useState } from 'react'
import { useNavigate } from 'react-router'
import { toast } from 'sonner'

// ---------------------------------------------------------------------------
// Team avatar section (captain only upload/delete)
// ---------------------------------------------------------------------------
function TeamAvatarSection({
  avatarUrl,
  teamName,
  isCaptain,
  isSolo,
}: {
  avatarUrl?: string
  teamName: string
  isCaptain: boolean
  isSolo: boolean
}) {
  const qc = useQueryClient()
  const [preview, setPreview] = useState<string | null>(null)
  const [uploading, setUploading] = useState(false)

  useEffect(() => {
    return () => {
      if (preview) URL.revokeObjectURL(preview)
    }
  }, [preview])

  const currentUrl = preview ?? avatarUrl ?? null

  const handleFile = async (file: File) => {
    setPreview(URL.createObjectURL(file))
    setUploading(true)
    try {
      const form = new FormData()
      form.append('file', file)
      const { error: uploadError } = await api.PUT('/teams/me/avatar', { body: form as never })
      if (uploadError) {
        toast.error('Avatar upload failed')
        setPreview(null)
        return
      }
      void qc.invalidateQueries({ queryKey: ['my-team'] })
      toast.success(isSolo ? 'Profile avatar updated' : 'Team avatar updated')
    } catch {
      toast.error('Avatar upload failed')
      setPreview(null)
    } finally {
      setUploading(false)
    }
  }

  const handleDelete = async () => {
    try {
      const { error: deleteError } = await api.DELETE('/teams/me/avatar')
      if (deleteError) {
        toast.error('Failed to remove avatar')
        return
      }
      setPreview(null)
      void qc.invalidateQueries({ queryKey: ['my-team'] })
      toast.success(isSolo ? 'Profile avatar removed' : 'Team avatar removed')
    } catch {
      toast.error('Failed to remove avatar')
    }
  }

  return (
    <div className="flex items-start gap-4">
      <Avatar src={currentUrl} {...(teamName ? { alt: teamName } : {})} size="xl" />
      {isCaptain && (
        <div className="flex flex-col gap-2 flex-1">
          <FileUpload
            onFile={handleFile}
            accept="image/jpeg,image/png,image/webp,image/gif"
            maxSizeMb={5}
            label={`${isSolo ? 'Change profile avatar' : 'Change team avatar'} (JPEG, PNG, WebP, GIF static · max 5 MB)`}
            disabled={uploading}
          />
          {(avatarUrl || preview) && (
            <Button
              variant="ghost"
              size="sm"
              onClick={handleDelete}
              disabled={uploading}
              className="w-fit text-text-muted hover:text-nova-red"
            >
              Remove avatar
            </Button>
          )}
        </div>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Team name section with inline edit
// ---------------------------------------------------------------------------
function TeamNameSection({
  name,
  isCaptain,
  isSolo,
}: {
  name: string
  isCaptain: boolean
  isSolo: boolean
}) {
  const [editing, setEditing] = useState(false)
  const [value, setValue] = useState(name)
  const { mutate, isPending } = useUpdateTeamName()

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault()
    if (!value.trim() || value.trim() === name) {
      setEditing(false)
      return
    }
    mutate(value.trim(), {
      onSuccess: () => {
        setEditing(false)
        toast.success(isSolo ? 'Display name updated' : 'Team name updated')
      },
      onError: (err) => toast.error(apiErrorMessage(err, 'Failed to update name')),
    })
  }

  if (editing) {
    return (
      <form onSubmit={handleSubmit} className="flex items-center gap-2">
        <input
          data-testid="team-name-edit"
          autoFocus
          type="text"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          maxLength={64}
          className="h-9 rounded-[var(--radius-md)] border border-cosmic-blue bg-space-dark px-3 text-lg font-bold text-text-primary focus:outline-none focus:ring-2 focus:ring-cosmic-blue/50 min-w-0"
          style={{ fontFamily: 'var(--font-display)' }}
        />
        <Button type="submit" size="sm" variant="primary" loading={isPending}>
          Save
        </Button>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          onClick={() => {
            setValue(name)
            setEditing(false)
          }}
        >
          Cancel
        </Button>
      </form>
    )
  }

  return (
    <div className="flex items-center gap-3 min-w-0">
      <h1
        className="text-2xl font-black text-text-primary truncate"
        style={{ fontFamily: 'var(--font-display)' }}
      >
        {name}
      </h1>
      {isCaptain && (
        <button
          onClick={() => setEditing(true)}
          className="shrink-0 p-1.5 rounded text-text-muted hover:text-text-primary hover:bg-space-border transition-colors"
          aria-label={isSolo ? 'Edit display name' : 'Edit team name'}
        >
          <svg className="w-4 h-4" viewBox="0 0 16 16" fill="currentColor">
            <path d="M11.013 1.427a1.75 1.75 0 0 1 2.474 0l1.086 1.086a1.75 1.75 0 0 1 0 2.474l-8.61 8.61c-.21.21-.47.364-.756.445l-3.251.93a.75.75 0 0 1-.927-.928l.929-3.25c.081-.286.235-.547.445-.758l8.61-8.61Zm.176 4.823L9.75 4.81l-6.286 6.287a.253.253 0 0 0-.064.108l-.558 1.953 1.953-.558a.253.253 0 0 0 .108-.064Zm1.238-3.763a.25.25 0 0 0-.354 0L10.811 3.75l1.439 1.44 1.263-1.263a.25.25 0 0 0 0-.354Z" />
          </svg>
        </button>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Invite link section
// ---------------------------------------------------------------------------
function InviteLinkSection({ isCaptain }: { isCaptain: boolean }) {
  const { data, isLoading } = useTeamInvite(isCaptain)
  const { mutate: regenerate, isPending } = useRegenerateInvite()
  const { copy, copied } = useCopyToClipboard()

  if (!isCaptain) return null

  const inviteUrl = data?.invite_token
    ? `${window.location.origin}/team/enroll?token=${data.invite_token}`
    : ''

  return (
    <div className="flex flex-col gap-2">
      <h3 className="text-xs font-semibold text-text-muted uppercase tracking-wide">Invite Link</h3>
      <div className="flex items-center gap-2">
        {isLoading ? (
          <Skeleton className="h-9 flex-1" />
        ) : (
          <input
            readOnly
            value={inviteUrl}
            className="flex-1 h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-xs text-text-secondary font-mono focus:outline-none"
          />
        )}
        <Button
          data-testid="team-invite-copy"
          size="sm"
          variant="ghost"
          onClick={() => copy(inviteUrl)}
          disabled={!inviteUrl}
        >
          {copied ? '✓ Copied' : 'Copy'}
        </Button>
        <Button
          data-testid="team-invite-regen"
          size="sm"
          variant="ghost"
          loading={isPending}
          onClick={() =>
            regenerate(undefined, {
              onSuccess: () => toast.success('Invite link regenerated'),
            })
          }
        >
          Regenerate
        </Button>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Members list with kick + transfer captain actions
// ---------------------------------------------------------------------------
interface MembersListProps {
  members: Array<{ id?: string; username?: string; avatar_url?: string }>
  captainId?: string
  currentUserId?: string
  isCaptain: boolean
  frozen: boolean
  showTeamActions: boolean
}

function MembersList({
  members,
  captainId,
  currentUserId,
  isCaptain,
  frozen,
  showTeamActions,
}: MembersListProps) {
  const { mutate: kick, isPending: kicking } = useKickMember()
  const { mutate: transfer, isPending: transferring } = useTransferCaptain()
  const [confirmKick, setConfirmKick] = useState<string | null>(null)
  const [transferOpen, setTransferOpen] = useState(false)

  const otherMembers = members.filter((m) => m.id !== currentUserId)

  const handleKick = (userId: string) => {
    kick(userId, {
      onSuccess: () => {
        toast.success('Member removed')
        setConfirmKick(null)
      },
      onError: (err) => {
        const msg = apiErrorMessage(err, 'Failed to kick member')
        toast.error(msg)
        setConfirmKick(null)
      },
    })
  }

  const handleTransfer = (userId: string) => {
    transfer(userId, {
      onSuccess: () => {
        toast.success('Captain transferred')
        setTransferOpen(false)
      },
      onError: (err) => toast.error(apiErrorMessage(err, 'Failed to transfer captain')),
    })
  }

  return (
    <div className="flex flex-col gap-1">
      {frozen && (
        <div className="mx-3 mt-2 mb-1 px-3 py-2 rounded-[var(--radius-md)] bg-solar-yellow/10 border border-solar-yellow/30 text-xs text-solar-yellow">
          Roster is frozen - member changes are disabled
        </div>
      )}

      {members.map((m) => {
        const isMemberCaptain = m.id === captainId
        const isMe = m.id === currentUserId
        const isConfirming = confirmKick === m.id

        return (
          <div
            key={m.id}
            data-testid={`team-member-row-${m.id}`}
            className="flex items-center gap-3 px-3 py-2 rounded-[var(--radius-md)] hover:bg-space-border/30 transition-colors"
          >
            <Avatar
              src={avatarSrc(m.avatar_url)}
              {...(m.username ? { alt: m.username } : {})}
              size="sm"
            />
            <span className="flex-1 text-sm text-text-primary truncate">
              {m.username ?? 'Unknown'}
            </span>
            <div className="flex items-center gap-1.5 shrink-0">
              {showTeamActions && isMemberCaptain && (
                <span className="text-base" title="Captain" aria-label="Captain">
                  👑
                </span>
              )}
              {isMe && (
                <span className="text-xs px-1.5 py-0.5 rounded bg-cosmic-blue/10 border border-cosmic-blue/30 text-cosmic-blue font-medium">
                  You
                </span>
              )}
              {showTeamActions &&
                isCaptain &&
                !isMe &&
                !frozen &&
                (isConfirming ? (
                  <>
                    <span className="text-xs text-text-muted">Kick?</span>
                    <button
                      onClick={() => m.id && handleKick(m.id)}
                      disabled={kicking}
                      className="text-xs text-nova-red hover:underline disabled:opacity-50"
                    >
                      Yes
                    </button>
                    <button
                      onClick={() => setConfirmKick(null)}
                      className="text-xs text-text-muted hover:underline"
                    >
                      No
                    </button>
                  </>
                ) : (
                  <button
                    data-testid={`team-kick-${m.id}`}
                    onClick={() => setConfirmKick(m.id ?? null)}
                    className="text-xs text-text-muted hover:text-nova-red transition-colors px-1.5 py-0.5 rounded hover:bg-nova-red/10"
                  >
                    Kick
                  </button>
                ))}
            </div>
          </div>
        )
      })}

      {/* Transfer Captain button (captain only, requires ≥2 members) */}
      {showTeamActions && isCaptain && otherMembers.length > 0 && !frozen && (
        <div className="px-3 pt-2 pb-1">
          {transferOpen ? (
            <div className="flex flex-col gap-1.5">
              <span className="text-xs text-text-muted">Transfer captain to:</span>
              {otherMembers.map((m) => (
                <button
                  key={m.id}
                  onClick={() => m.id && handleTransfer(m.id)}
                  disabled={transferring}
                  className="flex items-center gap-2 px-2 py-1.5 rounded-[var(--radius-md)] text-sm text-text-secondary hover:text-text-primary hover:bg-space-border/50 transition-colors text-left disabled:opacity-50"
                >
                  <Avatar
                    src={avatarSrc(m.avatar_url)}
                    {...(m.username ? { alt: m.username } : {})}
                    size="xs"
                  />
                  {m.username ?? 'Unknown'}
                </button>
              ))}
              <button
                onClick={() => setTransferOpen(false)}
                className="text-xs text-text-muted hover:text-text-primary mt-1"
              >
                Cancel
              </button>
            </div>
          ) : (
            <button
              onClick={() => setTransferOpen(true)}
              className="text-xs text-text-muted hover:text-cosmic-blue transition-colors"
            >
              Transfer captain…
            </button>
          )}
        </div>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Danger zone: Leave / Disband
// ---------------------------------------------------------------------------
function DangerZone({ isCaptain, frozen }: { isCaptain: boolean; frozen: boolean }) {
  const navigate = useNavigate()
  const [leaveConfirm, setLeaveConfirm] = useState(false)
  const [disbandStep, setDisbandStep] = useState(0)

  const { mutate: leave, isPending: leaving } = useLeaveTeam()
  const { mutate: disband, isPending: disbanding } = useDisbandTeam()

  const handleLeave = () => {
    leave(undefined, {
      onSuccess: () => {
        toast.success('You left the team')
        void navigate('/team/enroll')
      },
      onError: (err) => {
        const msg = apiErrorMessage(err, 'Failed to leave team')
        if (isApiError(err) && err.code === 'CAPTAIN_CANNOT_LEAVE') {
          toast.error('Transfer captain before leaving')
        } else {
          toast.error(msg)
        }
        setLeaveConfirm(false)
      },
    })
  }

  const handleDisband = () => {
    disband(undefined, {
      onSuccess: () => {
        toast.success('Team disbanded')
        void navigate('/team/enroll')
      },
      onError: (err) => {
        toast.error(apiErrorMessage(err, 'Failed to disband team'))
        setDisbandStep(0)
      },
    })
  }

  return (
    <div className="flex flex-col gap-3 pt-2 border-t border-space-border">
      <h3 className="text-xs font-semibold text-text-muted uppercase tracking-wide">Danger Zone</h3>

      {/* Leave team */}
      {!isCaptain &&
        (leaveConfirm ? (
          <div className="flex items-center gap-3 p-3 rounded-[var(--radius-md)] border border-nova-red/30 bg-nova-red/5">
            <span className="text-sm text-text-secondary flex-1">Leave this team?</span>
            <Button size="sm" variant="danger" loading={leaving} onClick={handleLeave}>
              Leave
            </Button>
            <Button size="sm" variant="ghost" onClick={() => setLeaveConfirm(false)}>
              Cancel
            </Button>
          </div>
        ) : (
          <Button
            data-testid="team-leave"
            variant="ghost"
            size="sm"
            disabled={frozen}
            onClick={() => setLeaveConfirm(true)}
            className="w-fit text-nova-red/80 hover:text-nova-red hover:bg-nova-red/10"
          >
            Leave team
          </Button>
        ))}

      {/* Disband team (captain only, 2-step) */}
      {isCaptain &&
        (disbandStep === 0 ? (
          <Button
            data-testid="team-disband"
            variant="ghost"
            size="sm"
            disabled={frozen}
            onClick={() => setDisbandStep(1)}
            className="w-fit text-nova-red/80 hover:text-nova-red hover:bg-nova-red/10"
          >
            Disband team
          </Button>
        ) : disbandStep === 1 ? (
          <div className="flex items-center gap-3 p-3 rounded-[var(--radius-md)] border border-nova-red/30 bg-nova-red/5">
            <span className="text-sm text-text-secondary flex-1">
              This will permanently disband the team.
            </span>
            <Button size="sm" variant="danger" onClick={() => setDisbandStep(2)}>
              Continue
            </Button>
            <Button size="sm" variant="ghost" onClick={() => setDisbandStep(0)}>
              Cancel
            </Button>
          </div>
        ) : (
          <div className="flex items-center gap-3 p-3 rounded-[var(--radius-md)] border border-nova-red bg-nova-red/10">
            <span className="text-sm font-semibold text-nova-red flex-1">
              Are you absolutely sure?
            </span>
            <Button size="sm" variant="danger" loading={disbanding} onClick={handleDisband}>
              Disband forever
            </Button>
            <Button size="sm" variant="ghost" onClick={() => setDisbandStep(0)}>
              Cancel
            </Button>
          </div>
        ))}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------
export function TeamPage() {
  const { data: team, isLoading } = useMyTeam()
  const user = useAuthStore((s) => s.user)
  const isCaptain = !!user?.id && user.id === team?.captain_id
  const frozen = team?.is_banned ?? false
  const isSolo = team?.is_solo ?? false

  if (isLoading) {
    return (
      <div className="flex flex-col gap-6 max-w-2xl mx-auto px-4 py-6 sm:px-6">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-9 w-full" />
        <div className="flex flex-col gap-2">
          {[0, 1, 2].map((i) => (
            <div key={i} className="flex items-center gap-3 px-3 py-2">
              <Skeleton circle className="w-8 h-8" />
              <Skeleton className="h-4 w-32" />
            </div>
          ))}
        </div>
      </div>
    )
  }

  if (!team) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[40vh] text-text-muted">
        <p className="text-sm">You have not selected a participation mode.</p>
      </div>
    )
  }

  const members = Array.isArray(team.members) ? team.members : []

  return (
    <div className="flex flex-col gap-6 max-w-2xl mx-auto px-4 py-6 sm:px-6">
      {/* Team avatar */}
      {(() => {
        const avatarUrl = team.avatar_url ? avatarSrc(team.avatar_url) : null
        return (
          <TeamAvatarSection
            {...(avatarUrl ? { avatarUrl } : {})}
            teamName={team.name ?? ''}
            isCaptain={isCaptain}
            isSolo={isSolo}
          />
        )
      })()}

      {/* Team name */}
      <TeamNameSection name={team.name ?? ''} isCaptain={isCaptain} isSolo={isSolo} />

      {/* Invite link (captain only) */}
      {!isSolo && <InviteLinkSection isCaptain={isCaptain} />}

      {/* Members */}
      <div className="flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <h3 className="text-xs font-semibold text-text-muted uppercase tracking-wide">
            {isSolo ? 'Player' : `Members (${members.length})`}
          </h3>
          {!isSolo && team.min_team_size && team.min_team_size > 0 && (
            <span
              className={`text-xs font-medium ${team.meets_min_size ? 'text-stellar-green' : 'text-nova-red'}`}
            >
              {team.meets_min_size
                ? `✓ Min size met (${team.min_team_size})`
                : `Min ${team.min_team_size} required`}
            </span>
          )}
        </div>
        <div className="rounded-[var(--radius-lg)] border border-space-border bg-space-card overflow-hidden">
          {members.length === 0 ? (
            <EmptyState message="No members yet." />
          ) : (
            <MembersList
              members={members}
              {...(team.captain_id ? { captainId: team.captain_id } : {})}
              {...(user?.id ? { currentUserId: user.id } : {})}
              isCaptain={isCaptain}
              frozen={frozen}
              showTeamActions={!isSolo}
            />
          )}
        </div>
      </div>

      {/* Danger zone */}
      {!isSolo && <DangerZone isCaptain={isCaptain} frozen={frozen} />}
    </div>
  )
}
