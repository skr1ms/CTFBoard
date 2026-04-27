import { api, isApiError } from '@/shared/api/client'
import type { components, operations } from '@/shared/api/schema.d'
import { EmptyState } from '@/shared/ui/empty-state'
import { Skeleton } from '@/shared/ui/skeleton'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { toast } from 'sonner'

// ---------------------------------------------------------------------------
// Visibility Panel
// ---------------------------------------------------------------------------
type VisibilityOption = 'public' | 'private' | 'admins' | 'hidden'

const VISIBILITY_CONFIGS: {
  key: string
  label: string
  description: string
  options: VisibilityOption[]
}[] = [
  {
    key: 'challenge_visibility',
    label: 'Challenges',
    description: 'Who can view the challenge list',
    options: ['public', 'private', 'admins'],
  },
  {
    key: 'score_visibility',
    label: 'Scoreboard',
    description: 'Who can view scores and statistics',
    options: ['public', 'private', 'hidden', 'admins'],
  },
  {
    key: 'account_visibility',
    label: 'Profiles',
    description: 'Who can view user and team profiles',
    options: ['public', 'private', 'admins'],
  },
]

const OPTION_COLORS: Record<VisibilityOption, string> = {
  public: 'text-nebula-green',
  private: 'text-solar-yellow',
  hidden: 'text-text-muted',
  admins: 'text-nova-red',
}

function useVisibilityConfig(key: string) {
  return useQuery({
    queryKey: ['admin', 'visibility-config', key],
    queryFn: async () => {
      const { data, error } = await api.GET('/admin/configs/{key}', {
        params: { path: { key } },
      })
      if (error) throw error
      return (data?.value ?? 'public') as VisibilityOption
    },
    staleTime: 0,
  })
}

function useSetVisibilityConfig() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async ({ key, value }: { key: string; value: VisibilityOption }) => {
      const { error } = await api.PUT('/admin/configs/{key}', {
        params: { path: { key } },
        body: { value, value_type: 'string' },
      })
      if (error) throw error
    },
    onSuccess: (_, { key }) => {
      void qc.invalidateQueries({ queryKey: ['admin', 'visibility-config', key] })
      void qc.invalidateQueries({ queryKey: ['admin', 'configs'] })
      void qc.invalidateQueries({ queryKey: ['configs', 'public'] })
      toast.success('Visibility updated')
    },
    onError: (err) => toast.error(isApiError(err) ? err.message : 'Save failed'),
  })
}

function VisibilitySelect({
  key: configKey,
  label,
  description,
  options,
}: (typeof VISIBILITY_CONFIGS)[number]) {
  const { data: current, isLoading } = useVisibilityConfig(configKey)
  const { mutate: save, isPending } = useSetVisibilityConfig()

  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm font-medium text-text-primary">{label}</p>
          <p className="text-xs text-text-muted">{description}</p>
        </div>
        {isLoading ? (
          <div className="h-8 w-28 rounded-[var(--radius-md)] bg-space-border/40 animate-pulse" />
        ) : (
          <select
            value={current ?? 'public'}
            disabled={isPending}
            onChange={(e) => save({ key: configKey, value: e.target.value as VisibilityOption })}
            className={`h-8 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-2 text-sm font-medium focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40 disabled:opacity-50 ${OPTION_COLORS[current ?? 'public']}`}
          >
            {options.map((opt) => (
              <option key={opt} value={opt} className="text-text-primary">
                {opt}
              </option>
            ))}
          </select>
        )}
      </div>
    </div>
  )
}

