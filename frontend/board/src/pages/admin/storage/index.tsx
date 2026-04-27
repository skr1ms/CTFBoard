import { api, isApiError } from '@/shared/api/client'
import type { components } from '@/shared/api/schema.d'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { toast } from 'sonner'

type StorageObject = components['schemas']['StorageObjectResponse']

// ---------------------------------------------------------------------------
// Hooks
// ---------------------------------------------------------------------------
function useStorageList(prefix: string) {
  return useQuery({
    queryKey: ['admin', 'storage', prefix],
    queryFn: async () => {
      const { data, error } = await api.GET('/admin/storage', {
        params: { query: prefix ? { prefix } : {} },
      })
      if (error) throw error
      return data
    },
    staleTime: 10_000,
  })
}

function useDeleteStorageObject() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (path: string) => {
      const { error } = await api.DELETE('/admin/storage/{path}', {
        params: { path: { path } },
      })
      if (error) throw error
    },
    onSuccess: (_, path) => {
      toast.success(`Deleted: ${path}`)
      qc.invalidateQueries({ queryKey: ['admin', 'storage'] })
    },
    onError: (err) => toast.error(isApiError(err) ? err.message : 'Delete failed'),
  })
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
function formatSize(bytes?: number): string {
  if (bytes == null) return '-'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(2)} MB`
}

function formatDate(iso?: string): string {
  if (!iso) return '-'
  return new Date(iso).toLocaleString()
}

// ---------------------------------------------------------------------------
// Confirm-delete row
// ---------------------------------------------------------------------------
function StorageRow({
  obj,
  onDelete,
  deleting,
}: {
  obj: StorageObject
  onDelete: (path: string) => void
  deleting: boolean
}) {
  const [confirm, setConfirm] = useState(false)

  return (
    <tr className="border-b border-space-border hover:bg-space-border/20 transition-colors">
      <td className="px-3 py-2 font-mono text-xs text-text-primary max-w-[40vw] truncate">
        {obj.path ?? '-'}
      </td>
      <td className="px-3 py-2 text-xs text-text-secondary whitespace-nowrap">
        {formatSize(obj.size)}
      </td>
      <td className="px-3 py-2 text-xs text-text-secondary whitespace-nowrap">
        {formatDate(obj.last_modified)}
      </td>
      <td className="px-3 py-2 text-right whitespace-nowrap">
        {confirm ? (
          <span className="inline-flex items-center gap-2">
            <button
              onClick={() => {
                if (obj.path) onDelete(obj.path)
              }}
              disabled={deleting}
              className="px-2 py-1 text-xs rounded bg-nova-red text-white hover:bg-nova-red/90 transition-colors disabled:opacity-50"
            >
              {deleting ? '…' : 'Confirm'}
            </button>
            <button
              onClick={() => setConfirm(false)}
              className="px-2 py-1 text-xs rounded border border-space-border text-text-secondary hover:text-text-primary transition-colors"
            >
              Cancel
            </button>
          </span>
        ) : (
          <button
            onClick={() => setConfirm(true)}
            className="px-2 py-1 text-xs rounded border border-nova-red/40 text-nova-red hover:bg-nova-red/10 transition-colors"
          >
            Delete
          </button>
        )}
      </td>
    </tr>
  )
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------
export function AdminStoragePage() {
  const [prefix, setPrefix] = useState('')
  const [inputValue, setInputValue] = useState('')
  const { data, isLoading, isError, error } = useStorageList(prefix)
  const { mutate: deleteObj, isPending: isDeleting } = useDeleteStorageObject()

  const objects = data?.objects ?? []
  const total = data?.total ?? 0

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    setPrefix(inputValue.trim())
  }

  return (
    <div className="flex flex-col gap-5">
      <h1
        className="text-2xl font-black text-text-primary"
        style={{ fontFamily: 'var(--font-display)' }}
      >
        Storage
      </h1>

      {/* Prefix filter */}
      <form onSubmit={handleSearch} className="flex items-center gap-2">
        <input
          type="text"
          value={inputValue}
          onChange={(e) => setInputValue(e.target.value)}
          placeholder="Filter by prefix (e.g. users/)"
          className="h-9 flex-1 max-w-sm rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
        />
        <button
          type="submit"
          className="h-9 px-4 text-sm rounded-[var(--radius-md)] bg-cosmic-blue hover:bg-cosmic-blue/90 text-white transition-colors"
        >
          Search
        </button>
        {prefix && (
          <button
            type="button"
            onClick={() => {
              setPrefix('')
              setInputValue('')
            }}
            className="h-9 px-3 text-sm rounded-[var(--radius-md)] border border-space-border text-text-secondary hover:text-text-primary transition-colors"
          >
            Clear
          </button>
        )}
      </form>

      {/* Table */}
      <div className="rounded-[var(--radius-lg)] border border-space-border bg-space-card overflow-hidden">
        {isLoading && <div className="p-6 text-center text-sm text-text-muted">Loading…</div>}
        {isError && (
          <div className="p-6 text-center text-sm text-nova-red">
            {isApiError(error) ? error.message : 'Failed to load storage objects'}
          </div>
        )}
        {!isLoading && !isError && (
          <>
            <div className="px-3 py-2 border-b border-space-border flex items-center justify-between">
              <span className="text-xs text-text-muted">
                {total} object{total !== 1 ? 's' : ''}
                {prefix ? ` matching "${prefix}"` : ''}
              </span>
            </div>
            {objects.length === 0 ? (
              <div className="p-6 text-center text-sm text-text-muted">No objects found</div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-space-border">
                      <th className="px-3 py-2 text-left text-xs font-medium text-text-muted uppercase tracking-wider">
                        Path
                      </th>
                      <th className="px-3 py-2 text-left text-xs font-medium text-text-muted uppercase tracking-wider">
                        Size
                      </th>
                      <th className="px-3 py-2 text-left text-xs font-medium text-text-muted uppercase tracking-wider">
                        Last Modified
                      </th>
                      <th className="px-3 py-2" />
                    </tr>
                  </thead>
                  <tbody>
                    {objects.map((obj) => (
                      <StorageRow
                        key={obj.path}
                        obj={obj}
                        onDelete={(path) => deleteObj(path)}
                        deleting={isDeleting}
                      />
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}
