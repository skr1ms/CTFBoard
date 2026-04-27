import { api, isApiError } from '@/shared/api/client'
import type { components, operations } from '@/shared/api/schema.d'
import { EmptyState } from '@/shared/ui/empty-state'
import { Skeleton } from '@/shared/ui/skeleton'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { toast } from 'sonner'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------
type Tag = NonNullable<
  operations['GetTags']['responses']['200']['content']['application/json']
>[number]

type CreateBody = components['schemas']['CreateTagRequest']
type UpdateBody = components['schemas']['UpdateTagRequest']

// ---------------------------------------------------------------------------
// Hooks
// ---------------------------------------------------------------------------
function useTags() {
  return useQuery({
    queryKey: ['admin', 'tags'],
    queryFn: async () => {
      const { data, error } = await api.GET('/tags')
      if (error) throw error
      return (data ?? []) as Tag[]
    },
    staleTime: 0,
  })
}

function useCreateTag() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (body: CreateBody) => {
      const { error } = await api.POST('/admin/tags', { body })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'tags'] })
      toast.success('Tag created')
    },
    onError: (err) => toast.error(isApiError(err) ? err.message : 'Create failed'),
  })
}

function useUpdateTag() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async ({ id, body }: { id: string; body: UpdateBody }) => {
      const { error } = await api.PUT('/admin/tags/{ID}', {
        params: { path: { ID: id } },
        body,
      })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'tags'] })
      toast.success('Tag updated')
    },
    onError: (err) => toast.error(isApiError(err) ? err.message : 'Update failed'),
  })
}

function useDeleteTag() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      const { error } = await api.DELETE('/admin/tags/{ID}', {
        params: { path: { ID: id } },
      })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'tags'] })
      toast.success('Tag deleted')
    },
    onError: (err) => toast.error(isApiError(err) ? err.message : 'Delete failed'),
  })
}

// ---------------------------------------------------------------------------
// Create form
// ---------------------------------------------------------------------------
function CreateTagForm() {
  const { mutateAsync: create, isPending } = useCreateTag()
  const [name, setName] = useState('')
  const [color, setColor] = useState('#6b7280')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim()) return
    try {
      await create({ name: name.trim(), color })
      setName('')
      setColor('#6b7280')
    } catch {
      // error already toasted
    }
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="flex flex-wrap items-end gap-3 p-4 rounded-[var(--radius-lg)] border border-space-border bg-space-card"
    >
      <div className="flex flex-col gap-1 flex-1 min-w-[160px]">
        <label className="text-xs font-medium text-text-muted">Tag name *</label>
        <input
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="e.g. web, crypto"
          required
          className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
        />
      </div>

      <div className="flex flex-col gap-1">
        <label className="text-xs font-medium text-text-muted">Color</label>
        <div className="flex items-center gap-2">
          <input
            type="color"
            value={color}
            onChange={(e) => setColor(e.target.value)}
            className="h-9 w-14 rounded-[var(--radius-md)] border border-space-border bg-space-dark cursor-pointer p-1"
          />
          <span className="text-xs font-mono text-text-muted">{color}</span>
        </div>
      </div>

      <button
        type="submit"
        disabled={isPending || !name.trim()}
        className="h-9 px-4 text-sm rounded-[var(--radius-md)] bg-cosmic-blue hover:bg-cosmic-blue/90 text-white transition-colors disabled:opacity-50"
      >
        {isPending ? 'Creating…' : 'Add tag'}
      </button>
    </form>
  )
}

// ---------------------------------------------------------------------------
// Edit row
// ---------------------------------------------------------------------------
interface EditTagRowProps {
  tag: Tag
  onCancel: () => void
}

function EditTagRow({ tag, onCancel }: EditTagRowProps) {
  const { mutateAsync: update, isPending } = useUpdateTag()
  const [name, setName] = useState(tag.name ?? '')
  const [color, setColor] = useState(tag.color ?? '#6b7280')

  const handleSave = async () => {
    if (!tag.id) return
    try {
      const body: UpdateBody = { name }
      body.color = color
      await update({ id: tag.id, body })
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
        <div className="flex items-center gap-2">
          <input
            type="color"
            value={color}
            onChange={(e) => setColor(e.target.value)}
            className="h-7 w-10 rounded border border-space-border bg-space-dark cursor-pointer p-0.5"
          />
          <span className="text-xs font-mono text-text-muted">{color}</span>
        </div>
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
export function AdminTagsPage() {
  const { data: tags = [], isLoading } = useTags()
  const { mutate: deleteTag, isPending: deleting } = useDeleteTag()
  const [editingId, setEditingId] = useState<string | null>(null)

  return (
    <div className="flex flex-col gap-5">
      <h1
        className="text-2xl font-black text-text-primary"
        style={{ fontFamily: 'var(--font-display)' }}
      >
        Tags
      </h1>

      <CreateTagForm />

      <div className="overflow-x-auto rounded-[var(--radius-lg)] border border-space-border">
        <table className="w-full min-w-[400px]">
          <thead>
            <tr className="border-b border-space-border bg-space-dark/60">
              <th className="px-3 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
                Name
              </th>
              <th className="px-3 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
                Color
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
                  {Array.from({ length: 3 }).map((__, j) => (
                    <td key={j} className="px-3 py-2.5">
                      <Skeleton className="h-4 w-full" />
                    </td>
                  ))}
                </tr>
              ))
            ) : tags.length === 0 ? (
              <tr>
                <td colSpan={3}>
                  <EmptyState message="No tags yet." />
                </td>
              </tr>
            ) : (
              tags.map((tag) =>
                editingId === tag.id ? (
                  <EditTagRow key={tag.id} tag={tag} onCancel={() => setEditingId(null)} />
                ) : (
                  <tr
                    key={tag.id}
                    data-testid={`admin-row-${tag.id}`}
                    className="border-b border-space-border last:border-b-0 hover:bg-space-border/20 transition-colors"
                  >
                    <td className="px-3 py-2.5 text-sm text-text-primary font-medium">
                      <span className="flex items-center gap-2">
                        <span
                          className="inline-block w-3 h-3 rounded-full shrink-0"
                          style={{ backgroundColor: tag.color ?? '#6b7280' }}
                        />
                        {tag.name ?? '-'}
                      </span>
                    </td>
                    <td className="px-3 py-2.5">
                      <span className="flex items-center gap-2 text-xs font-mono text-text-muted">
                        <span
                          className="inline-block w-5 h-5 rounded border border-space-border shrink-0"
                          style={{ backgroundColor: tag.color ?? '#6b7280' }}
                        />
                        {tag.color ?? '-'}
                      </span>
                    </td>
                    <td className="px-3 py-2.5 text-right whitespace-nowrap">
                      <button
                        data-testid={`admin-edit-${tag.id}`}
                        onClick={() => setEditingId(tag.id ?? null)}
                        className="text-xs text-cosmic-blue hover:underline mr-3"
                      >
                        Edit
                      </button>
                      <button
                        data-testid={`admin-delete-${tag.id}`}
                        onClick={() => tag.id && deleteTag(tag.id)}
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
        {tags.length} tag{tags.length !== 1 ? 's' : ''} total
      </p>
    </div>
  )
}