function VisibilityPanel() {
  return (
    <div className="flex flex-col gap-3 p-4 rounded-[var(--radius-lg)] border border-space-border bg-space-card">
      <div className="flex items-center gap-2 mb-1">
        <svg
          className="w-4 h-4 text-cosmic-blue shrink-0"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
          />
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
          />
        </svg>
        <h2 className="text-sm font-semibold text-text-primary">Visibility</h2>
        <span className="text-xs text-text-muted ml-1">
          - controls what guests and users can see
        </span>
      </div>
      <div className="flex flex-col divide-y divide-space-border/50">
        {VISIBILITY_CONFIGS.map((cfg) => (
          <div key={cfg.key} className="py-2 first:pt-0 last:pb-0">
            <VisibilitySelect {...cfg} />
          </div>
        ))}
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------
type Config = NonNullable<
  operations['GetAdminConfigs']['responses']['200']['content']['application/json']
>[number]

type CategoryItem = NonNullable<
  operations['GetAdminConfigsCategories']['responses']['200']['content']['application/json']
>[number]

type SetBody = NonNullable<
  operations['PutAdminConfigsKey']['requestBody']
>['content']['application/json']

type BatchItem = components['schemas']['BatchSetConfigItem']

type ValueType = NonNullable<SetBody['value_type']>

const VALUE_TYPES = ['string', 'int', 'bool', 'json'] as const
function toValueType(s: string): ValueType {
  return (VALUE_TYPES as readonly string[]).includes(s) ? (s as ValueType) : 'string'
}

// ---------------------------------------------------------------------------
// Hooks
// ---------------------------------------------------------------------------
function useConfigs(category: string) {
  return useQuery({
    queryKey: ['admin', 'configs', category],
    queryFn: async () => {
      if (category) {
        const { data, error } = await api.GET('/admin/configs/category/{category}', {
          params: { path: { category } },
        })
        if (error) throw error
        return (data ?? []) as Config[]
      }
      const { data, error } = await api.GET('/admin/configs')
      if (error) throw error
      return (data ?? []) as Config[]
    },
    staleTime: 0,
  })
}

function useCategories() {
  return useQuery({
    queryKey: ['admin', 'configs-categories'],
    queryFn: async () => {
      const { data, error } = await api.GET('/admin/configs/categories')
      if (error) throw error
      return (data ?? []) as CategoryItem[]
    },
    staleTime: 30_000,
  })
}

function useSetConfig() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async ({ key, body }: { key: string; body: SetBody }) => {
      const { error } = await api.PUT('/admin/configs/{key}', {
        params: { path: { key } },
        body,
      })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'configs'] })
      void qc.invalidateQueries({ queryKey: ['admin', 'configs-categories'] })
      toast.success('Config saved')
    },
    onError: (err) => toast.error(isApiError(err) ? err.message : 'Save failed'),
  })
}

function useDeleteConfig() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (key: string) => {
      const { error } = await api.DELETE('/admin/configs/{key}', {
        params: { path: { key } },
      })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'configs'] })
      void qc.invalidateQueries({ queryKey: ['admin', 'configs-categories'] })
      toast.success('Config deleted')
    },
    onError: (err) => toast.error(isApiError(err) ? err.message : 'Delete failed'),
  })
}

function useBatchUpdate() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (configs: BatchItem[]) => {
      const { error } = await api.PUT('/admin/configs/batch', { body: { configs } })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'configs'] })
      void qc.invalidateQueries({ queryKey: ['admin', 'configs-categories'] })
      toast.success('Batch saved')
    },
    onError: (err) => toast.error(isApiError(err) ? err.message : 'Batch save failed'),
  })
}

// ---------------------------------------------------------------------------
// Add config form
// ---------------------------------------------------------------------------
function AddConfigForm() {
  const { mutateAsync: setConfig, isPending } = useSetConfig()
  const [key, setKey] = useState('')
  const [value, setValue] = useState('')
  const [valueType, setValueType] = useState<ValueType>('string')
  const [category, setCategory] = useState('')
  const [description, setDescription] = useState('')
  const [open, setOpen] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!key.trim() || !value.trim()) return
    try {
      const body: SetBody = { value: value.trim(), value_type: valueType }
      if (category.trim()) body.category = category.trim()
      if (description.trim()) body.description = description.trim()
      await setConfig({ key: key.trim(), body })
      setKey('')
      setValue('')
      setCategory('')
      setDescription('')
      setValueType('string')
      setOpen(false)
    } catch {
      // error already toasted
    }
  }

  if (!open) {
    return (
      <button
        onClick={() => setOpen(true)}
        className="self-start px-4 py-1.5 text-sm rounded-[var(--radius-md)] bg-cosmic-blue hover:bg-cosmic-blue/90 text-white transition-colors"
      >
        + Add config
      </button>
    )
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="flex flex-col gap-4 p-4 rounded-[var(--radius-lg)] border border-space-border bg-space-card"
    >
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold text-text-primary">Add / Update Config</h2>
        <button
          type="button"
          onClick={() => setOpen(false)}
          className="text-xs text-text-muted hover:text-text-primary"
        >
          Cancel
        </button>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-text-muted">Key *</label>
          <input
            type="text"
            value={key}
            onChange={(e) => setKey(e.target.value)}
            placeholder="e.g. max_team_size"
            required
            className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40 font-mono"
          />
        </div>

        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-text-muted">Value *</label>
          <input
            type="text"
            value={value}
            onChange={(e) => setValue(e.target.value)}
            placeholder="value"
            required
            className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
          />
        </div>

        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-text-muted">Type</label>
          <select
            value={valueType}
            onChange={(e) => setValueType(e.target.value as ValueType)}
            className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-2 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
          >
            <option value="string">string</option>
            <option value="int">int</option>
            <option value="bool">bool</option>
            <option value="json">json</option>
          </select>
        </div>

        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-text-muted">Category</label>
          <input
            type="text"
            value={category}
            onChange={(e) => setCategory(e.target.value)}
            placeholder="e.g. competition"
            className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
          />
        </div>

        <div className="flex flex-col gap-1 sm:col-span-2">
          <label className="text-xs font-medium text-text-muted">Description</label>
          <input
            type="text"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Optional description"
            className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
          />
        </div>
      </div>

      <div className="flex justify-end">
        <button
          type="submit"
          disabled={isPending || !key.trim() || !value.trim()}
          className="px-4 py-1.5 text-sm rounded-[var(--radius-md)] bg-cosmic-blue hover:bg-cosmic-blue/90 text-white transition-colors disabled:opacity-50"
        >
          {isPending ? 'Saving…' : 'Save config'}
        </button>
      </div>
    </form>
  )
}

