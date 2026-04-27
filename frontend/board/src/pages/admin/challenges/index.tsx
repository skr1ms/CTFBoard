import { api, isApiError } from '@/shared/api/client'
import type { components, operations } from '@/shared/api/schema.d'
import { useChallengeTypes } from '@/features/challenges/useChallenges'
import { useDebounce } from '@/shared/lib/hooks/useDebounce'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { EmptyState } from '@/shared/ui/empty-state'
import { MarkdownRenderer } from '@/shared/ui/markdown-renderer'
import { Modal } from '@/shared/ui/modal'
import { Skeleton } from '@/shared/ui/skeleton'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Link } from 'react-router'
import { toast } from 'sonner'
import { ChallengeDetailModal } from './detail'

type ChallengeRow = components['schemas']['ChallengeResponse']
type ChallengeState = 'visible' | 'hidden' | 'locked'
type TagItem = { id?: string; name?: string; color?: string }

type CreateBody = NonNullable<
  operations['PostAdminChallenges']['requestBody']['content']['application/json']
>
type UpdateBody = NonNullable<
  operations['PutAdminChallengesID']['requestBody']['content']['application/json']
>

// ---------------------------------------------------------------------------
// Hooks
// ---------------------------------------------------------------------------
function useAdminChallenges() {
  return useQuery({
    queryKey: ['admin', 'challenges'],
    queryFn: async () => {
      const { data, error } = await api.GET('/challenges')
      if (error) throw error
      return (data ?? []) as ChallengeRow[]
    },
    staleTime: 0,
  })
}

function useTags() {
  return useQuery({
    queryKey: ['tags'],
    queryFn: async () => {
      const { data, error } = await api.GET('/tags')
      if (error) throw error
      return (data ?? []) as TagItem[]
    },
    staleTime: 60_000,
  })
}

function useCreateChallenge() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (body: CreateBody) => {
      const { data, error } = await api.POST('/admin/challenges', { body })
      if (error) throw error
      return data
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'challenges'] })
      void qc.invalidateQueries({ queryKey: ['challenges'] })
    },
  })
}

function useUpdateChallenge() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async ({ id, body }: { id: string; body: UpdateBody }) => {
      const { data, error } = await api.PUT('/admin/challenges/{ID}', {
        params: { path: { ID: id } },
        body,
      })
      if (error) throw error
      return data
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'challenges'] })
      void qc.invalidateQueries({ queryKey: ['challenges'] })
    },
  })
}

function useDeleteChallenge() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      const { error } = await api.DELETE('/admin/challenges/{ID}', {
        params: { path: { ID: id } },
      })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'challenges'] })
      void qc.invalidateQueries({ queryKey: ['challenges'] })
    },
  })
}

function useAdminChallengeFlags(id: string | undefined) {
  return useQuery({
    queryKey: ['admin', 'challenge', id, 'flags'],
    queryFn: async () => {
      const { data, error } = await api.GET('/admin/challenges/{challengeID}/flags', {
        params: { path: { challengeID: id! } },
      })
      if (error) throw error
      return data
    },
    enabled: !!id,
    staleTime: 0,
  })
}

// ---------------------------------------------------------------------------
// State badge
// ---------------------------------------------------------------------------
const STATE_VARIANT: Record<ChallengeState, 'green' | 'default' | 'yellow'> = {
  visible: 'green',
  hidden: 'default',
  locked: 'yellow',
}

function StateBadge({ state }: { state?: ChallengeState | undefined }) {
  const s = state ?? 'hidden'
  return <Badge variant={STATE_VARIANT[s]}>{s}</Badge>
}

// ---------------------------------------------------------------------------
// State filter tabs
// ---------------------------------------------------------------------------
const STATE_FILTERS: Array<ChallengeState | 'all'> = ['all', 'visible', 'hidden', 'locked']

