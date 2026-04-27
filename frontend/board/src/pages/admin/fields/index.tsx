import { api, isApiError } from '@/shared/api/client'
import type { components } from '@/shared/api/schema.d'
import { EmptyState } from '@/shared/ui/empty-state'
import { Skeleton } from '@/shared/ui/skeleton'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { toast } from 'sonner'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------
type Field = components['schemas']['FieldResponse']
type CreateBody = components['schemas']['CreateFieldRequest']
type UpdateBody = components['schemas']['UpdateFieldRequest']
type FieldType = NonNullable<Field['field_type']>
type EntityType = NonNullable<Field['entity_type']>

// ---------------------------------------------------------------------------
// Hooks
// ---------------------------------------------------------------------------
function useFields() {
  return useQuery({
    queryKey: ['admin', 'fields'],
    queryFn: async () => {
      const [userRes, teamRes] = await Promise.all([
        api.GET('/fields', { params: { query: { entity_type: 'user' } } }),
        api.GET('/fields', { params: { query: { entity_type: 'team' } } }),
      ])
      if (userRes.error) throw userRes.error
      if (teamRes.error) throw teamRes.error
      return [...((userRes.data ?? []) as Field[]), ...((teamRes.data ?? []) as Field[])]
    },
    staleTime: 0,
  })
}

function useCreateField() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (body: CreateBody) => {
      const { error } = await api.POST('/admin/fields', { body })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'fields'] })
      toast.success('Field created')
    },
    onError: (err) => toast.error(isApiError(err) ? err.message : 'Create failed'),
  })
}

function useUpdateField() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async ({ id, body }: { id: string; body: UpdateBody }) => {
      const { error } = await api.PUT('/admin/fields/{ID}', {
        params: { path: { ID: id } },
        body,
      })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'fields'] })
      toast.success('Field updated')
    },
    onError: (err) => toast.error(isApiError(err) ? err.message : 'Update failed'),
  })
}

function useDeleteField() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      const { error } = await api.DELETE('/admin/fields/{ID}', {
        params: { path: { ID: id } },
      })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'fields'] })
      toast.success('Field deleted')
    },
    onError: (err) => toast.error(isApiError(err) ? err.message : 'Delete failed'),
  })
}

// ---------------------------------------------------------------------------
// Options editor (comma-separated for select type)
// ---------------------------------------------------------------------------
function OptionsInput({
  value,
  onChange,
}: {
  value: string[]
  onChange: (opts: string[]) => void
}) {
  const [raw, setRaw] = useState(value.join(', '))

  const handleBlur = () => {
    const opts = raw
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean)
    onChange(opts)
    setRaw(opts.join(', '))
  }

  return (
    <div className="flex flex-col gap-1">
      <label className="text-xs font-medium text-text-muted">
        Options <span className="text-text-muted/60">(comma-separated)</span>
      </label>
      <input
        type="text"
        value={raw}
        onChange={(e) => setRaw(e.target.value)}
        onBlur={handleBlur}
        placeholder="opt1, opt2, opt3"
        className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
      />
    </div>
  )
}

// ---------------------------------------------------------------------------
// Create form
// ---------------------------------------------------------------------------
function CreateFieldForm() {
  const { mutateAsync: create, isPending } = useCreateField()
  const [name, setName] = useState('')
  const [fieldType, setFieldType] = useState<FieldType>('text')
  const [entityType, setEntityType] = useState<EntityType>('user')
  const [required, setRequired] = useState(false)
  const [options, setOptions] = useState<string[]>([])
  const [orderIndex, setOrderIndex] = useState('0')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim()) return
    try {
      const body: CreateBody = {
        name: name.trim(),
        field_type: fieldType,
        entity_type: entityType,
        required,
        order_index: parseInt(orderIndex, 10) || 0,
      }
      if (fieldType === 'select' && options.length > 0) body.options = options
      await create(body)
      setName('')
      setFieldType('text')
      setEntityType('user')
      setRequired(false)
      setOptions([])
      setOrderIndex('0')
    } catch {
      // error already toasted
    }
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="flex flex-col gap-4 p-4 rounded-[var(--radius-lg)] border border-space-border bg-space-card"
    >
      <h2 className="text-sm font-semibold text-text-primary">Create Custom Field</h2>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-text-muted">Name *</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g. Discord handle"
            required
            className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
          />
        </div>

        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-text-muted">Field type</label>
          <select
            value={fieldType}
            onChange={(e) => setFieldType(e.target.value as FieldType)}
            className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-2 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
          >
            <option value="text">text</option>
            <option value="number">number</option>
            <option value="boolean">boolean</option>
            <option value="select">select</option>
          </select>
        </div>

        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-text-muted">Entity type</label>
          <select
            value={entityType}
            onChange={(e) => setEntityType(e.target.value as EntityType)}
            className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-2 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
          >
            <option value="user">user</option>
            <option value="team">team</option>
          </select>
        </div>

        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-text-muted">Order</label>
          <input
            type="number"
            value={orderIndex}
            onChange={(e) => setOrderIndex(e.target.value)}
            min="0"
            className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
          />
        </div>

        {fieldType === 'select' && (
          <div className="sm:col-span-2">
            <OptionsInput value={options} onChange={setOptions} />
          </div>
        )}
      </div>

      <div className="flex items-center justify-between">
        <label className="flex items-center gap-2 cursor-pointer select-none">
          <input
            type="checkbox"
            checked={required}
            onChange={(e) => setRequired(e.target.checked)}
            className="w-4 h-4 rounded accent-cosmic-blue"
          />
          <span className="text-sm text-text-secondary">Required field</span>
        </label>

        <button
          type="submit"
          disabled={isPending || !name.trim()}
          className="px-4 py-1.5 text-sm rounded-[var(--radius-md)] bg-cosmic-blue hover:bg-cosmic-blue/90 text-white transition-colors disabled:opacity-50"
        >
          {isPending ? 'Creating…' : 'Create field'}
        </button>
      </div>
    </form>
  )
}

