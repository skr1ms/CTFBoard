import { api, isApiError } from '@/shared/api/client'
import type { components, operations } from '@/shared/api/schema.d'
import { MarkdownRenderer } from '@/shared/ui/markdown-renderer'
import { Modal } from '@/shared/ui/modal'
import { Skeleton } from '@/shared/ui/skeleton'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router'
import { toast } from 'sonner'

type ChallengeRow = components['schemas']['ChallengeResponse']
type TagItem = { id?: string; name?: string; color?: string }
type FileItem = {
  id?: string
  filename?: string
  size?: number
  sha256?: string
  created_at?: string
}
type HintItem = {
  id?: string
  title?: string
  content?: string
  cost?: number
  order_index?: number
}
type ReqItem = { challenge_id?: string; challenge_title?: string; challenge_category?: string }

type HintBody = NonNullable<
  operations['PostAdminChallengesChallengeIDHints']['requestBody']['content']['application/json']
>
type HintUpdateBody = NonNullable<
  operations['PutAdminHintsID']['requestBody']['content']['application/json']
>

// ---------------------------------------------------------------------------
// CSS helpers
// ---------------------------------------------------------------------------
const inputCls =
  'h-9 w-full rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40'

const textareaCls =
  'w-full rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40 resize-y'

function Label({ children }: { children: React.ReactNode }) {
  return (
    <label className="block text-xs font-medium text-text-muted uppercase tracking-wide mb-1">
      {children}
    </label>
  )
}

