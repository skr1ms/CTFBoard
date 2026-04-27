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
type Bracket = components['schemas']['BracketResponse']
type CreateBody = components['schemas']['CreateBracketRequest']
type UpdateBody = components['schemas']['UpdateBracketRequest']

// ---------------------------------------------------------------------------
// Hooks
// ---------------------------------------------------------------------------
function useBrackets() {
  return useQuery({
    queryKey: ['admin', 'brackets'],
    queryFn: async () => {
      const { data, error } = await api.GET('/brackets')
      if (error) throw error
      return (data ?? []) as Bracket[]
    },
    staleTime: 0,
  })
}

function useCreateBracket() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (body: CreateBody) => {
      const { error } = await api.POST('/admin/brackets', { body })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'brackets'] })
      toast.success('Bracket created')
    },
    onError: (err) => toast.error(isApiError(err) ? err.message : 'Create failed'),
  })
}

function useUpdateBracket() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async ({ id, body }: { id: string; body: UpdateBody }) => {
      const { error } = await api.PUT('/admin/brackets/{ID}', {
        params: { path: { ID: id } },
        body,
      })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'brackets'] })
      toast.success('Bracket updated')
    },
    onError: (err) => toast.error(isApiError(err) ? err.message : 'Update failed'),
  })
}

function useDeleteBracket() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      const { error } = await api.DELETE('/admin/brackets/{ID}', {
        params: { path: { ID: id } },
      })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'brackets'] })
      toast.success('Bracket deleted')
    },
    onError: (err) => toast.error(isApiError(err) ? err.message : 'Delete failed'),
  })
}

// ---------------------------------------------------------------------------
// Create form
// ---------------------------------------------------------------------------
function CreateBracketForm() {
  const { mutateAsync: create, isPending } = useCreateBracket()
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [isDefault, setIsDefault] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim()) return
    try {
      const body: CreateBody = { name: name.trim(), is_default: isDefault }
      if (description.trim()) body.description = description.trim()
      await create(body)
      setName('')
      setDescription('')
      setIsDefault(false)
    } catch {
      // error already toasted
    }
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="flex flex-col gap-4 p-4 rounded-[var(--radius-lg)] border border-space-border bg-space-card"
    >
      <h2 className="text-sm font-semibold text-text-primary">Create Bracket</h2>
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-text-muted">Name *</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g. Open, University"
            required
            className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
          />
        </div>

        <div className="flex flex-col gap-1">
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

      <div className="flex items-center justify-between">
        <label className="flex items-center gap-2 cursor-pointer select-none">
          <input
            type="checkbox"
            checked={isDefault}
            onChange={(e) => setIsDefault(e.target.checked)}
            className="w-4 h-4 rounded accent-cosmic-blue"
          />
          <span className="text-sm text-text-secondary">Set as default bracket</span>
        </label>

        <button
          type="submit"
          disabled={isPending || !name.trim()}
          className="px-4 py-1.5 text-sm rounded-[var(--radius-md)] bg-cosmic-blue hover:bg-cosmic-blue/90 text-white transition-colors disabled:opacity-50"
        >
          {isPending ? 'Creating…' : 'Create bracket'}
        </button>
      </div>
    </form>
  )
}

// ---------------------------------------------------------------------------
// Edit row
// ---------------------------------------------------------------------------
interface EditBracketRowProps {
  bracket: Bracket
  onCancel: () => void
}

