import { api, isApiError } from '@/shared/api/client'
import type { components } from '@/shared/api/schema.d'
import { EmptyState } from '@/shared/ui/empty-state'
import { MarkdownRenderer } from '@/shared/ui/markdown-renderer'
import { Skeleton } from '@/shared/ui/skeleton'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { toast } from 'sonner'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------
type Page = components['schemas']['PageResponse']
type CreateBody = components['schemas']['CreatePageRequest']
type UpdateBody = components['schemas']['UpdatePageRequest']

// ---------------------------------------------------------------------------
// Hooks
// ---------------------------------------------------------------------------
function usePages() {
  return useQuery({
    queryKey: ['admin', 'pages'],
    queryFn: async () => {
      const { data, error } = await api.GET('/admin/pages')
      if (error) throw error
      return (data ?? []) as Page[]
    },
    staleTime: 0,
  })
}

function useCreatePage() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (body: CreateBody) => {
      const { data, error } = await api.POST('/admin/pages', { body })
      if (error) throw error
      return data as Page
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'pages'] })
      toast.success('Page created')
    },
    onError: (err) => {
      const msg = isApiError(err) ? err.message : 'Create failed'
      const code = isApiError(err) ? err.code : ''
      toast.error(code === 'PAGE_SLUG_CONFLICT' ? 'Slug already taken' : msg)
    },
  })
}

function useUpdatePage() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async ({ id, body }: { id: string; body: UpdateBody }) => {
      const { data, error } = await api.PUT('/admin/pages/{ID}', {
        params: { path: { ID: id } },
        body,
      })
      if (error) throw error
      return data as Page
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'pages'] })
      toast.success('Page saved')
    },
    onError: (err) => {
      const msg = isApiError(err) ? err.message : 'Save failed'
      const code = isApiError(err) ? err.code : ''
      toast.error(code === 'PAGE_SLUG_CONFLICT' ? 'Slug already taken' : msg)
    },
  })
}

function useDeletePage() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      const { error } = await api.DELETE('/admin/pages/{ID}', {
        params: { path: { ID: id } },
      })
      if (error) throw error
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'pages'] })
      toast.success('Page deleted')
    },
    onError: (err) => toast.error(isApiError(err) ? err.message : 'Delete failed'),
  })
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
const SLUG_RE = /^[a-z0-9]+(?:-[a-z0-9]+)*$/

function slugify(s: string) {
  return s
    .toLowerCase()
    .replace(/[^a-z0-9\s-]/g, '')
    .trim()
    .replace(/\s+/g, '-')
    .replace(/-+/g, '-')
}

// ---------------------------------------------------------------------------
// Editor form
// ---------------------------------------------------------------------------
interface EditorForm {
  title: string
  slug: string
  content: string
  is_draft: boolean
  order_index: string
}

const emptyForm = (): EditorForm => ({
  title: '',
  slug: '',
  content: '',
  is_draft: true,
  order_index: '0',
})

function pageToForm(p: Page): EditorForm {
  return {
    title: p.title ?? '',
    slug: p.slug ?? '',
    content: p.content ?? '',
    is_draft: p.is_draft ?? true,
    order_index: String(p.order_index ?? 0),
  }
}

interface PageEditorProps {
  editing: Page | null
  onClose: () => void
}