function StateFilter({
  value,
  onChange,
}: {
  value: ChallengeState | 'all'
  onChange: (v: ChallengeState | 'all') => void
}) {
  return (
    <div className="flex gap-1 border-b border-space-border pb-0">
      {STATE_FILTERS.map((f) => (
        <button
          key={f}
          onClick={() => onChange(f)}
          className={`px-3 py-1.5 text-xs font-medium capitalize whitespace-nowrap border-b-2 -mb-px transition-colors focus-visible:outline-none
            ${
              f === value
                ? 'border-cosmic-blue text-cosmic-blue'
                : 'border-transparent text-text-secondary hover:text-text-primary hover:border-space-border'
            }`}
        >
          {f}
        </button>
      ))}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Bulk delete bar
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
        Delete <strong className="text-nova-red">{count}</strong> challenge{count !== 1 ? 's' : ''}?
        This cannot be undone.
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
// Tag multi-select
// ---------------------------------------------------------------------------
function TagSelect({
  tags,
  selected,
  onChange,
}: {
  tags: TagItem[]
  selected: string[]
  onChange: (ids: string[]) => void
}) {
  const toggle = (id: string) => {
    if (selected.includes(id)) {
      onChange(selected.filter((x) => x !== id))
    } else {
      onChange([...selected, id])
    }
  }

  if (tags.length === 0) {
    return <p className="text-xs text-text-muted">No tags available.</p>
  }

  return (
    <div className="flex flex-wrap gap-1.5">
      {tags.map((tag) => {
        const id = tag.id ?? ''
        const active = selected.includes(id)
        return (
          <button
            key={id}
            type="button"
            onClick={() => toggle(id)}
            className={`px-2.5 py-1 text-xs rounded-full border transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-cosmic-blue
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
    </div>
  )
}

// ---------------------------------------------------------------------------
// Challenge form state
// ---------------------------------------------------------------------------
interface FormState {
  title: string
  description: string
  category: string
  points: number
  state: ChallengeState
  connectionInfo: string
  maxAttempts: number
  flag: string
  scoringMode: 'static' | 'dynamic'
  initialValue: number
  minValue: number
  decay: number
  tagIds: string[]
}

// Flag config fields are derived from server data (flagsData) + user overrides,
// so they live outside FormState to avoid effect-based sync.
interface FlagOverrides {
  isRegex?: boolean
  isCaseInsensitive?: boolean
  flagFormatRegex?: string
}

function defaultForm(): FormState {
  return {
    title: '',
    description: '',
    category: '',
    points: 100,
    state: 'hidden',
    connectionInfo: '',
    maxAttempts: 0,
    flag: '',
    scoringMode: 'static',
    initialValue: 500,
    minValue: 100,
    decay: 20,
    tagIds: [],
  }
}

function formFromChallenge(c: ChallengeRow): FormState {
  return {
    title: c.title ?? '',
    description: c.description ?? '',
    category: c.category ?? '',
    points: c.points ?? 100,
    state: (c.state as ChallengeState) ?? 'hidden',
    connectionInfo: c.connection_info ?? '',
    maxAttempts: c.max_attempts ?? 0,
    flag: '',
    scoringMode: 'static',
    initialValue: 500,
    minValue: 100,
    decay: 20,
    tagIds: (c.tags ?? []).map((t) => t.id ?? '').filter(Boolean),
  }
}

// ---------------------------------------------------------------------------
// CSS helpers
// ---------------------------------------------------------------------------
const inputCls =
  'h-9 w-full rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40'

const textareaCls =
  'w-full rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40 resize-y font-mono'

const numberCls =
  'h-9 w-28 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40'

function Label({ children }: { children: React.ReactNode }) {
  return <label className="block text-sm font-medium text-text-secondary mb-1">{children}</label>
}

function FormRow({
  label,
  children,
  hint,
}: {
  label: string
  children: React.ReactNode
  hint?: string
}) {
  return (
    <div>
      <Label>{label}</Label>
      {hint && <p className="text-xs text-text-muted mb-1">{hint}</p>}
      {children}
    </div>
  )
}

function Checkbox({
  checked,
  onChange,
  label,
}: {
  checked: boolean
  onChange: (v: boolean) => void
  label: string
}) {
  return (
    <label className="flex items-center gap-2 cursor-pointer select-none">
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        className="rounded border-space-border accent-cosmic-blue h-4 w-4"
      />
      <span className="text-sm text-text-secondary">{label}</span>
    </label>
  )
}

// ---------------------------------------------------------------------------
// Challenge Form Modal
// ---------------------------------------------------------------------------
type FormMode = { kind: 'create' } | { kind: 'edit'; challenge: ChallengeRow }

function ChallengeFormModal({ mode, onClose }: { mode: FormMode; onClose: () => void }) {
  const { data: tags = [] } = useTags()
  const { data: challengeTypes = [] } = useChallengeTypes()
  const { mutateAsync: createChallenge, isPending: creating } = useCreateChallenge()
  const { mutateAsync: updateChallenge, isPending: updating } = useUpdateChallenge()
  const isPending = creating || updating

  const editId = mode.kind === 'edit' ? (mode.challenge.id ?? undefined) : undefined
  const { data: flagsData, isLoading: isLoadingFlags } = useAdminChallengeFlags(editId)

  const [form, setForm] = useState<FormState>(
    mode.kind === 'edit' ? formFromChallenge(mode.challenge) : defaultForm(),
  )
  const hasFlag = (flagsData?.flags?.length ?? 0) > 0
  const [descTab, setDescTab] = useState<'write' | 'preview'>('write')
  const [error, setError] = useState('')

  // Flag config: derived from server data + user overrides (no effect-based sync needed).
  // User changes are tracked in flagOverrides; unset keys fall back to server defaults.
  const [flagOverrides, setFlagOverrides] = useState<FlagOverrides>({})
  const isRegex = flagOverrides.isRegex ?? flagsData?.is_regex ?? false
  const isCaseInsensitive =
    flagOverrides.isCaseInsensitive ?? flagsData?.is_case_insensitive ?? false
  const flagFormatRegex = flagOverrides.flagFormatRegex ?? flagsData?.flag_format_regex ?? ''

  const set = <K extends keyof FormState>(key: K, val: FormState[K]) =>
    setForm((prev) => ({ ...prev, [key]: val }))

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')

    try {
      if (mode.kind === 'create') {
        const body: CreateBody = {
          title: form.title,
          description: form.description,
          category: form.category,
          points: form.points,
          flag: form.flag,
          state: form.state,
        }
        body.connection_info = form.connectionInfo
        if (form.maxAttempts !== undefined) body.max_attempts = form.maxAttempts
        body.is_regex = isRegex
        body.is_case_insensitive = isCaseInsensitive
        body.flag_format_regex = flagFormatRegex
        body.tag_ids = form.tagIds
        if (form.scoringMode === 'dynamic') {
          body.initial_value = form.initialValue
          body.min_value = form.minValue
          body.decay = form.decay
        }
        await createChallenge(body)
        toast.success('Challenge created')
      } else {
        const body: UpdateBody = {
          title: form.title,
          description: form.description,
          category: form.category,
          points: form.points,
          state: form.state,
        }
        body.connection_info = form.connectionInfo
        if (form.maxAttempts !== undefined) body.max_attempts = form.maxAttempts
        if (form.flag) body.flag = form.flag
        // Include flag config fields only when flagsData hasn't loaded yet or user changed them.
        if (!flagsData || 'isRegex' in flagOverrides) body.is_regex = isRegex
        if (!flagsData || 'isCaseInsensitive' in flagOverrides)
          body.is_case_insensitive = isCaseInsensitive
        if (!flagsData || 'flagFormatRegex' in flagOverrides)
          body.flag_format_regex = flagFormatRegex
        body.tag_ids = form.tagIds
        if (form.scoringMode === 'dynamic') {
          body.initial_value = form.initialValue
          body.min_value = form.minValue
          body.decay = form.decay
        }
        await updateChallenge({ id: mode.challenge.id ?? '', body })
        toast.success('Challenge updated')
      }
      onClose()
    } catch (err) {
      setError(isApiError(err) ? err.message : 'Save failed')
    }
  }

  const title = mode.kind === 'create' ? 'Create Challenge' : `Edit: ${mode.challenge.title ?? ''}`

  return (
    <Modal open onClose={onClose} title={title} maxWidth="max-w-2xl">
      <form onSubmit={handleSubmit} className="flex flex-col gap-5">
        {/* Title + Category row */}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <FormRow label="Title *">
            <input
              type="text"
              required
              value={form.title}
              onChange={(e) => set('title', e.target.value)}
              className={inputCls}
              placeholder="Web Challenge 1"
            />
          </FormRow>
          <FormRow label="Category *">
            <input
              list="challenge-types-list"
              type="text"
              required
              value={form.category}
              onChange={(e) => set('category', e.target.value)}
              className={inputCls}
              placeholder="Web"
            />
            <datalist id="challenge-types-list">
              {challengeTypes.map((t) => (
                <option key={t} value={t} />
              ))}
            </datalist>
          </FormRow>
        </div>

        {/* Description with preview */}
        <div>
          <div className="flex items-center justify-between mb-1">
            <Label>Description *</Label>
            <div className="flex gap-1">
              {(['write', 'preview'] as const).map((tab) => (
                <button
                  key={tab}
                  type="button"
                  onClick={() => setDescTab(tab)}
                  className={`px-2.5 py-0.5 text-xs rounded transition-colors capitalize
                    ${
                      descTab === tab
                        ? 'bg-cosmic-blue/20 text-cosmic-blue'
                        : 'text-text-muted hover:text-text-secondary'
                    }`}
                >
                  {tab}
                </button>
              ))}
            </div>
          </div>
          {descTab === 'write' ? (
            <textarea
              required
              rows={5}
              value={form.description}
              onChange={(e) => set('description', e.target.value)}
              className={textareaCls}
              placeholder="Markdown supported…"
            />
          ) : (
            <div className="min-h-[120px] rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 py-2">
              {form.description ? (
                <MarkdownRenderer content={form.description} />
              ) : (
                <p className="text-text-muted text-sm italic">Nothing to preview</p>
              )}
            </div>
          )}
        </div>

        {/* Scoring mode toggle */}
        <div className="rounded-[var(--radius-md)] border border-space-border bg-space-dark/40 p-4 flex flex-col gap-4">
          <div className="flex items-center gap-4">
            <span className="text-sm font-medium text-text-secondary mr-2">Scoring:</span>
            {(['static', 'dynamic'] as const).map((m) => (
              <label key={m} className="flex items-center gap-1.5 cursor-pointer">
                <input
                  type="radio"
                  name="scoringMode"
                  value={m}
                  checked={form.scoringMode === m}
                  onChange={() => set('scoringMode', m)}
                  className="accent-cosmic-blue"
                />
                <span className="text-sm text-text-secondary capitalize">{m}</span>
              </label>
            ))}
          </div>

          {form.scoringMode === 'static' ? (
            <FormRow label="Points *">
              <input
                type="number"
                required
                min={0}
                value={form.points}
                onChange={(e) => set('points', Number(e.target.value))}
                className={numberCls}
              />
            </FormRow>
          ) : (
            <div className="grid grid-cols-3 gap-3">
              <FormRow label="Initial value" hint="Starting points">
                <input
                  type="number"
                  min={0}
                  value={form.initialValue}
                  onChange={(e) => set('initialValue', Number(e.target.value))}
                  className={numberCls}
                />
              </FormRow>
              <FormRow label="Min value" hint="Floor">
                <input
                  type="number"
                  min={0}
                  value={form.minValue}
                  onChange={(e) => set('minValue', Number(e.target.value))}
                  className={numberCls}
                />
              </FormRow>
              <FormRow label="Decay" hint="Rate of decrease">
                <input
                  type="number"
                  min={1}
                  value={form.decay}
                  onChange={(e) => set('decay', Number(e.target.value))}
                  className={numberCls}
                />
              </FormRow>
            </div>
          )}
        </div>

        {/* State + Connection + Max attempts */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <FormRow label="State">
            <select
              value={form.state}
              onChange={(e) => set('state', e.target.value as ChallengeState)}
              className={inputCls}
            >
              <option value="visible">Visible</option>
              <option value="hidden">Hidden</option>
              <option value="locked">Locked</option>
            </select>
          </FormRow>
          <FormRow label="Max attempts" hint="0 = unlimited">
            <input
              type="number"
              min={0}
              value={form.maxAttempts}
              onChange={(e) => set('maxAttempts', Number(e.target.value))}
              className={numberCls}
            />
          </FormRow>
        </div>

        <FormRow label="Connection info" hint="e.g. nc challenge.ctf 1337">
          <input
            type="text"
            value={form.connectionInfo}
            onChange={(e) => set('connectionInfo', e.target.value)}
            className={inputCls}
            placeholder="nc challenge.ctf 1337"
          />
        </FormRow>

        {/* Flag section */}
        <div className="rounded-[var(--radius-md)] border border-space-border bg-space-dark/40 p-4 flex flex-col gap-3">
          <div className="flex items-center gap-2">
            <p className="text-xs font-semibold text-text-muted uppercase tracking-wide">Flag</p>
            {mode.kind === 'edit' && isLoadingFlags && (
              <span className="text-xs text-text-muted">Loading...</span>
            )}
            {mode.kind === 'edit' && !isLoadingFlags && (
              <span
                className={`text-xs px-2 py-0.5 rounded-full ${
                  hasFlag
                    ? 'bg-stellar-green/10 text-stellar-green border border-stellar-green/30'
                    : 'bg-space-border/30 text-text-muted border border-space-border'
                }`}
              >
                {hasFlag ? 'Flag is set' : 'No flag set'}
              </span>
            )}
          </div>
          <fieldset disabled={mode.kind === 'edit' && isLoadingFlags} className="contents">
            <FormRow
              label={mode.kind === 'create' ? 'Flag *' : 'Flag (leave blank to keep current)'}
            >
              <input
                type="text"
                required={mode.kind === 'create'}
                value={form.flag}
                onChange={(e) => set('flag', e.target.value)}
                className={inputCls}
                placeholder={isRegex ? 'CTF\\{[a-zA-Z0-9_]+\\}' : 'CTF{flag_here}'}
              />
            </FormRow>
            <div className="flex flex-wrap gap-4">
              <Checkbox
                checked={isRegex}
                onChange={(v) => setFlagOverrides((prev) => ({ ...prev, isRegex: v }))}
                label="Is regex"
              />
              <Checkbox
                checked={isCaseInsensitive}
                onChange={(v) => setFlagOverrides((prev) => ({ ...prev, isCaseInsensitive: v }))}
                label="Case insensitive"
              />
            </div>
            <FormRow
              label="Flag format regex"
              hint="Overrides competition default - shown to participants"
            >
              <input
                type="text"
                value={flagFormatRegex}
                onChange={(e) =>
                  setFlagOverrides((prev) => ({ ...prev, flagFormatRegex: e.target.value }))
                }
                className={inputCls}
                placeholder="^CTF\{[a-zA-Z0-9_]+\}$"
              />
            </FormRow>
          </fieldset>
        </div>

        {/* Tags */}
        <FormRow label="Tags">
          <TagSelect tags={tags} selected={form.tagIds} onChange={(ids) => set('tagIds', ids)} />
        </FormRow>

        {error && (
          <p className="text-sm text-nova-red bg-nova-red/10 rounded-[var(--radius-md)] px-4 py-3">
            {error}
          </p>
        )}

        <div className="flex gap-2 justify-end pt-1">
          <Button type="button" variant="ghost" onClick={onClose} disabled={isPending}>
            Cancel
          </Button>
          <Button type="submit" variant="primary" loading={isPending}>
            {mode.kind === 'create' ? 'Create' : 'Save'}
          </Button>
        </div>
      </form>
    </Modal>
  )
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------
export function AdminChallengesPage() {
  const qc = useQueryClient()
  const { data: challenges, isLoading } = useAdminChallenges()
  const { data: tags = [] } = useTags()
  const { mutateAsync: deleteOne, isPending: deleting } = useDeleteChallenge()

  const [search, setSearch] = useState('')
  const debouncedSearch = useDebounce(search, 300)
  const [stateFilter, setStateFilter] = useState<ChallengeState | 'all'>('all')
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [confirmBulk, setConfirmBulk] = useState(false)
  const [formMode, setFormMode] = useState<FormMode | null>(null)
  const [detailChallenge, setDetailChallenge] = useState<ChallengeRow | null>(null)

  // Filter
  const filtered = (challenges ?? []).filter((c) => {
    const matchesState = stateFilter === 'all' || c.state === stateFilter
    const q = debouncedSearch.toLowerCase()
    const matchesSearch =
      !q ||
      (c.title ?? '').toLowerCase().includes(q) ||
      (c.category ?? '').toLowerCase().includes(q)
    return matchesState && matchesSearch
  })

  // Selection helpers
  const allIds = filtered.map((c) => c.id ?? '')
  const allSelected = allIds.length > 0 && allIds.every((id) => selected.has(id))

  const toggleAll = () => {
    if (allSelected) {
      setSelected((prev) => {
        const next = new Set(prev)
        allIds.forEach((id) => next.delete(id))
        return next
      })
    } else {
      setSelected((prev) => new Set([...prev, ...allIds]))
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
        deleteOne(id).catch((err) => {
          failed++
          console.error('Delete failed', id, err)
        }),
      ),
    )
    if (failed === 0) {
      toast.success(`Deleted ${ids.length} challenge${ids.length !== 1 ? 's' : ''}`)
    } else {
      toast.error(`${failed} deletion${failed !== 1 ? 's' : ''} failed`)
    }
    setSelected(new Set())
    setConfirmBulk(false)
  }

  const handleSingleDelete = async (e: React.MouseEvent, id: string, title: string) => {
    e.stopPropagation()
    try {
      await deleteOne(id)
      toast.success(`Deleted "${title}"`)
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
    <div className="flex flex-col gap-4">
      {/* Header */}
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <h1
          className="text-2xl font-black text-text-primary"
          style={{ fontFamily: 'var(--font-display)' }}
        >
          Challenges
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
          <Button variant="primary" size="sm" onClick={() => setFormMode({ kind: 'create' })}>
            + Create Challenge
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

      {/* Search */}
      <input
        type="text"
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        placeholder="Search by name or category…"
        className="h-9 w-full max-w-sm rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
      />

      {/* State filter */}
      <StateFilter value={stateFilter} onChange={setStateFilter} />

      {/* Table */}
      <div className="overflow-x-auto rounded-[var(--radius-lg)] border border-space-border">
        <table className="w-full min-w-[600px] text-sm">
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
                Title
              </th>
              <th className="px-4 py-3 text-left text-xs font-semibold text-text-muted uppercase tracking-wide hidden sm:table-cell">
                Category
              </th>
              <th className="px-4 py-3 text-right text-xs font-semibold text-text-muted uppercase tracking-wide">
                Points
              </th>
              <th className="px-4 py-3 text-center text-xs font-semibold text-text-muted uppercase tracking-wide">
                State
              </th>
              <th className="px-4 py-3 text-right text-xs font-semibold text-text-muted uppercase tracking-wide hidden md:table-cell">
                Solves
              </th>
              <th className="w-16 px-3 py-3" />
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              Array.from({ length: 6 }).map((_, i) => (
                <tr key={i} className="border-b border-space-border last:border-b-0">
                  <td className="px-3 py-3">
                    <Skeleton className="h-4 w-4" />
                  </td>
                  <td className="px-4 py-3">
                    <Skeleton className="h-4 w-40" />
                  </td>
                  <td className="px-4 py-3 hidden sm:table-cell">
                    <Skeleton className="h-4 w-24" />
                  </td>
                  <td className="px-4 py-3 text-right">
                    <Skeleton className="h-4 w-12 ml-auto" />
                  </td>
                  <td className="px-4 py-3 text-center">
                    <Skeleton className="h-5 w-16 mx-auto" />
                  </td>
                  <td className="px-4 py-3 hidden md:table-cell text-right">
                    <Skeleton className="h-4 w-8 ml-auto" />
                  </td>
                  <td className="px-3 py-3" />
                </tr>
              ))
            ) : filtered.length === 0 ? (
              <tr>
                <td colSpan={7}>
                  <EmptyState
                    message={
                      (challenges ?? []).length === 0
                        ? 'No challenges yet.'
                        : 'No challenges match your filters.'
                    }
                  />
                </td>
              </tr>
            ) : (
              filtered.map((c) => {
                const id = c.id ?? ''
                const isSelected = selected.has(id)
                return (
                  <tr
                    key={id}
                    data-testid={`admin-row-${id}`}
                    onClick={() => setFormMode({ kind: 'edit', challenge: c })}
                    className={`border-b border-space-border last:border-b-0 cursor-pointer transition-colors
                      ${isSelected ? 'bg-cosmic-blue/5' : 'hover:bg-space-border/20'}`}
                  >
                    <td className="px-3 py-3">
                      <input
                        type="checkbox"
                        checked={isSelected}
                        onChange={() => toggleOne(id)}
                        onClick={(e) => e.stopPropagation()}
                        className="rounded border-space-border accent-cosmic-blue"
                        aria-label={`Select ${c.title}`}
                      />
                    </td>
                    <td className="px-4 py-3 font-medium text-text-primary truncate max-w-[220px]">
                      {c.title ?? '-'}
                    </td>
                    <td className="px-4 py-3 text-text-secondary hidden sm:table-cell">
                      {c.category ?? '-'}
                    </td>
                    <td className="px-4 py-3 text-right tabular-nums font-mono text-text-secondary">
                      {c.points ?? 0}
                    </td>
                    <td className="px-4 py-3 text-center">
                      <StateBadge state={c.state} />
                    </td>
                    <td className="px-4 py-3 text-right tabular-nums text-text-muted hidden md:table-cell">
                      {c.solve_count ?? 0}
                    </td>
                    <td className="px-3 py-3 text-right">
                      <div className="flex items-center justify-end gap-1">
                        <Link
                          to={`/admin/challenges/${id}`}
                          onClick={(e) => e.stopPropagation()}
                          className="text-xs text-text-muted hover:text-cosmic-blue transition-colors px-1.5 py-1 rounded hover:bg-cosmic-blue/10"
                        >
                          Manage
                        </Link>
                        <button
                          data-testid={`admin-edit-${id}`}
                          onClick={(e) => {
                            e.stopPropagation()
                            setDetailChallenge(c)
                          }}
                          className="text-xs text-text-muted hover:text-cosmic-blue transition-colors px-1.5 py-1 rounded hover:bg-cosmic-blue/10"
                        >
                          Details
                        </button>
                        <button
                          data-testid={`admin-delete-${id}`}
                          onClick={(e) => handleSingleDelete(e, id, c.title ?? 'this challenge')}
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

      {!isLoading && filtered.length > 0 && (
        <p className="text-xs text-text-muted text-right">
          {filtered.length} challenge{filtered.length !== 1 ? 's' : ''}
          {stateFilter !== 'all' || debouncedSearch
            ? ` (filtered from ${(challenges ?? []).length})`
            : ''}
        </p>
      )}

      {/* Create / Edit modal */}
      {formMode && <ChallengeFormModal mode={formMode} onClose={() => setFormMode(null)} />}

      {/* Detail tabs modal */}
      {detailChallenge && (
        <ChallengeDetailModal
          challenge={detailChallenge}
          allChallenges={challenges ?? []}
          allTags={tags}
          onClose={() => setDetailChallenge(null)}
          onTagsChanged={() => void qc.invalidateQueries({ queryKey: ['admin', 'challenges'] })}
        />
      )}
    </div>
  )
}
