import { api, isApiError } from '@/shared/api/client'
import type { operations } from '@/shared/api/schema.d'
import { validateEmail, validatePassword, validateUsername } from '@/shared/lib/validation'
import { avatarSrc } from '@/shared/lib/utils'
import { useAuthStore } from '@/shared/stores/authStore'
import { Avatar } from '@/shared/ui/avatar'
import { Button } from '@/shared/ui/button'
import { FileUpload } from '@/shared/ui/file-upload'
import { Input } from '@/shared/ui/input'
import { Modal } from '@/shared/ui/modal'
import { PasswordInput } from '@/shared/ui/password-input'
import { Skeleton } from '@/shared/ui/skeleton'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useRef, useState } from 'react'
import { toast } from 'sonner'

// ---------------------------------------------------------------------------
// Avatar section
// ---------------------------------------------------------------------------

function AvatarSection() {
  const user = useAuthStore((s) => s.user)
  const setUser = useAuthStore((s) => s.setUser)
  const queryClient = useQueryClient()
  const [preview, setPreview] = useState<string | null>(null)
  const [uploading, setUploading] = useState(false)

  useEffect(() => {
    return () => {
      if (preview) URL.revokeObjectURL(preview)
    }
  }, [preview])

  const handleFile = async (file: File) => {
    setPreview(URL.createObjectURL(file))
    setUploading(true)

    try {
      const form = new FormData()
      form.append('file', file)

      const { error: uploadError } = await api.PUT('/users/me/avatar', { body: form as never })

      if (uploadError) {
        const msg = isApiError(uploadError) ? uploadError.message : 'Avatar upload failed'
        toast.error(msg)
        setPreview(null)
        return
      }

      const { data: me } = await api.GET('/auth/me')
      if (me) setUser(me)
      void queryClient.invalidateQueries({ queryKey: ['me'] })
      toast.success('Avatar updated')
    } catch {
      toast.error('Avatar upload failed')
      setPreview(null)
    } finally {
      setUploading(false)
    }
  }

  const deleteAvatar = useMutation({
    mutationFn: async () => {
      const { error } = await api.DELETE('/users/me/avatar')
      if (error) throw error
    },
    onSuccess: () => {
      if (user) {
        // eslint-disable-next-line @typescript-eslint/no-unused-vars
        const { avatar_url: _, ...rest } = user
        setUser(rest)
      }
      setPreview(null)
      void queryClient.invalidateQueries({ queryKey: ['me'] })
      toast.success('Avatar removed')
    },
    onError: () => toast.error('Failed to remove avatar'),
  })

  const currentUrl = preview ?? avatarSrc(user?.avatar_url) ?? null

  return (
    <div className="flex flex-col gap-4">
      <h2 className="text-sm font-semibold text-text-muted uppercase tracking-wide">Avatar</h2>
      <div className="flex items-start gap-6">
        <Avatar src={currentUrl} {...(user?.username ? { alt: user.username } : {})} size="xl" />
        <div className="flex flex-col gap-3 flex-1">
          <FileUpload
            onFile={handleFile}
            accept="image/jpeg,image/png,image/webp,image/gif"
            maxSizeMb={5}
            label="Drag & drop or click to upload (JPEG, PNG, WebP, GIF static · max 5 MB)"
            disabled={uploading}
          />
          {(user?.avatar_url || preview) && (
            <Button
              data-testid="settings-avatar-delete"
              variant="ghost"
              size="sm"
              onClick={() => deleteAvatar.mutate()}
              disabled={deleteAvatar.isPending || uploading}
              className="w-fit text-text-muted hover:text-red-400"
            >
              Remove avatar
            </Button>
          )}
        </div>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Profile section
// ---------------------------------------------------------------------------

function ProfileSection() {
  const user = useAuthStore((s) => s.user)
  const setUser = useAuthStore((s) => s.setUser)
  const [username, setUsername] = useState(user?.username ?? '')
  const [email, setEmail] = useState(user?.email ?? '')
  const [usernameError, setUsernameError] = useState<string | undefined>()
  const [emailError, setEmailError] = useState<string | undefined>()

  // OAuth-only users have no email in their profile - email field is read-only for them.
  const hasEmail = !!user?.email

  const update = useMutation({
    mutationFn: async () => {
      const body: { username?: string; email?: string } = {}
      if (username !== user?.username) body.username = username
      if (hasEmail && email !== user?.email) body.email = email

      if (Object.keys(body).length === 0) return null

      const { data, error } = await api.PATCH('/auth/me', { body })
      if (error) throw error
      return data
    },
    onSuccess: (data) => {
      if (!data) return
      setUser({ ...user!, ...data })
      toast.success('Profile saved')
    },
    onError: (err) => {
      const msg = isApiError(err) ? err.message : 'Failed to save profile'
      toast.error(msg)
    },
  })

  const isDirty = username !== (user?.username ?? '') || (hasEmail && email !== (user?.email ?? ''))

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    const uErr = validateUsername(username)
    setUsernameError(uErr ?? undefined)
    const eErr = hasEmail ? (validateEmail(email) ?? undefined) : undefined
    setEmailError(eErr)
    if (uErr || eErr) return
    update.mutate()
  }

  return (
    <form className="flex flex-col gap-4" onSubmit={handleSubmit}>
      <h2 className="text-sm font-semibold text-text-muted uppercase tracking-wide">Profile</h2>
      <Input
        data-testid="settings-username"
        label="Username"
        value={username}
        onChange={(e) => {
          setUsername(e.target.value)
          setUsernameError(undefined)
        }}
        autoComplete="username"
        disabled={update.isPending}
        {...(usernameError ? { error: usernameError } : {})}
      />
      <Input
        data-testid="settings-email"
        label="Email"
        type="email"
        value={email}
        onChange={(e) => {
          setEmail(e.target.value)
          setEmailError(undefined)
        }}
        autoComplete="email"
        disabled={update.isPending || !hasEmail}
        {...(!hasEmail ? { hint: 'Email is managed by your OAuth provider' } : {})}
        {...(emailError ? { error: emailError } : {})}
      />
      <div className="flex justify-end">
        <Button
          data-testid="settings-profile-save"
          type="submit"
          variant="primary"
          size="sm"
          disabled={!isDirty}
          loading={update.isPending}
        >
          Save profile
        </Button>
      </div>
    </form>
  )
}

// ---------------------------------------------------------------------------
// Security section
// ---------------------------------------------------------------------------

function SecuritySection() {
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [confirm, setConfirm] = useState('')
  const [nextError, setNextError] = useState<string | undefined>()
  const [confirmError, setConfirmError] = useState<string | undefined>()
  const formRef = useRef<HTMLFormElement>(null)

  const changePassword = useMutation({
    mutationFn: async ({
      currentPassword,
      newPassword,
    }: {
      currentPassword: string
      newPassword: string
    }) => {
      const { data, error } = await api.PATCH('/auth/me', {
        body: { password: newPassword, current_password: currentPassword },
      })
      if (error) throw error
      return data
    },
    onSuccess: (data) => {
      if (data) useAuthStore.getState().setUser(data)
      toast.success('Password changed')
      setCurrent('')
      setNext('')
      setConfirm('')
      setNextError(undefined)
      setConfirmError(undefined)
    },
    onError: (err) => {
      const msg = isApiError(err) ? err.message : 'Failed to change password'
      toast.error(msg)
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    let valid = true
    const pwErr = validatePassword(next)
    if (pwErr) {
      setNextError(pwErr)
      valid = false
    } else setNextError(undefined)
    if (next !== confirm) {
      setConfirmError('Passwords do not match')
      valid = false
    } else setConfirmError(undefined)
    if (!valid) return
    changePassword.mutate({ currentPassword: current, newPassword: next })
  }

  return (
    <form ref={formRef} className="flex flex-col gap-4" onSubmit={handleSubmit}>
      <h2 className="text-sm font-semibold text-text-muted uppercase tracking-wide">Security</h2>
      <PasswordInput
        data-testid="settings-password-current"
        label="Current password"
        value={current}
        onChange={(e) => setCurrent(e.target.value)}
        autoComplete="current-password"
        disabled={changePassword.isPending}
        required
      />
      <PasswordInput
        data-testid="settings-password-new"
        label="New password"
        value={next}
        onChange={(e) => {
          setNext(e.target.value)
          setNextError(undefined)
        }}
        autoComplete="new-password"
        disabled={changePassword.isPending}
        {...(nextError ? { error: nextError } : {})}
        required
      />
      <PasswordInput
        label="Confirm new password"
        value={confirm}
        onChange={(e) => {
          setConfirm(e.target.value)
          setConfirmError(undefined)
        }}
        autoComplete="new-password"
        disabled={changePassword.isPending}
        {...(confirmError ? { error: confirmError } : {})}
        required
      />
      <div className="flex justify-end">
        <Button
          data-testid="settings-password-save"
          type="submit"
          variant="primary"
          size="sm"
          disabled={!current || !next || !confirm}
          loading={changePassword.isPending}
        >
          Change password
        </Button>
      </div>
    </form>
  )
}

// ---------------------------------------------------------------------------
// API Tokens section
// ---------------------------------------------------------------------------
type TokenItem = NonNullable<
  operations['GetUserTokens']['responses']['200']['content']['application/json']
>[number]

type CreateTokenBody = NonNullable<
  operations['PostUserTokens']['requestBody']['content']['application/json']
>

type CreatedToken = NonNullable<
  operations['PostUserTokens']['responses']['201']['content']['application/json']
>

function formatDate(iso?: string) {
  if (!iso) return '-'
  return new Date(iso).toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

function ApiTokensSection() {
  const qc = useQueryClient()

  const { data: tokens = [], isLoading } = useQuery({
    queryKey: ['user', 'tokens'],
    queryFn: async () => {
      const { data, error } = await api.GET('/user/tokens')
      if (error) throw error
      return (data ?? []) as TokenItem[]
    },
    staleTime: 0,
  })

  const { mutateAsync: createToken, isPending: creating } = useMutation({
    mutationFn: async (body: CreateTokenBody) => {
      const { data, error } = await api.POST('/user/tokens', { body })
      if (error) throw error
      return data as CreatedToken
    },
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['user', 'tokens'] }),
  })

  const { mutateAsync: deleteToken, isPending: deleting } = useMutation({
    mutationFn: async (id: string) => {
      const { error } = await api.DELETE('/user/tokens/{ID}', {
        params: { path: { ID: id } },
      })
      if (error) throw error
    },
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['user', 'tokens'] }),
  })

  const [showCreate, setShowCreate] = useState(false)
  const [description, setDescription] = useState('')
  const [expiresAt, setExpiresAt] = useState('')
  const [createdToken, setCreatedToken] = useState<CreatedToken | null>(null)
  const [copied, setCopied] = useState(false)

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      const body: CreateTokenBody = {}
      if (description.trim()) body.description = description.trim()
      if (expiresAt) body.expires_at = new Date(expiresAt).toISOString()
      const result = await createToken(body)
      setCreatedToken(result)
      setShowCreate(false)
      setDescription('')
      setExpiresAt('')
    } catch (err) {
      toast.error(isApiError(err) ? err.message : 'Create failed')
    }
  }

  const handleCopy = async () => {
    if (!createdToken?.token) return
    try {
      await navigator.clipboard.writeText(createdToken.token)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      toast.error('Copy failed')
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await deleteToken(id)
      toast.success('Token revoked')
    } catch (err) {
      toast.error(isApiError(err) ? err.message : 'Delete failed')
    }
  }

  const inputCls =
    'h-9 w-full rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40'

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold text-text-muted uppercase tracking-wide">
          API Tokens
        </h2>
        <Button
          data-testid="settings-token-create"
          size="sm"
          variant="ghost"
          onClick={() => setShowCreate(true)}
        >
          + Create token
        </Button>
      </div>

      {/* Token list */}
      {isLoading ? (
        <div className="flex flex-col gap-2">
          {[1, 2].map((i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      ) : tokens.length === 0 ? (
        <p className="text-sm text-text-muted">No API tokens yet.</p>
      ) : (
        <div className="rounded-[var(--radius-lg)] border border-space-border overflow-hidden">
          {tokens.map((t) => (
            <div
              key={t.id}
              data-testid={`settings-token-row-${t.id}`}
              className="flex items-center gap-3 px-4 py-3 border-b border-space-border last:border-b-0"
            >
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-text-primary truncate">
                  {t.description || <span className="text-text-muted italic">No description</span>}
                </p>
                <p className="text-xs text-text-muted mt-0.5">
                  Created {formatDate(t.created_at)}
                  {t.expires_at ? ` · Expires ${formatDate(t.expires_at)}` : ' · No expiry'}
                  {t.last_used_at ? ` · Last used ${formatDate(t.last_used_at)}` : ''}
                </p>
              </div>
              <button
                onClick={() => handleDelete(t.id ?? '')}
                disabled={deleting}
                className="text-xs text-text-muted hover:text-nova-red transition-colors px-2 py-1 rounded hover:bg-nova-red/10 disabled:opacity-40 shrink-0"
              >
                Revoke
              </button>
            </div>
          ))}
        </div>
      )}

      {/* Create modal */}
      <Modal open={showCreate} onClose={() => setShowCreate(false)} title="Create API Token">
        <form onSubmit={handleCreate} className="flex flex-col gap-4">
          <div>
            <label className="block text-sm font-medium text-text-secondary mb-1">
              Description <span className="text-text-muted font-normal">(optional)</span>
            </label>
            <input
              type="text"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="My CI token"
              className={inputCls}
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-text-secondary mb-1">
              Expires at <span className="text-text-muted font-normal">(optional)</span>
            </label>
            <input
              type="datetime-local"
              value={expiresAt}
              onChange={(e) => setExpiresAt(e.target.value)}
              className={inputCls}
            />
          </div>
          <div className="flex gap-2 justify-end pt-1">
            <Button
              type="button"
              variant="ghost"
              onClick={() => setShowCreate(false)}
              disabled={creating}
            >
              Cancel
            </Button>
            <Button type="submit" variant="primary" loading={creating}>
              Create
            </Button>
          </div>
        </form>
      </Modal>

      {/* Reveal modal - shown once after creation */}
      <Modal
        open={createdToken !== null}
        onClose={() => {
          setCreatedToken(null)
          setCopied(false)
        }}
        title="Token created"
      >
        <div className="flex flex-col gap-4">
          <div className="flex items-start gap-3 rounded-[var(--radius-md)] bg-yellow-500/10 border border-yellow-500/30 px-4 py-3">
            <svg
              className="w-4 h-4 text-yellow-400 mt-0.5 shrink-0"
              fill="currentColor"
              viewBox="0 0 20 20"
            >
              <path
                fillRule="evenodd"
                d="M8.485 2.495c.673-1.167 2.357-1.167 3.03 0l6.28 10.875c.673 1.167-.17 2.625-1.516 2.625H3.72c-1.347 0-2.189-1.458-1.515-2.625L8.485 2.495ZM10 5a.75.75 0 0 1 .75.75v3.5a.75.75 0 0 1-1.5 0v-3.5A.75.75 0 0 1 10 5Zm0 9a1 1 0 1 0 0-2 1 1 0 0 0 0 2Z"
                clipRule="evenodd"
              />
            </svg>
            <p className="text-sm text-yellow-300">
              Copy this token now. You won&apos;t be able to see it again.
            </p>
          </div>

          <div className="flex items-center gap-2">
            <code className="flex-1 bg-space-dark border border-space-border rounded-[var(--radius-md)] px-3 py-2 text-sm font-mono text-cosmic-blue break-all">
              {createdToken?.token}
            </code>
            <button
              onClick={handleCopy}
              className="shrink-0 px-3 py-2 text-sm rounded-[var(--radius-md)] border border-space-border text-text-secondary hover:text-text-primary hover:border-cosmic-blue/50 transition-colors"
            >
              {copied ? '✓ Copied' : 'Copy'}
            </button>
          </div>

          <div className="flex justify-end">
            <Button
              variant="primary"
              onClick={() => {
                setCreatedToken(null)
                setCopied(false)
              }}
            >
              Done
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  )
}

// ---------------------------------------------------------------------------
// SettingsPage
// ---------------------------------------------------------------------------
export function SettingsPage() {
  const user = useAuthStore((s) => s.user)
  // has_password is false only for OAuth-only accounts that have never set a local password.
  // undefined means the backend hasn't returned the field yet (treat as true for safety).
  const showPasswordSection = user?.has_password !== false

  return (
    <div className="flex flex-col gap-8 max-w-2xl mx-auto px-4 py-6 sm:px-6">
      <h1
        className="text-2xl font-black text-text-primary"
        style={{ fontFamily: 'var(--font-display)' }}
      >
        Settings
      </h1>

      <div className="flex flex-col gap-8 divide-y divide-space-border">
        {/* Profile card: avatar + username/email unified */}
        <div className="flex flex-col gap-6">
          <AvatarSection />
          <ProfileSection />
        </div>
        {showPasswordSection && (
          <div className="pt-8">
            <SecuritySection />
          </div>
        )}
        <div className="pt-8">
          <ApiTokensSection />
        </div>
      </div>
    </div>
  )
}