// ---------------------------------------------------------------------------
// Inline edit row
// ---------------------------------------------------------------------------
interface EditRowProps {
  config: Config
  onCancel: () => void
}

function EditRow({ config, onCancel }: EditRowProps) {
  const { mutateAsync: setConfig, isPending } = useSetConfig()
  const [value, setValue] = useState(config.value)
  const [valueType, setValueType] = useState<ValueType>(toValueType(config.value_type))
  const [description, setDescription] = useState(config.description ?? '')
  const [category, setCategory] = useState(config.category ?? '')

  const handleSave = async () => {
    if (!config.key) return
    try {
      const body: SetBody = { value, value_type: valueType }
      body.category = category.trim()
      body.description = description.trim()
      await setConfig({ key: config.key, body })
      onCancel()
    } catch {
      // error already toasted
    }
  }

  return (
    <tr className="border-b border-space-border bg-space-card/60">
      <td className="px-3 py-2 text-sm font-mono text-nebula-purple">{config.key}</td>
      <td className="px-3 py-2">
        <input
          value={value}
          onChange={(e) => setValue(e.target.value)}
          className="w-full h-7 rounded-[var(--radius-sm)] border border-space-border bg-space-dark px-2 text-sm text-text-primary focus:outline-none focus:ring-1 focus:ring-cosmic-blue/40"
        />
      </td>
      <td className="px-3 py-2">
        <select
          value={valueType}
          onChange={(e) => setValueType(e.target.value as ValueType)}
          className="h-7 rounded-[var(--radius-sm)] border border-space-border bg-space-dark px-1 text-xs text-text-primary focus:outline-none"
        >
          <option value="string">string</option>
          <option value="int">int</option>
          <option value="bool">bool</option>
          <option value="json">json</option>
        </select>
      </td>
      <td className="px-3 py-2 hidden lg:table-cell">
        <input
          value={category}
          onChange={(e) => setCategory(e.target.value)}
          placeholder="category"
          className="w-full h-7 rounded-[var(--radius-sm)] border border-space-border bg-space-dark px-2 text-sm text-text-primary focus:outline-none focus:ring-1 focus:ring-cosmic-blue/40"
        />
      </td>
      <td className="px-3 py-2 hidden md:table-cell">
        <input
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="description"
          className="w-full h-7 rounded-[var(--radius-sm)] border border-space-border bg-space-dark px-2 text-sm text-text-primary focus:outline-none focus:ring-1 focus:ring-cosmic-blue/40"
        />
      </td>
      <td className="px-3 py-2 text-right whitespace-nowrap">
        <button
          onClick={handleSave}
          disabled={isPending}
          className="text-xs text-nebula-green hover:underline disabled:opacity-40 mr-3"
        >
          {isPending ? 'Saving…' : 'Save'}
        </button>
        <button onClick={onCancel} className="text-xs text-text-muted hover:text-text-primary">
          Cancel
        </button>
      </td>
    </tr>
  )
}

// ---------------------------------------------------------------------------
// Batch editor
// ---------------------------------------------------------------------------
interface BatchEditorProps {
  configs: Config[]
  onClose: () => void
}