function EditBracketRow({ bracket, onCancel }: EditBracketRowProps) {
  const { mutateAsync: update, isPending } = useUpdateBracket()
  const [name, setName] = useState(bracket.name ?? '')
  const [description, setDescription] = useState(bracket.description ?? '')
  const [isDefault, setIsDefault] = useState(bracket.is_default ?? false)

  const handleSave = async () => {
    if (!bracket.id) return
    try {
      const body: UpdateBody = { name }
      body.description = description.trim()
      body.is_default = isDefault
      await update({ id: bracket.id, body })
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
      <td className="px-3 py-2 hidden sm:table-cell">
        <input
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="description"
          className="w-full h-7 rounded-[var(--radius-sm)] border border-space-border bg-space-dark px-2 text-sm text-text-primary focus:outline-none focus:ring-1 focus:ring-cosmic-blue/40"
        />
      </td>
      <td className="px-3 py-2 text-center">
        <input
          type="checkbox"
          checked={isDefault}
          onChange={(e) => setIsDefault(e.target.checked)}
          className="w-4 h-4 rounded accent-cosmic-blue"
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
export function AdminBracketsPage() {
  const { data: brackets = [], isLoading } = useBrackets()
  const { mutate: deleteBracket, isPending: deleting } = useDeleteBracket()
  const [editingId, setEditingId] = useState<string | null>(null)

  return (
    <div className="flex flex-col gap-5">
      <h1
        className="text-2xl font-black text-text-primary"
        style={{ fontFamily: 'var(--font-display)' }}
      >
        Brackets
      </h1>

      <CreateBracketForm />

      <div className="overflow-x-auto rounded-[var(--radius-lg)] border border-space-border">
        <table className="w-full min-w-[480px]">
          <thead>
            <tr className="border-b border-space-border bg-space-dark/60">
              <th className="px-3 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
                Name
              </th>
              <th className="px-3 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide hidden sm:table-cell">
                Description
              </th>
              <th className="px-3 py-2.5 text-center text-xs font-semibold text-text-muted uppercase tracking-wide">
                Default
              </th>
              <th className="px-3 py-2.5 text-right text-xs font-semibold text-text-muted uppercase tracking-wide">
                Actions
              </th>
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              Array.from({ length: 3 }).map((_, i) => (
                <tr key={i} className="border-b border-space-border">
                  {Array.from({ length: 4 }).map((__, j) => (
                    <td key={j} className="px-3 py-2.5">
                      <Skeleton className="h-4 w-full" />
                    </td>
                  ))}
                </tr>
              ))
            ) : brackets.length === 0 ? (
              <tr>
                <td colSpan={4}>
                  <EmptyState message="No brackets yet." />
                </td>
              </tr>
            ) : (
              brackets.map((bracket) =>
                editingId === bracket.id ? (
                  <EditBracketRow
                    key={bracket.id}
                    bracket={bracket}
                    onCancel={() => setEditingId(null)}
                  />
                ) : (
                  <tr
                    key={bracket.id}
                    data-testid={`admin-row-${bracket.id}`}
                    className="border-b border-space-border last:border-b-0 hover:bg-space-border/20 transition-colors"
                  >
                    <td className="px-3 py-2.5 text-sm text-text-primary font-medium">
                      {bracket.name ?? '-'}
                    </td>
                    <td className="px-3 py-2.5 text-sm text-text-muted hidden sm:table-cell">
                      {bracket.description ?? '-'}
                    </td>
                    <td className="px-3 py-2.5 text-center">
                      {bracket.is_default ? (
                        <span className="inline-block px-1.5 py-0.5 text-[10px] rounded bg-cosmic-blue/20 text-cosmic-blue font-semibold">
                          default
                        </span>
                      ) : (
                        <span className="text-text-muted text-xs">-</span>
                      )}
                    </td>
                    <td className="px-3 py-2.5 text-right whitespace-nowrap">
                      <button
                        data-testid={`admin-edit-${bracket.id}`}
                        onClick={() => setEditingId(bracket.id ?? null)}
                        className="text-xs text-cosmic-blue hover:underline mr-3"
                      >
                        Edit
                      </button>
                      <button
                        data-testid={`admin-delete-${bracket.id}`}
                        onClick={() => bracket.id && deleteBracket(bracket.id)}
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

      <p className="text-xs text-text-muted">
        {brackets.length} bracket{brackets.length !== 1 ? 's' : ''} total
      </p>
    </div>
  )
}