// ---------------------------------------------------------------------------
// Edit row
// ---------------------------------------------------------------------------
interface EditFieldRowProps {
  field: Field
  onCancel: () => void
}

function EditFieldRow({ field, onCancel }: EditFieldRowProps) {
  const { mutateAsync: update, isPending } = useUpdateField()
  const [name, setName] = useState(field.name ?? '')
  const [fieldType, setFieldType] = useState<FieldType>(field.field_type ?? 'text')
  const [required, setRequired] = useState(field.required ?? false)
  const [options, setOptions] = useState<string[]>(field.options ?? [])
  const [orderIndex, setOrderIndex] = useState(String(field.order_index ?? 0))

  const handleSave = async () => {
    if (!field.id) return
    try {
      const body: UpdateBody = {
        name,
        field_type: fieldType,
        required,
        order_index: parseInt(orderIndex, 10) || 0,
      }
      if (fieldType === 'select') body.options = options
      await update({ id: field.id, body })
      onCancel()
    } catch {
      // error already toasted
    }
  }

  return (
    <tr className="border-b border-space-border bg-space-card/60">
      <td className="px-3 py-2">
        <input
          value={name}
          onChange={(e) => setName(e.target.value)}
          className="w-full h-7 rounded-[var(--radius-sm)] border border-space-border bg-space-dark px-2 text-sm text-text-primary focus:outline-none focus:ring-1 focus:ring-cosmic-blue/40"
        />
      </td>
      <td className="px-3 py-2">
        <select
          value={fieldType}
          onChange={(e) => setFieldType(e.target.value as FieldType)}
          className="h-7 rounded-[var(--radius-sm)] border border-space-border bg-space-dark px-1 text-xs text-text-primary focus:outline-none"
        >
          <option value="text">text</option>
          <option value="number">number</option>
          <option value="boolean">boolean</option>
          <option value="select">select</option>
        </select>
      </td>
      <td className="px-3 py-2 hidden md:table-cell text-xs text-text-muted">
        {field.entity_type ?? '-'}
      </td>
      <td className="px-3 py-2 text-center">
        <input
          type="checkbox"
          checked={required}
          onChange={(e) => setRequired(e.target.checked)}
          className="w-4 h-4 rounded accent-cosmic-blue"
        />
      </td>
      <td className="px-3 py-2 hidden lg:table-cell">
        {fieldType === 'select' ? (
          <input
            type="text"
            value={options.join(', ')}
            onChange={(e) =>
              setOptions(
                e.target.value
                  .split(',')
                  .map((s) => s.trim())
                  .filter(Boolean),
              )
            }
            placeholder="opt1, opt2"
            className="w-full h-7 rounded-[var(--radius-sm)] border border-space-border bg-space-dark px-2 text-xs text-text-primary focus:outline-none focus:ring-1 focus:ring-cosmic-blue/40"
          />
        ) : (
          <span className="text-xs text-text-muted">-</span>
        )}
      </td>
      <td className="px-3 py-2 hidden sm:table-cell">
        <input
          type="number"
          value={orderIndex}
          onChange={(e) => setOrderIndex(e.target.value)}
          min="0"
          className="w-16 h-7 rounded-[var(--radius-sm)] border border-space-border bg-space-dark px-2 text-sm text-text-primary focus:outline-none focus:ring-1 focus:ring-cosmic-blue/40"
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
// Page
// ---------------------------------------------------------------------------
export function AdminFieldsPage() {
  const { data: fields = [], isLoading } = useFields()
  const { mutate: deleteField, isPending: deleting } = useDeleteField()
  const [editingId, setEditingId] = useState<string | null>(null)
  const [filterEntity, setFilterEntity] = useState<'' | 'user' | 'team'>('')

  const displayed = filterEntity ? fields.filter((f) => f.entity_type === filterEntity) : fields
  const sorted = [...displayed].sort((a, b) => (a.order_index ?? 0) - (b.order_index ?? 0))

  return (
    <div className="flex flex-col gap-5">
      <h1
        className="text-2xl font-black text-text-primary"
        style={{ fontFamily: 'var(--font-display)' }}
      >
        Custom Fields
      </h1>

      <CreateFieldForm />

      {/* Filter */}
      <div className="flex items-center gap-3">
        <label className="text-xs font-medium text-text-muted">Entity:</label>
        {(['', 'user', 'team'] as const).map((v) => (
          <button
            key={v}
            onClick={() => setFilterEntity(v)}
            className={`px-3 py-1 text-xs rounded-[var(--radius-md)] border transition-colors ${
              filterEntity === v
                ? 'border-cosmic-blue bg-cosmic-blue/10 text-cosmic-blue'
                : 'border-space-border text-text-muted hover:text-text-primary'
            }`}
          >
            {v === '' ? 'All' : v}
          </button>
        ))}
        <span className="ml-auto text-xs text-text-muted">
          {sorted.length} field{sorted.length !== 1 ? 's' : ''}
        </span>
      </div>

      <div className="overflow-x-auto rounded-[var(--radius-lg)] border border-space-border">
        <table className="w-full min-w-[600px]">
          <thead>
            <tr className="border-b border-space-border bg-space-dark/60">
              <th className="px-3 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
                Name
              </th>
              <th className="px-3 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
                Type
              </th>
              <th className="px-3 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide hidden md:table-cell">
                Entity
              </th>
              <th className="px-3 py-2.5 text-center text-xs font-semibold text-text-muted uppercase tracking-wide">
                Req
              </th>
              <th className="px-3 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide hidden lg:table-cell">
                Options
              </th>
              <th className="px-3 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide hidden sm:table-cell">
                Order
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
                  {Array.from({ length: 7 }).map((__, j) => (
                    <td key={j} className="px-3 py-2.5">
                      <Skeleton className="h-4 w-full" />
                    </td>
                  ))}
                </tr>
              ))
            ) : sorted.length === 0 ? (
              <tr>
                <td colSpan={7}>
                  <EmptyState
                    message={
                      fields.length === 0
                        ? 'No custom fields yet.'
                        : 'No fields for this entity type.'
                    }
                  />
                </td>
              </tr>
            ) : (
              sorted.map((field) =>
                editingId === field.id ? (
                  <EditFieldRow key={field.id} field={field} onCancel={() => setEditingId(null)} />
                ) : (
                  <tr
                    key={field.id}
                    data-testid={`admin-row-${field.id}`}
                    className="border-b border-space-border last:border-b-0 hover:bg-space-border/20 transition-colors"
                  >
                    <td className="px-3 py-2.5 text-sm text-text-primary font-medium">
                      {field.name ?? '-'}
                    </td>
                    <td className="px-3 py-2.5">
                      <span className="inline-block px-1.5 py-0.5 text-[10px] rounded bg-space-border/60 text-text-muted font-mono">
                        {field.field_type ?? '-'}
                      </span>
                    </td>
                    <td className="px-3 py-2.5 text-xs text-text-muted hidden md:table-cell">
                      <span
                        className={`inline-block px-1.5 py-0.5 rounded text-[10px] font-medium ${
                          field.entity_type === 'team'
                            ? 'bg-nebula-purple/20 text-nebula-purple'
                            : 'bg-cosmic-blue/20 text-cosmic-blue'
                        }`}
                      >
                        {field.entity_type ?? '-'}
                      </span>
                    </td>
                    <td className="px-3 py-2.5 text-center text-xs">
                      {field.required ? (
                        <span className="text-nova-red font-bold">✓</span>
                      ) : (
                        <span className="text-text-muted">-</span>
                      )}
                    </td>
                    <td className="px-3 py-2.5 text-xs text-text-muted hidden lg:table-cell max-w-[160px]">
                      {field.field_type === 'select' &&
                      field.options &&
                      field.options.length > 0 ? (
                        <span className="truncate block" title={field.options.join(', ')}>
                          {field.options.join(', ')}
                        </span>
                      ) : (
                        '-'
                      )}
                    </td>
                    <td className="px-3 py-2.5 text-xs text-text-muted hidden sm:table-cell font-mono">
                      {field.order_index ?? 0}
                    </td>
                    <td className="px-3 py-2.5 text-right whitespace-nowrap">
                      <button
                        data-testid={`admin-edit-${field.id}`}
                        onClick={() => setEditingId(field.id ?? null)}
                        className="text-xs text-cosmic-blue hover:underline mr-3"
                      >
                        Edit
                      </button>
                      <button
                        data-testid={`admin-delete-${field.id}`}
                        onClick={() => field.id && deleteField(field.id)}
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