function BatchEditor({ configs, onClose }: BatchEditorProps) {
  const { mutateAsync: batchUpdate, isPending } = useBatchUpdate()
  const [edits, setEdits] = useState<Record<string, string>>(() =>
    Object.fromEntries(configs.map((c) => [c.key, c.value])),
  )

  const handleSave = async () => {
    const items: BatchItem[] = configs
      .filter((c) => c.key && edits[c.key] !== c.value)
      .slice(0, 50)
      .map((c) => {
        const item: BatchItem = {
          key: c.key,
          value: edits[c.key] ?? c.value,
          value_type: toValueType(c.value_type),
        }
        if (c.category) item.category = c.category
        if (c.description) item.description = c.description
        return item
      })

    if (items.length === 0) {
      toast.info('No changes to save')
      return
    }

    try {
      await batchUpdate(items)
      onClose()
    } catch {
      // error already toasted
    }
  }

  const changed = configs.filter((c) => c.key && edits[c.key] !== c.value)

  return (
    <div className="flex flex-col gap-4 p-4 rounded-[var(--radius-lg)] border border-space-border bg-space-card">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-sm font-semibold text-text-primary">Batch Edit</h2>
          <p className="text-xs text-text-muted mt-0.5">
            Edit up to 50 configs at once. Only changed values will be saved.
          </p>
        </div>
        <button onClick={onClose} className="text-xs text-text-muted hover:text-text-primary">
          Cancel
        </button>
      </div>

      <div className="flex flex-col gap-2 max-h-80 overflow-y-auto pr-1">
        {configs.map((c) => (
          <div key={c.key} className="flex items-center gap-2">
            <span
              className="w-48 shrink-0 text-xs font-mono text-nebula-purple truncate"
              title={c.key}
            >
              {c.key}
            </span>
            <input
              value={edits[c.key] ?? c.value}
              onChange={(e) => setEdits((prev) => ({ ...prev, [c.key]: e.target.value }))}
              className="flex-1 h-7 rounded-[var(--radius-sm)] border border-space-border bg-space-dark px-2 text-sm text-text-primary focus:outline-none focus:ring-1 focus:ring-cosmic-blue/40"
            />
            <span className="w-12 shrink-0 text-xs text-text-muted text-right">{c.value_type}</span>
          </div>
        ))}
      </div>

      <div className="flex items-center justify-between">
        <span className="text-xs text-text-muted">
          {changed.length} change{changed.length !== 1 ? 's' : ''}
          {changed.length > 50 && <span className="text-nova-yellow ml-1">(max 50 per batch)</span>}
        </span>
        <button
          onClick={handleSave}
          disabled={isPending || changed.length === 0}
          className="px-4 py-1.5 text-sm rounded-[var(--radius-md)] bg-cosmic-blue hover:bg-cosmic-blue/90 text-white transition-colors disabled:opacity-50"
        >
          {isPending
            ? 'Saving…'
            : `Save ${Math.min(changed.length, 50)} change${Math.min(changed.length, 50) !== 1 ? 's' : ''}`}
        </button>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Row skeleton
// ---------------------------------------------------------------------------
function RowSkeleton() {
  return (
    <>
      {Array.from({ length: 8 }).map((_, i) => (
        <tr key={i} className="border-b border-space-border">
          {Array.from({ length: 6 }).map((__, j) => (
            <td key={j} className="px-3 py-2.5">
              <Skeleton className="h-4 w-full" />
            </td>
          ))}
        </tr>
      ))}
    </>
  )
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------
export function AdminConfigsPage() {
  const [filterCategory, setFilterCategory] = useState('')
  const [search, setSearch] = useState('')
  const [editingKey, setEditingKey] = useState<string | null>(null)
  const [batchOpen, setBatchOpen] = useState(false)
  const [prevFilterCategory, setPrevFilterCategory] = useState(filterCategory)

  const { data: configs = [], isLoading } = useConfigs(filterCategory)
  const { data: categories = [] } = useCategories()
  const { mutate: deleteConfig, isPending: deleting } = useDeleteConfig()

  if (prevFilterCategory !== filterCategory) {
    setPrevFilterCategory(filterCategory)
    setEditingKey(null)
    setBatchOpen(false)
  }

  const displayed = search.trim()
    ? configs.filter(
        (c) =>
          c.key.toLowerCase().includes(search.toLowerCase()) ||
          (c.value ?? '').toLowerCase().includes(search.toLowerCase()) ||
          (c.description ?? '').toLowerCase().includes(search.toLowerCase()),
      )
    : configs

  return (
    <div className="flex flex-col gap-5">
      <h1
        className="text-2xl font-black text-text-primary"
        style={{ fontFamily: 'var(--font-display)' }}
      >
        Configs
      </h1>

      {/* Visibility quick-settings */}
      <VisibilityPanel />

      {/* Add form */}
      <AddConfigForm />

      {/* Batch editor */}
      {batchOpen && (
        <BatchEditor configs={displayed.slice(0, 50)} onClose={() => setBatchOpen(false)} />
      )}

      {/* Filters */}
      <div className="flex flex-wrap items-center gap-3">
        {/* Category filter */}
        <select
          value={filterCategory}
          onChange={(e) => setFilterCategory(e.target.value)}
          className="h-8 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-2 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
        >
          <option value="">All categories</option>
          {categories.map((cat) => (
            <option key={cat.name} value={cat.name}>
              {cat.name} ({cat.count})
            </option>
          ))}
        </select>

        {/* Search */}
        <input
          type="search"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search key / value…"
          className="h-8 w-56 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
        />

        {(filterCategory || search) && (
          <button
            onClick={() => {
              setFilterCategory('')
              setSearch('')
            }}
            className="text-xs text-text-muted hover:text-text-primary transition-colors"
          >
            Clear
          </button>
        )}

        <span className="ml-auto text-xs text-text-muted">
          {displayed.length} config{displayed.length !== 1 ? 's' : ''}
        </span>

        {!batchOpen && displayed.length > 0 && (
          <button
            onClick={() => setBatchOpen(true)}
            className="px-3 py-1 text-xs rounded-[var(--radius-md)] border border-space-border text-text-secondary hover:text-text-primary hover:border-cosmic-blue/40 transition-colors"
          >
            Batch edit
          </button>
        )}
      </div>

      {/* Table */}
      <div className="overflow-x-auto rounded-[var(--radius-lg)] border border-space-border">
        <table className="w-full min-w-[680px]">
          <thead>
            <tr className="border-b border-space-border bg-space-dark/60">
              <th className="px-3 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
                Key
              </th>
              <th className="px-3 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
                Value
              </th>
              <th className="px-3 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
                Type
              </th>
              <th className="px-3 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide hidden lg:table-cell">
                Category
              </th>
              <th className="px-3 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide hidden md:table-cell">
                Description
              </th>
              <th className="px-3 py-2.5 text-right text-xs font-semibold text-text-muted uppercase tracking-wide">
                Actions
              </th>
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              <RowSkeleton />
            ) : displayed.length === 0 ? (
              <tr>
                <td colSpan={6}>
                  <EmptyState
                    message={
                      configs.length === 0 ? 'No configs yet.' : 'No configs match your search.'
                    }
                  />
                </td>
              </tr>
            ) : (
              displayed.map((config) =>
                editingKey === config.key ? (
                  <EditRow key={config.key} config={config} onCancel={() => setEditingKey(null)} />
                ) : (
                  <tr
                    key={config.key}
                    className="border-b border-space-border last:border-b-0 hover:bg-space-border/20 transition-colors"
                  >
                    <td className="px-3 py-2.5 text-sm font-mono text-nebula-purple max-w-[200px]">
                      <span className="truncate block" title={config.key}>
                        {config.key}
                      </span>
                    </td>
                    <td className="px-3 py-2.5 text-sm text-text-primary max-w-[220px]">
                      <span className="truncate block font-mono text-xs" title={config.value}>
                        {config.value}
                      </span>
                    </td>
                    <td className="px-3 py-2.5">
                      <span className="inline-block px-1.5 py-0.5 text-[10px] rounded bg-space-border/60 text-text-muted font-mono">
                        {config.value_type}
                      </span>
                    </td>
                    <td className="px-3 py-2.5 text-xs text-text-muted hidden lg:table-cell">
                      {config.category ?? '-'}
                    </td>
                    <td className="px-3 py-2.5 text-xs text-text-muted hidden md:table-cell max-w-[200px]">
                      <span className="truncate block" title={config.description ?? ''}>
                        {config.description ?? '-'}
                      </span>
                    </td>
                    <td className="px-3 py-2.5 text-right whitespace-nowrap">
                      <button
                        onClick={() => setEditingKey(config.key)}
                        className="text-xs text-cosmic-blue hover:underline mr-3"
                      >
                        Edit
                      </button>
                      <button
                        onClick={() => deleteConfig(config.key)}
                        disabled={deleting}
                        className="text-xs text-nova-red hover:underline disabled:opacity-40"
                      >
                        Delete
                      </button>
                    </td>
                  </tr>
                ),
              )
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}