function PageEditor({ editing, onClose }: PageEditorProps) {
  const { mutateAsync: create, isPending: creating } = useCreatePage()
  const { mutateAsync: update, isPending: updating } = useUpdatePage()
  const isPending = creating || updating

  const [form, setForm] = useState<EditorForm>(() => (editing ? pageToForm(editing) : emptyForm()))
  const [mode, setMode] = useState<'write' | 'preview'>('write')
  const [slugTouched, setSlugTouched] = useState(!!editing)
  const [slugError, setSlugError] = useState('')

  const setField = <K extends keyof EditorForm>(k: K, v: EditorForm[K]) => {
    setForm((prev) => ({ ...prev, [k]: v }))
  }

  const handleTitleChange = (title: string) => {
    setField('title', title)
    if (!slugTouched) {
      setField('slug', slugify(title))
    }
  }

  const handleSlugChange = (slug: string) => {
    setSlugTouched(true)
    const cleaned = slug.toLowerCase().replace(/[^a-z0-9-]/g, '')
    setField('slug', cleaned)
    setSlugError(
      cleaned && !SLUG_RE.test(cleaned)
        ? 'Slug must be lowercase letters, numbers and hyphens'
        : '',
    )
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!form.title.trim() || !form.slug.trim()) return
    if (!SLUG_RE.test(form.slug)) {
      setSlugError('Invalid slug format')
      return
    }
    try {
      if (editing?.id) {
        const body: UpdateBody = {
          title: form.title.trim(),
          slug: form.slug.trim(),
          is_draft: form.is_draft,
          order_index: parseInt(form.order_index, 10) || 0,
        }
        body.content = form.content
        await update({ id: editing.id, body })
      } else {
        const body: CreateBody = {
          title: form.title.trim(),
          slug: form.slug.trim(),
          is_draft: form.is_draft,
          order_index: parseInt(form.order_index, 10) || 0,
        }
        body.content = form.content
        await create(body)
      }
      onClose()
    } catch {
      // error already toasted
    }
  }

  const textareaCls =
    'w-full rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40 resize-y'

  return (
    <form
      onSubmit={handleSubmit}
      className="flex flex-col gap-4 p-4 rounded-[var(--radius-lg)] border border-cosmic-blue/30 bg-space-card"
    >
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-semibold text-text-primary">
          {editing ? 'Edit page' : 'New page'}
        </h2>
        <button
          type="button"
          onClick={onClose}
          className="text-xs text-text-muted hover:text-text-primary"
        >
          Cancel
        </button>
      </div>

      {/* Title + Slug */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-text-muted">Title *</label>
          <input
            type="text"
            value={form.title}
            onChange={(e) => handleTitleChange(e.target.value)}
            placeholder="Page title"
            required
            className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
          />
        </div>

        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-text-muted">
            Slug * <span className="text-text-muted/60 font-normal">(e.g. about-us)</span>
          </label>
          <input
            type="text"
            value={form.slug}
            onChange={(e) => handleSlugChange(e.target.value)}
            placeholder="page-slug"
            required
            className={`h-9 rounded-[var(--radius-md)] border px-3 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40 bg-space-dark font-mono ${
              slugError ? 'border-nova-red' : 'border-space-border'
            }`}
          />
          {slugError && <p className="text-xs text-nova-red">{slugError}</p>}
        </div>
      </div>

      {/* Order */}
      <div className="flex items-center gap-4">
        <div className="flex flex-col gap-1 w-28">
          <label className="text-xs font-medium text-text-muted">Order</label>
          <input
            type="number"
            value={form.order_index}
            onChange={(e) => setField('order_index', e.target.value)}
            min="0"
            className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
          />
        </div>

        <label className="flex items-center gap-2 cursor-pointer select-none mt-4">
          <input
            type="checkbox"
            checked={form.is_draft}
            onChange={(e) => setField('is_draft', e.target.checked)}
            className="w-4 h-4 rounded accent-cosmic-blue"
          />
          <span className="text-sm text-text-secondary">Draft (not publicly visible)</span>
        </label>
      </div>

      {/* Content editor */}
      <div className="flex flex-col gap-2">
        <div className="flex items-center gap-1 border-b border-space-border pb-1">
          <span className="text-xs font-medium text-text-muted mr-2">Content</span>
          {(['write', 'preview'] as const).map((tab) => (
            <button
              key={tab}
              type="button"
              onClick={() => setMode(tab)}
              className={`px-3 py-1 text-xs rounded-[var(--radius-sm)] capitalize transition-colors ${
                mode === tab
                  ? 'bg-cosmic-blue/20 text-cosmic-blue'
                  : 'text-text-muted hover:text-text-secondary'
              }`}
            >
              {tab}
            </button>
          ))}
        </div>

        {mode === 'write' ? (
          <textarea
            value={form.content}
            onChange={(e) => setField('content', e.target.value)}
            rows={14}
            placeholder="Write Markdown content here…"
            className={textareaCls + ' font-mono'}
          />
        ) : (
          <div className="min-h-[200px] rounded-[var(--radius-md)] border border-space-border bg-space-dark p-4 overflow-y-auto">
            {form.content ? (
              <MarkdownRenderer content={form.content} />
            ) : (
              <p className="text-text-muted text-sm italic">Nothing to preview</p>
            )}
          </div>
        )}
      </div>

      {/* Actions */}
      <div className="flex justify-end gap-3">
        <button
          type="button"
          onClick={onClose}
          className="px-4 py-1.5 text-sm rounded-[var(--radius-md)] border border-space-border text-text-secondary hover:text-text-primary transition-colors"
        >
          Cancel
        </button>
        <button
          type="submit"
          disabled={isPending || !form.title.trim() || !form.slug.trim() || !!slugError}
          className="px-4 py-1.5 text-sm rounded-[var(--radius-md)] bg-cosmic-blue hover:bg-cosmic-blue/90 text-white transition-colors disabled:opacity-50"
        >
          {isPending ? 'Saving…' : editing ? 'Save changes' : 'Create page'}
        </button>
      </div>
    </form>
  )
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------
export function AdminPagesPage() {
  const { data: pages = [], isLoading } = usePages()
  const { mutate: deletePage, isPending: deleting } = useDeletePage()
  const [editing, setEditing] = useState<Page | null | 'new'>(null)

  const sorted = [...pages].sort((a, b) => (a.order_index ?? 0) - (b.order_index ?? 0))

  const handleEdit = (page: Page) => setEditing(page)
  const handleNew = () => setEditing('new')
  const handleClose = () => setEditing(null)

  return (
    <div className="flex flex-col gap-5">
      <div className="flex items-center justify-between">
        <h1
          className="text-2xl font-black text-text-primary"
          style={{ fontFamily: 'var(--font-display)' }}
        >
          Pages
        </h1>
        {editing === null && (
          <button
            onClick={handleNew}
            className="px-4 py-1.5 text-sm rounded-[var(--radius-md)] bg-cosmic-blue hover:bg-cosmic-blue/90 text-white transition-colors"
          >
            + New page
          </button>
        )}
      </div>

      {/* Editor */}
      {editing !== null && (
        <PageEditor
          key={typeof editing === 'object' ? editing.id : editing}
          editing={editing === 'new' ? null : editing}
          onClose={handleClose}
        />
      )}

      {/* List */}
      <div className="overflow-x-auto rounded-[var(--radius-lg)] border border-space-border">
        <table className="w-full min-w-[480px]">
          <thead>
            <tr className="border-b border-space-border bg-space-dark/60">
              <th className="px-3 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
                Title
              </th>
              <th className="px-3 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide">
                Slug
              </th>
              <th className="px-3 py-2.5 text-center text-xs font-semibold text-text-muted uppercase tracking-wide">
                Status
              </th>
              <th className="px-3 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide hidden sm:table-cell">
                Order
              </th>
              <th className="px-3 py-2.5 text-left text-xs font-semibold text-text-muted uppercase tracking-wide hidden md:table-cell">
                Updated
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
                  {Array.from({ length: 6 }).map((__, j) => (
                    <td key={j} className="px-3 py-2.5">
                      <Skeleton className="h-4 w-full" />
                    </td>
                  ))}
                </tr>
              ))
            ) : sorted.length === 0 ? (
              <tr>
                <td colSpan={6}>
                  <EmptyState message="No pages yet. Create your first page." />
                </td>
              </tr>
            ) : (
              sorted.map((page) => (
                <tr
                  key={page.id}
                  data-testid={`admin-row-${page.id}`}
                  className={`border-b border-space-border last:border-b-0 transition-colors ${
                    typeof editing === 'object' && editing !== null && editing.id === page.id
                      ? 'bg-cosmic-blue/5'
                      : 'hover:bg-space-border/20'
                  }`}
                >
                  <td className="px-3 py-2.5 text-sm text-text-primary font-medium">
                    {page.title ?? '-'}
                  </td>
                  <td className="px-3 py-2.5 text-xs font-mono text-nebula-purple">
                    /{page.slug ?? '-'}
                  </td>
                  <td className="px-3 py-2.5 text-center">
                    {page.is_draft ? (
                      <span className="inline-block px-1.5 py-0.5 text-[10px] rounded bg-nova-yellow/20 text-nova-yellow font-semibold">
                        draft
                      </span>
                    ) : (
                      <span className="inline-block px-1.5 py-0.5 text-[10px] rounded bg-nebula-green/20 text-nebula-green font-semibold">
                        published
                      </span>
                    )}
                  </td>
                  <td className="px-3 py-2.5 text-xs text-text-muted hidden sm:table-cell font-mono">
                    {page.order_index ?? 0}
                  </td>
                  <td className="px-3 py-2.5 text-xs text-text-muted hidden md:table-cell whitespace-nowrap">
                    {page.updated_at ? new Date(page.updated_at).toLocaleDateString() : '-'}
                  </td>
                  <td className="px-3 py-2.5 text-right whitespace-nowrap">
                    <button
                      data-testid={`admin-edit-${page.id}`}
                      onClick={() => handleEdit(page)}
                      className="text-xs text-cosmic-blue hover:underline mr-3"
                    >
                      Edit
                    </button>
                    <button
                      data-testid={`admin-delete-${page.id}`}
                      onClick={() => page.id && deletePage(page.id)}
                      disabled={deleting}
                      className="text-xs text-nova-red hover:underline disabled:opacity-40"
                    >
                      Delete
                    </button>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      <p className="text-xs text-text-muted">
        {pages.length} page{pages.length !== 1 ? 's' : ''} total
        {pages.filter((p) => !p.is_draft).length > 0 && (
          <span className="ml-2 text-nebula-green">
            ({pages.filter((p) => !p.is_draft).length} published)
          </span>
        )}
      </p>
    </div>
  )
}