function formatBytes(n: number) {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / 1024 / 1024).toFixed(1)} MB`
}

// ---------------------------------------------------------------------------
// Tab bar
// ---------------------------------------------------------------------------
type Tab = 'files' | 'hints' | 'flags' | 'requirements' | 'solution' | 'tags' | 'stats'
const TABS: { value: Tab; label: string }[] = [
  { value: 'files', label: 'Files' },
  { value: 'hints', label: 'Hints' },
  { value: 'flags', label: 'Flags' },
  { value: 'requirements', label: 'Requirements' },
  { value: 'solution', label: 'Solution' },
  { value: 'tags', label: 'Tags' },
  { value: 'stats', label: 'Stats' },
]

function TabBar({ active, onChange }: { active: Tab; onChange: (t: Tab) => void }) {
  return (
    <div className="flex gap-1 border-b border-space-border overflow-x-auto scrollbar-none shrink-0">
      {TABS.map((t) => (
        <button
          key={t.value}
          type="button"
          onClick={() => onChange(t.value)}
          className={`px-3 py-2 text-sm whitespace-nowrap border-b-2 -mb-px transition-colors focus-visible:outline-none
            ${
              active === t.value
                ? 'border-cosmic-blue text-cosmic-blue font-medium'
                : 'border-transparent text-text-secondary hover:text-text-primary hover:border-space-border'
            }`}
        >
          {t.label}
        </button>
      ))}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Files tab
// ---------------------------------------------------------------------------
function FilesTab({ challengeId }: { challengeId: string }) {
  const qc = useQueryClient()
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [uploading, setUploading] = useState(false)

  const { data: files = [], isLoading } = useQuery({
    queryKey: ['admin', 'challenge', challengeId, 'files'],
    queryFn: async () => {
      const { data, error } = await api.GET('/challenges/{challengeID}/files', {
        params: { path: { challengeID: challengeId } },
      })
      if (error) throw error
      return (data ?? []) as FileItem[]
    },
    staleTime: 0,
  })

  const { mutateAsync: deleteFile, isPending: deleting } = useMutation({
    mutationFn: async (id: string) => {
      const { error } = await api.DELETE('/admin/files/{ID}', {
        params: { path: { ID: id } },
      })
      if (error) throw error
    },
    onSuccess: () =>
      void qc.invalidateQueries({ queryKey: ['admin', 'challenge', challengeId, 'files'] }),
  })

  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    setUploading(true)
    try {
      const fd = new FormData()
      fd.append('file', file)
      fd.append('type', 'challenge')
      const { error } = await api.POST('/admin/challenges/{challengeID}/files', {
        params: { path: { challengeID: challengeId } },
        body: fd as never,
      })
      if (error) throw error
      await qc.invalidateQueries({ queryKey: ['admin', 'challenge', challengeId, 'files'] })
      toast.success('File uploaded')
    } catch (err) {
      toast.error(isApiError(err) ? err.message : 'Upload failed')
    } finally {
      setUploading(false)
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-2">
        <input
          ref={fileInputRef}
          type="file"
          className="hidden"
          onChange={handleUpload}
          disabled={uploading}
        />
        <button
          type="button"
          disabled={uploading}
          onClick={() => fileInputRef.current?.click()}
          className="px-3 py-1.5 text-sm rounded-[var(--radius-md)] border border-space-border text-text-secondary hover:text-text-primary hover:border-cosmic-blue/50 transition-colors disabled:opacity-50"
        >
          {uploading ? 'Uploading…' : '+ Upload file'}
        </button>
      </div>

      {isLoading ? (
        <div className="flex flex-col gap-2">
          {[1, 2].map((i) => (
            <Skeleton key={i} className="h-10 w-full" />
          ))}
        </div>
      ) : files.length === 0 ? (
        <p className="text-sm text-text-muted">No files attached.</p>
      ) : (
        <div className="rounded-[var(--radius-lg)] border border-space-border overflow-hidden">
          {files.map((f, i) => (
            <div
              key={f.id ?? f.sha256 ?? f.filename ?? i}
              className="flex items-center gap-3 px-4 py-3 border-b border-space-border last:border-b-0"
            >
              <svg
                className="w-4 h-4 text-text-muted shrink-0"
                fill="none"
                stroke="currentColor"
                strokeWidth={1.5}
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M15.75 17.25v3.375c0 .621-.504 1.125-1.125 1.125h-9.75a1.125 1.125 0 0 1-1.125-1.125V7.875c0-.621.504-1.125 1.125-1.125H6.75a9.06 9.06 0 0 1 1.5.124m7.5 10.376h3.375c.621 0 1.125-.504 1.125-1.125V11.25c0-4.46-3.243-8.161-7.5-8.876a9.06 9.06 0 0 0-1.5-.124H9.375c-.621 0-1.125.504-1.125 1.125v3.5m7.5 10.375H9.375a1.125 1.125 0 0 1-1.125-1.125v-9.25m12 6.625v-1.875a3.375 3.375 0 0 0-3.375-3.375h-1.5a1.125 1.125 0 0 1-1.125-1.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H9.75"
                />
              </svg>
              <span className="flex-1 text-sm text-text-primary truncate">{f.filename ?? '-'}</span>
              <span className="text-xs text-text-muted">
                {f.size != null ? formatBytes(f.size) : ''}
              </span>
              <button
                type="button"
                disabled={deleting}
                onClick={() =>
                  deleteFile(f.id ?? '').catch((err) =>
                    toast.error(isApiError(err) ? err.message : 'Delete failed'),
                  )
                }
                className="text-xs text-text-muted hover:text-nova-red transition-colors px-1.5 py-1 rounded hover:bg-nova-red/10 disabled:opacity-40"
              >
                Delete
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Hints tab
// ---------------------------------------------------------------------------
interface HintFormState {
  title: string
  content: string
  cost: number
  orderIndex: number
}

function defaultHintForm(): HintFormState {
  return { title: '', content: '', cost: 0, orderIndex: 0 }
}

function HintsTab({ challengeId }: { challengeId: string }) {
  const qc = useQueryClient()
  const [editing, setEditing] = useState<HintItem | null>(null)
  const [creating, setCreating] = useState(false)
  const [form, setForm] = useState<HintFormState>(defaultHintForm())

  const { data: hints = [], isLoading } = useQuery({
    queryKey: ['admin', 'challenge', challengeId, 'hints'],
    queryFn: async () => {
      const { data, error } = await api.GET('/challenges/{challengeID}/hints', {
        params: { path: { challengeID: challengeId } },
      })
      if (error) throw error
      return (data ?? []) as HintItem[]
    },
    staleTime: 0,
  })

  const { mutateAsync: createHint, isPending: pendingCreate } = useMutation({
    mutationFn: async (body: HintBody) => {
      const { error } = await api.POST('/admin/challenges/{challengeID}/hints', {
        params: { path: { challengeID: challengeId } },
        body,
      })
      if (error) throw error
    },
    onSuccess: () =>
      void qc.invalidateQueries({ queryKey: ['admin', 'challenge', challengeId, 'hints'] }),
  })

  const { mutateAsync: updateHint, isPending: pendingUpdate } = useMutation({
    mutationFn: async ({ id, body }: { id: string; body: HintUpdateBody }) => {
      const { error } = await api.PUT('/admin/hints/{ID}', {
        params: { path: { ID: id } },
        body,
      })
      if (error) throw error
    },
    onSuccess: () =>
      void qc.invalidateQueries({ queryKey: ['admin', 'challenge', challengeId, 'hints'] }),
  })

  const { mutateAsync: deleteHint, isPending: pendingDelete } = useMutation({
    mutationFn: async (id: string) => {
      const { error } = await api.DELETE('/admin/hints/{ID}', {
        params: { path: { ID: id } },
      })
      if (error) throw error
    },
    onSuccess: () =>
      void qc.invalidateQueries({ queryKey: ['admin', 'challenge', challengeId, 'hints'] }),
  })

  const openCreate = () => {
    setEditing(null)
    setForm(defaultHintForm())
    setCreating(true)
  }

  const openEdit = (h: HintItem) => {
    setCreating(false)
    setForm({
      title: h.title ?? '',
      content: h.content ?? '',
      cost: h.cost ?? 0,
      orderIndex: h.order_index ?? 0,
    })
    setEditing(h)
  }

  const closeForm = () => {
    setCreating(false)
    setEditing(null)
  }

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      if (creating) {
        const body: HintBody = { content: form.content }
        body.title = form.title
        if (form.cost !== undefined) body.cost = form.cost
        if (form.orderIndex !== undefined) body.order_index = form.orderIndex
        await createHint(body)
        toast.success('Hint created')
      } else if (editing) {
        const body: HintUpdateBody = { content: form.content }
        body.title = form.title
        if (form.cost !== undefined) body.cost = form.cost
        if (form.orderIndex !== undefined) body.order_index = form.orderIndex
        await updateHint({ id: editing.id ?? '', body })
        toast.success('Hint updated')
      }
      closeForm()
    } catch (err) {
      toast.error(isApiError(err) ? err.message : 'Save failed')
    }
  }

  const isPending = pendingCreate || pendingUpdate

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <span className="text-sm text-text-muted">
          {hints.length} hint{hints.length !== 1 ? 's' : ''}
        </span>
        <button
          type="button"
          onClick={openCreate}
          className="px-3 py-1.5 text-sm rounded-[var(--radius-md)] border border-space-border text-text-secondary hover:text-text-primary hover:border-cosmic-blue/50 transition-colors"
        >
          + Add hint
        </button>
      </div>

      {(creating || editing) && (
        <form
          onSubmit={handleSave}
          className="rounded-[var(--radius-md)] border border-space-border bg-space-dark/40 p-4 flex flex-col gap-3"
        >
          <p className="text-xs font-semibold text-text-muted uppercase tracking-wide">
            {creating ? 'New hint' : 'Edit hint'}
          </p>
          <div>
            <Label>Title (optional)</Label>
            <input
              type="text"
              value={form.title}
              onChange={(e) => setForm((p) => ({ ...p, title: e.target.value }))}
              className={inputCls}
              placeholder="Hint title"
            />
          </div>
          <div>
            <Label>Content *</Label>
            <textarea
              required
              rows={3}
              value={form.content}
              onChange={(e) => setForm((p) => ({ ...p, content: e.target.value }))}
              className={textareaCls}
              placeholder="Hint content…"
            />
          </div>
          <div className="flex gap-3">
            <div>
              <Label>Cost (points)</Label>
              <input
                type="number"
                min={0}
                value={form.cost}
                onChange={(e) => setForm((p) => ({ ...p, cost: Number(e.target.value) }))}
                className="h-9 w-24 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
              />
            </div>
            <div>
              <Label>Order</Label>
              <input
                type="number"
                min={0}
                value={form.orderIndex}
                onChange={(e) => setForm((p) => ({ ...p, orderIndex: Number(e.target.value) }))}
                className="h-9 w-24 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
              />
            </div>
          </div>
          <div className="flex gap-2 justify-end">
            <button
              type="button"
              onClick={closeForm}
              disabled={isPending}
              className="px-3 py-1.5 text-sm text-text-secondary hover:text-text-primary transition-colors disabled:opacity-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isPending}
              className="px-3 py-1.5 text-sm rounded-[var(--radius-md)] bg-cosmic-blue hover:bg-cosmic-blue/90 text-white transition-colors disabled:opacity-50"
            >
              {isPending ? 'Saving…' : 'Save'}
            </button>
          </div>
        </form>
      )}

      {isLoading ? (
        <div className="flex flex-col gap-2">
          {[1, 2].map((i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      ) : hints.length === 0 ? (
        <p className="text-sm text-text-muted">No hints yet.</p>
      ) : (
        <div className="rounded-[var(--radius-lg)] border border-space-border overflow-hidden">
          {[...hints]
            .sort((a, b) => (a.order_index ?? 0) - (b.order_index ?? 0))
            .map((h, i) => (
              <div
                key={h.id ?? h.title ?? i}
                className="flex items-start gap-3 px-4 py-3 border-b border-space-border last:border-b-0"
              >
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-text-primary">
                    {h.title || <span className="text-text-muted italic">Untitled</span>}
                  </p>
                  <p className="text-xs text-text-muted truncate">{h.content}</p>
                </div>
                <span className="text-xs text-text-muted shrink-0">{h.cost ?? 0} pts</span>
                <button
                  type="button"
                  onClick={() => openEdit(h)}
                  className="text-xs text-text-muted hover:text-cosmic-blue transition-colors px-1.5 py-1 rounded hover:bg-cosmic-blue/10"
                >
                  Edit
                </button>
                <button
                  type="button"
                  disabled={pendingDelete}
                  onClick={() =>
                    deleteHint(h.id ?? '').catch((err) =>
                      toast.error(isApiError(err) ? err.message : 'Delete failed'),
                    )
                  }
                  className="text-xs text-text-muted hover:text-nova-red transition-colors px-1.5 py-1 rounded hover:bg-nova-red/10 disabled:opacity-40"
                >
                  Delete
                </button>
              </div>
            ))}
        </div>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Requirements tab
// ---------------------------------------------------------------------------
function RequirementsTab({
  challengeId,
  allChallenges,
}: {
  challengeId: string
  allChallenges: ChallengeRow[]
}) {
  const qc = useQueryClient()

  const { data: reqs = [], isLoading } = useQuery({
    queryKey: ['admin', 'challenge', challengeId, 'requirements'],
    queryFn: async () => {
      const { data, error } = await api.GET('/challenges/{challengeID}/requirements', {
        params: { path: { challengeID: challengeId } },
      })
      if (error) throw error
      return (data ?? []) as ReqItem[]
    },
    staleTime: 0,
  })

  const { mutateAsync: setReqs, isPending } = useMutation({
    mutationFn: async (ids: string[]) => {
      const { error } = await api.PUT('/admin/challenges/{challengeID}/requirements', {
        params: { path: { challengeID: challengeId } },
        body: { requirement_ids: ids },
      })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'challenge', challengeId, 'requirements'] })
      toast.success('Requirements updated')
    },
  })

  const currentIds = reqs.map((r) => r.challenge_id ?? '').filter(Boolean)
  const [selected, setSelected] = useState<string[] | null>(null)
  const displayed = selected ?? currentIds

  // Reset local selection when server data loads
  const [initialized, setInitialized] = useState(false)
  if (!initialized && reqs.length >= 0 && !isLoading) {
    setInitialized(true)
    setSelected(currentIds)
  }

  const toggle = (id: string) => {
    setSelected((prev) => {
      const base = prev ?? currentIds
      if (base.includes(id)) return base.filter((x) => x !== id)
      return [...base, id]
    })
  }

  const handleSave = async () => {
    try {
      await setReqs(displayed)
    } catch (err) {
      toast.error(isApiError(err) ? err.message : 'Save failed')
    }
  }

  const others = allChallenges.filter((c) => c.id !== challengeId)

  return (
    <div className="flex flex-col gap-4">
      <p className="text-xs text-text-muted">
        Select challenges that must be solved before this one is unlocked.
      </p>

      {isLoading ? (
        <div className="flex flex-col gap-2">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-8 w-full" />
          ))}
        </div>
      ) : others.length === 0 ? (
        <p className="text-sm text-text-muted">No other challenges available.</p>
      ) : (
        <div className="flex flex-col gap-1 max-h-72 overflow-y-auto">
          {others.map((c) => {
            const id = c.id ?? ''
            const checked = displayed.includes(id)
            return (
              <label
                key={id}
                className="flex items-center gap-3 px-3 py-2 rounded-[var(--radius-md)] hover:bg-space-border/20 cursor-pointer"
              >
                <input
                  type="checkbox"
                  checked={checked}
                  onChange={() => toggle(id)}
                  className="rounded border-space-border accent-cosmic-blue h-4 w-4"
                />
                <span className="text-sm text-text-primary flex-1">{c.title ?? id}</span>
                <span className="text-xs text-text-muted">{c.category ?? ''}</span>
              </label>
            )
          })}
        </div>
      )}

      <div className="flex justify-end">
        <button
          type="button"
          disabled={isPending}
          onClick={handleSave}
          className="px-3 py-1.5 text-sm rounded-[var(--radius-md)] bg-cosmic-blue hover:bg-cosmic-blue/90 text-white transition-colors disabled:opacity-50"
        >
          {isPending ? 'Saving…' : 'Save requirements'}
        </button>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Solution tab
// ---------------------------------------------------------------------------
function SolutionTab({ challengeId }: { challengeId: string }) {
  const qc = useQueryClient()
  const [previewMode, setPreviewMode] = useState<'write' | 'preview'>('write')

  const { data: solution, isLoading } = useQuery({
    queryKey: ['admin', 'challenge', challengeId, 'solution'],
    queryFn: async () => {
      const { data, error } = await api.GET('/challenges/{challengeID}/solution', {
        params: { path: { challengeID: challengeId } },
      })
      if (error) {
        if (isApiError(error) && error.message?.includes('not found')) return null
        throw error
      }
      return data
    },
    staleTime: 0,
    retry: false,
  })

  const [content, setContent] = useState('')
  const [initialized, setInitialized] = useState(false)
  if (!initialized && !isLoading) {
    setInitialized(true)
    setContent(solution?.content ?? '')
  }

  const { mutateAsync: saveSolution, isPending: saving } = useMutation({
    mutationFn: async (text: string) => {
      const { error } = await api.POST('/admin/challenges/{challengeID}/solution', {
        params: { path: { challengeID: challengeId } },
        body: { content: text },
      })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'challenge', challengeId, 'solution'] })
      toast.success('Solution saved')
    },
  })

  const { mutateAsync: deleteSolution, isPending: deleting } = useMutation({
    mutationFn: async () => {
      const { error } = await api.DELETE('/admin/challenges/{challengeID}/solution', {
        params: { path: { challengeID: challengeId } },
      })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'challenge', challengeId, 'solution'] })
      setContent('')
      setInitialized(false)
      toast.success('Solution deleted')
    },
  })

  if (isLoading) return <Skeleton className="h-40 w-full" />

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <div className="flex gap-1">
          {(['write', 'preview'] as const).map((tab) => (
            <button
              key={tab}
              type="button"
              onClick={() => setPreviewMode(tab)}
              className={`px-2.5 py-0.5 text-xs rounded transition-colors capitalize
                ${previewMode === tab ? 'bg-cosmic-blue/20 text-cosmic-blue' : 'text-text-muted hover:text-text-secondary'}`}
            >
              {tab}
            </button>
          ))}
        </div>
        {solution && (
          <button
            type="button"
            disabled={deleting}
            onClick={() =>
              deleteSolution().catch((err) =>
                toast.error(isApiError(err) ? err.message : 'Delete failed'),
              )
            }
            className="text-xs text-text-muted hover:text-nova-red transition-colors px-1.5 py-1 rounded hover:bg-nova-red/10 disabled:opacity-40"
          >
            Delete solution
          </button>
        )}
      </div>

      {previewMode === 'write' ? (
        <textarea
          rows={8}
          value={content}
          onChange={(e) => setContent(e.target.value)}
          className={textareaCls + ' font-mono'}
          placeholder="Markdown writeup…"
        />
      ) : (
        <div className="min-h-[160px] rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 py-2">
          {content ? (
            <MarkdownRenderer content={content} />
          ) : (
            <p className="text-text-muted text-sm italic">Nothing to preview</p>
          )}
        </div>
      )}

      <div className="flex justify-end">
        <button
          type="button"
          disabled={saving || !content.trim()}
          onClick={() =>
            saveSolution(content).catch((err) =>
              toast.error(isApiError(err) ? err.message : 'Save failed'),
            )
          }
          className="px-3 py-1.5 text-sm rounded-[var(--radius-md)] bg-cosmic-blue hover:bg-cosmic-blue/90 text-white transition-colors disabled:opacity-50"
        >
          {saving ? 'Saving…' : 'Save solution'}
        </button>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Tags tab
// ---------------------------------------------------------------------------
function TagsTab({
  challenge,
  challengeId,
  allTags,
  currentTagIds,
  onTagsChanged,
}: {
  challenge: ChallengeRow
  challengeId: string
  allTags: TagItem[]
  currentTagIds: string[]
  onTagsChanged: () => void
}) {
  const qc = useQueryClient()
  const [selected, setSelected] = useState<string[]>(currentTagIds)
  const [newTagName, setNewTagName] = useState('')
  const [creating, setCreating] = useState(false)

  const { mutateAsync: createTag, isPending: pendingCreate } = useMutation({
    mutationFn: async (name: string) => {
      const { data, error } = await api.POST('/admin/tags', { body: { name, color: '#6b7280' } })
      if (error) throw error
      return data
    },
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['tags'] }),
  })

  const { mutateAsync: updateChallenge, isPending: pendingSave } = useMutation({
    mutationFn: async (tagIds: string[]) => {
      const { error } = await api.PUT('/admin/challenges/{ID}', {
        params: { path: { ID: challengeId } },
        body: {
          title: challenge.title ?? '',
          description: challenge.description ?? '',
          category: challenge.category ?? '',
          points: challenge.points ?? 0,
          tag_ids: tagIds,
        },
      })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'challenges'] })
      void qc.invalidateQueries({ queryKey: ['challenges'] })
      onTagsChanged()
      toast.success('Tags updated')
    },
  })

  const toggle = (id: string) => {
    setSelected((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]))
  }

  const handleCreateTag = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newTagName.trim()) return
    try {
      const tag = await createTag(newTagName.trim())
      if (tag?.id) setSelected((prev) => [...prev, tag.id!])
      setNewTagName('')
      setCreating(false)
      toast.success('Tag created')
    } catch (err) {
      toast.error(isApiError(err) ? err.message : 'Create failed')
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap gap-1.5">
        {allTags.map((tag) => {
          const id = tag.id ?? ''
          const active = selected.includes(id)
          return (
            <button
              key={id}
              type="button"
              onClick={() => toggle(id)}
              className={`px-2.5 py-1 text-xs rounded-full border transition-colors
                ${
                  active
                    ? 'border-cosmic-blue bg-cosmic-blue/15 text-cosmic-blue'
                    : 'border-space-border text-text-secondary hover:border-space-border/80 hover:text-text-primary'
                }`}
            >
              {tag.name ?? id}
            </button>
          )
        })}
        <button
          type="button"
          onClick={() => setCreating((p) => !p)}
          className="px-2.5 py-1 text-xs rounded-full border border-dashed border-space-border text-text-muted hover:text-text-secondary hover:border-cosmic-blue/40 transition-colors"
        >
          + New tag
        </button>
      </div>

      {creating && (
        <form onSubmit={handleCreateTag} className="flex gap-2">
          <input
            type="text"
            value={newTagName}
            onChange={(e) => setNewTagName(e.target.value)}
            placeholder="Tag name"
            className="h-8 flex-1 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
            autoFocus
          />
          <button
            type="submit"
            disabled={pendingCreate}
            className="px-3 text-sm rounded-[var(--radius-md)] bg-cosmic-blue text-white disabled:opacity-50"
          >
            {pendingCreate ? '…' : 'Create'}
          </button>
          <button
            type="button"
            onClick={() => setCreating(false)}
            className="px-3 text-sm text-text-muted hover:text-text-primary"
          >
            Cancel
          </button>
        </form>
      )}

      <div className="flex justify-end">
        <button
          type="button"
          disabled={pendingSave}
          onClick={() =>
            updateChallenge(selected).catch((err) =>
              toast.error(isApiError(err) ? err.message : 'Save failed'),
            )
          }
          className="px-3 py-1.5 text-sm rounded-[var(--radius-md)] bg-cosmic-blue hover:bg-cosmic-blue/90 text-white transition-colors disabled:opacity-50"
        >
          {pendingSave ? 'Saving…' : 'Save tags'}
        </button>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Submission Stats tab
// ---------------------------------------------------------------------------
function SubmissionStatsTab({ challengeId }: { challengeId: string }) {
  const { data, isLoading, isError } = useQuery({
    queryKey: ['admin', 'challenge', challengeId, 'stats'],
    queryFn: async () => {
      const { data, error } = await api.GET('/admin/submissions/challenge/{challengeID}/stats', {
        params: { path: { challengeID: challengeId }, query: { live: true } },
      })
      if (error) throw error
      return data
    },
    staleTime: 0,
    refetchInterval: 30_000,
  })

  if (isLoading) return <Skeleton className="h-32 w-full" />
  if (isError) return <p className="text-sm text-nova-red">Failed to load stats.</p>

  const total = data?.total ?? 0
  const correct = data?.correct ?? 0
  const incorrect = data?.incorrect ?? 0
  const solveRate = total > 0 ? Math.round((correct / total) * 100) : 0

  const stats = [
    { label: 'Total submissions', value: total, color: 'text-text-primary' },
    { label: 'Correct', value: correct, color: 'text-nebula-green' },
    { label: 'Incorrect', value: incorrect, color: 'text-nova-red' },
    { label: 'Solve rate', value: `${solveRate}%`, color: 'text-cosmic-blue' },
  ]

  return (
    <div className="grid grid-cols-2 gap-3">
      {stats.map((s) => (
        <div
          key={s.label}
          className="rounded-[var(--radius-md)] border border-space-border bg-space-dark/40 px-4 py-3"
        >
          <p className="text-xs text-text-muted mb-1">{s.label}</p>
          <p className={`text-2xl font-bold ${s.color}`}>{s.value}</p>
        </div>
      ))}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Flags tab
// ---------------------------------------------------------------------------
function FlagsTab({ challengeId }: { challengeId: string }) {
  const { data, isLoading, isError } = useQuery({
    queryKey: ['admin', 'challenge', challengeId, 'flags'],
    queryFn: async () => {
      const { data, error } = await api.GET('/admin/challenges/{challengeID}/flags', {
        params: { path: { challengeID: challengeId } },
      })
      if (error) throw error
      return data
    },
    staleTime: 0,
  })

  if (isLoading) return <Skeleton className="h-32 w-full" />
  if (isError) return <p className="text-sm text-nova-red">Failed to load flags.</p>

  const flags = data?.flags ?? []

  return (
    <div className="flex flex-col gap-4">
      <div className="flex gap-4 flex-wrap text-sm text-text-muted">
        {data?.is_regex && (
          <span className="px-2 py-0.5 rounded bg-cosmic-blue/10 text-cosmic-blue">Regex</span>
        )}
        {data?.is_case_insensitive && (
          <span className="px-2 py-0.5 rounded bg-space-border text-text-secondary">
            Case-insensitive
          </span>
        )}
        {data?.flag_format_regex && (
          <span className="font-mono text-xs text-text-muted">
            Format: {data.flag_format_regex}
          </span>
        )}
      </div>
      {flags.length === 0 ? (
        <p className="text-sm text-text-muted">No flags configured.</p>
      ) : (
        <div className="rounded-[var(--radius-lg)] border border-space-border overflow-hidden">
          {flags.map((f, i) => (
            <div
              key={i}
              className="flex items-center px-4 py-3 border-b border-space-border last:border-b-0 font-mono text-sm text-text-primary"
            >
              {f}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Challenge Detail Modal
// ---------------------------------------------------------------------------
export function ChallengeDetailModal({
  challenge,
  allChallenges,
  allTags,
  onClose,
  onTagsChanged,
}: {
  challenge: ChallengeRow
  allChallenges: ChallengeRow[]
  allTags: TagItem[]
  onClose: () => void
  onTagsChanged: () => void
}) {
  const [activeTab, setActiveTab] = useState<Tab>('files')
  const challengeId = challenge.id ?? ''

  return (
    <Modal
      open
      onClose={onClose}
      title={`Details: ${challenge.title ?? challengeId}`}
      maxWidth="max-w-2xl"
    >
      <div className="flex flex-col gap-4">
        <TabBar active={activeTab} onChange={setActiveTab} />

        <div className="min-h-[280px]">
          {activeTab === 'files' && <FilesTab challengeId={challengeId} />}
          {activeTab === 'hints' && <HintsTab challengeId={challengeId} />}
          {activeTab === 'flags' && <FlagsTab challengeId={challengeId} />}
          {activeTab === 'requirements' && (
            <RequirementsTab challengeId={challengeId} allChallenges={allChallenges} />
          )}
          {activeTab === 'solution' && <SolutionTab challengeId={challengeId} />}
          {activeTab === 'stats' && <SubmissionStatsTab challengeId={challengeId} />}
          {activeTab === 'tags' && (
            <TagsTab
              challenge={challenge}
              challengeId={challengeId}
              allTags={allTags}
              currentTagIds={(challenge.tags ?? []).map((t) => t.id ?? '').filter(Boolean)}
              onTagsChanged={onTagsChanged}
            />
          )}
        </div>
      </div>
    </Modal>
  )
}

// ---------------------------------------------------------------------------
// Admin Challenge Detail Page
// ---------------------------------------------------------------------------
export function AdminChallengeDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const {
    data: challenge,
    isLoading,
    isError,
  } = useQuery({
    queryKey: ['admin', 'challenge', id, 'detail'],
    queryFn: async () => {
      const { data, error } = await api.GET('/challenges/{challengeID}', {
        params: { path: { challengeID: id! } },
      })
      if (error) throw error
      return data as ChallengeRow
    },
    enabled: !!id,
    staleTime: 0,
  })

  const { data: allChallenges = [] } = useQuery({
    queryKey: ['challenges', 'all'],
    queryFn: async () => {
      const { data, error } = await api.GET('/challenges')
      if (error) throw error
      return (data ?? []) as ChallengeRow[]
    },
    staleTime: 30_000,
  })

  const { data: allTags = [] } = useQuery({
    queryKey: ['tags'],
    queryFn: async () => {
      const { data, error } = await api.GET('/tags')
      if (error) throw error
      return (data ?? []) as TagItem[]
    },
    staleTime: 60_000,
  })

  const [activeTab, setActiveTab] = useState<Tab>('files')
  const qc = useQueryClient()

  const challengeId = id ?? ''

  if (isLoading) {
    return (
      <div className="p-6 flex flex-col gap-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    )
  }

  if (isError || !challenge) {
    return (
      <div className="p-6">
        <p className="text-sm text-nova-red">Challenge not found.</p>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-3 px-6 py-4 border-b border-space-border shrink-0">
        <button
          type="button"
          onClick={() => navigate(-1)}
          className="text-text-muted hover:text-text-primary transition-colors"
          aria-label="Back"
        >
          ←
        </button>
        <h1 className="text-lg font-semibold text-text-primary truncate">
          {challenge.title ?? challengeId}
        </h1>
        <span className="text-xs text-text-muted ml-auto">{challenge.category}</span>
      </div>

      <div className="flex flex-col gap-4 p-6 flex-1 overflow-y-auto">
        <TabBar active={activeTab} onChange={setActiveTab} />

        <div className="flex-1">
          {activeTab === 'files' && <FilesTab challengeId={challengeId} />}
          {activeTab === 'hints' && <HintsTab challengeId={challengeId} />}
          {activeTab === 'flags' && <FlagsTab challengeId={challengeId} />}
          {activeTab === 'requirements' && (
            <RequirementsTab challengeId={challengeId} allChallenges={allChallenges} />
          )}
          {activeTab === 'solution' && <SolutionTab challengeId={challengeId} />}
          {activeTab === 'stats' && <SubmissionStatsTab challengeId={challengeId} />}
          {activeTab === 'tags' && (
            <TagsTab
              challenge={challenge}
              challengeId={challengeId}
              allTags={allTags}
              currentTagIds={(challenge.tags ?? []).map((t) => t.id ?? '').filter(Boolean)}
              onTagsChanged={() =>
                void qc.invalidateQueries({ queryKey: ['admin', 'challenge', id, 'detail'] })
              }
            />
          )}
        </div>
      </div>
    </div>
  )
}
