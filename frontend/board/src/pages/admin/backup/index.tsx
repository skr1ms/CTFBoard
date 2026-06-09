import { api, isApiError, refreshTokens } from '@/shared/api/client'
import type { components } from '@/shared/api/schema.d'
import { env } from '@/shared/config/env'
import { useAuthStore } from '@/shared/stores/authStore'
import { useMutation } from '@tanstack/react-query'
import { useRef, useState } from 'react'
import { toast } from 'sonner'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------
type ResetBody = components['schemas']['AdminResetRequest']
type ImportResult = components['schemas']['ImportResult']
type CSVImportResult = components['schemas']['CSVImportResult']
type CsvTable = 'users' | 'teams' | 'challenges' | 'submissions' | 'solves' | 'awards'

const CSV_TABLES: CsvTable[] = ['users', 'teams', 'challenges', 'submissions', 'solves', 'awards']

// ---------------------------------------------------------------------------
// Download helper - blob fetch with one automatic 401->refresh retry
// ---------------------------------------------------------------------------
async function downloadBlob(path: string, filename: string): Promise<void> {
  const doFetch = () => {
    const token = useAuthStore.getState().accessToken ?? ''
    return fetch(`${env.apiBaseUrl}${path}`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
  }

  let res = await doFetch()

  if (res.status === 401) {
    const ok = await refreshTokens()
    if (!ok) throw new Error('Session expired. Please log in again.')
    res = await doFetch()
  }

  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as { message?: string }
    throw new Error(body.message ?? `Export failed (${res.status})`)
  }
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

// ---------------------------------------------------------------------------
// Hooks
// ---------------------------------------------------------------------------
function useReset() {
  return useMutation({
    mutationFn: async (body: ResetBody) => {
      const { error } = await api.POST('/admin/reset', { body })
      if (error) throw error
    },
    onSuccess: () => toast.success('Reset completed'),
    onError: (err) => toast.error(isApiError(err) ? err.message : 'Reset failed'),
  })
}

function useImportZip() {
  return useMutation({
    mutationFn: async ({
      file,
      eraseExisting,
      conflictMode,
    }: {
      file: File
      eraseExisting: boolean
      conflictMode: 'overwrite'
    }): Promise<ImportResult> => {
      const fd = new FormData()
      fd.append('file', file)
      fd.append('erase_existing', String(eraseExisting))
      fd.append('conflict_mode', conflictMode)
      const { data, error } = await api.POST('/admin/import', { body: fd as never })
      if (error) throw error
      return data as ImportResult
    },
    onSuccess: (result) => {
      if (result.success) {
        toast.success('Import completed')
      } else {
        toast.warning(`Import finished with errors: ${(result.errors ?? []).join(', ')}`)
      }
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : 'Import failed'),
  })
}

function useImportCsv() {
  return useMutation({
    mutationFn: async ({
      file,
      table,
    }: {
      file: File
      table: CsvTable
    }): Promise<CSVImportResult> => {
      const fd = new FormData()
      fd.append('file', file)
      fd.append('table', table)
      const { data, error } = await api.POST('/admin/import/csv', { body: fd as never })
      if (error) throw error
      return data as CSVImportResult
    },
    onSuccess: (result) => {
      toast.success(`CSV imported: ${result.imported_count ?? 0} rows`)
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : 'CSV import failed'),
  })
}

// ---------------------------------------------------------------------------
// Section: Export
// ---------------------------------------------------------------------------
function ExportSection() {
  const [exporting, setExporting] = useState<string | null>(null)
  const [csvTable, setCsvTable] = useState<CsvTable>('users')

  const doExport = async (path: string, filename: string) => {
    setExporting(filename)
    try {
      await downloadBlob(path, filename)
      toast.success(`${filename} downloaded`)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Export failed')
    } finally {
      setExporting(null)
    }
  }

  return (
    <div className="flex flex-col gap-4 p-4 rounded-[var(--radius-lg)] border border-space-border bg-space-card">
      <h2 className="text-sm font-semibold text-text-primary">Export</h2>

      <div className="flex flex-wrap gap-3">
        <button
          onClick={() => doExport('/admin/export', `backup-${Date.now()}.json`)}
          disabled={!!exporting}
          className="px-4 py-2 text-sm rounded-[var(--radius-md)] border border-space-border text-text-secondary hover:text-text-primary hover:border-cosmic-blue/40 transition-colors disabled:opacity-50"
        >
          {exporting ? 'Exporting…' : 'Export JSON'}
        </button>

        <button
          onClick={() => doExport('/admin/export/zip', `backup-${Date.now()}.zip`)}
          disabled={!!exporting}
          className="px-4 py-2 text-sm rounded-[var(--radius-md)] border border-space-border text-text-secondary hover:text-text-primary hover:border-cosmic-blue/40 transition-colors disabled:opacity-50"
        >
          Export ZIP
        </button>
      </div>

      <div className="flex items-end gap-3">
        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-text-muted">CSV table</label>
          <select
            value={csvTable}
            onChange={(e) => setCsvTable(e.target.value as CsvTable)}
            className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-2 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
          >
            {CSV_TABLES.map((t) => (
              <option key={t} value={t}>
                {t}
              </option>
            ))}
          </select>
        </div>
        <button
          onClick={() => doExport(`/admin/export/csv?table=${csvTable}`, `${csvTable}.csv`)}
          disabled={!!exporting}
          className="h-9 px-4 text-sm rounded-[var(--radius-md)] border border-space-border text-text-secondary hover:text-text-primary hover:border-cosmic-blue/40 transition-colors disabled:opacity-50"
        >
          Export CSV
        </button>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Section: Import ZIP
// ---------------------------------------------------------------------------
function ImportZipSection() {
  const { mutateAsync: importZip, isPending } = useImportZip()
  const [dragging, setDragging] = useState(false)
  const [file, setFile] = useState<File | null>(null)
  const [eraseExisting, setEraseExisting] = useState(false)
  const [conflictMode, setConflictMode] = useState<'overwrite'>('overwrite')
  const [result, setResult] = useState<ImportResult | null>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    setDragging(false)
    const dropped = e.dataTransfer.files[0]
    if (dropped?.name.endsWith('.zip')) setFile(dropped)
    else toast.error('Please drop a .zip file')
  }

  const handleImport = async () => {
    if (!file) return
    try {
      const res = await importZip({ file, eraseExisting, conflictMode })
      setResult(res)
      setFile(null)
    } catch {
      // error already toasted
    }
  }

  return (
    <div className="flex flex-col gap-4 p-4 rounded-[var(--radius-lg)] border border-space-border bg-space-card">
      <h2 className="text-sm font-semibold text-text-primary">Import ZIP</h2>

      {/* Drop zone */}
      <div
        onDragOver={(e) => {
          e.preventDefault()
          setDragging(true)
        }}
        onDragLeave={() => setDragging(false)}
        onDrop={handleDrop}
        onClick={() => inputRef.current?.click()}
        className={`flex flex-col items-center justify-center gap-2 border-2 border-dashed rounded-[var(--radius-lg)] py-8 cursor-pointer transition-colors ${
          dragging
            ? 'border-cosmic-blue bg-cosmic-blue/5'
            : 'border-space-border hover:border-cosmic-blue/40'
        }`}
      >
        <svg
          className="w-8 h-8 text-text-muted"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={1.5}
            d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"
          />
        </svg>
        {file ? (
          <span className="text-sm text-cosmic-blue font-medium">{file.name}</span>
        ) : (
          <span className="text-sm text-text-muted">Drop .zip here or click to select</span>
        )}
        <input
          ref={inputRef}
          type="file"
          accept=".zip"
          className="hidden"
          onChange={(e) => setFile(e.target.files?.[0] ?? null)}
        />
      </div>

      <div className="flex flex-wrap items-center gap-4">
        <label className="flex items-center gap-2 cursor-pointer select-none">
          <input
            type="checkbox"
            checked={eraseExisting}
            onChange={(e) => setEraseExisting(e.target.checked)}
            className="w-4 h-4 rounded accent-nova-red"
          />
          <span className="text-sm text-text-secondary">Erase existing data</span>
        </label>

        <div className="flex items-center gap-2">
          <label className="text-xs font-medium text-text-muted">Conflict mode:</label>
          <select
            value={conflictMode}
            onChange={(e) => setConflictMode(e.target.value as typeof conflictMode)}
            className="h-8 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-2 text-sm text-text-primary focus:outline-none"
          >
            <option value="overwrite">overwrite</option>
          </select>
        </div>

        <button
          onClick={handleImport}
          disabled={!file || isPending}
          className="ml-auto px-4 py-1.5 text-sm rounded-[var(--radius-md)] bg-cosmic-blue hover:bg-cosmic-blue/90 text-white transition-colors disabled:opacity-50"
        >
          {isPending ? 'Importing…' : 'Import'}
        </button>
      </div>

      {result && (
        <div
          className={`text-xs rounded p-3 ${result.success ? 'bg-nebula-green/10 text-nebula-green' : 'bg-nova-red/10 text-nova-red'}`}
        >
          {result.success ? 'Import successful' : 'Import failed'}
          {(result.errors ?? []).length > 0 && (
            <ul className="mt-1 list-disc list-inside">
              {result.errors!.map((e, i) => (
                <li key={i}>{e}</li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Section: Import CSV
// ---------------------------------------------------------------------------
function ImportCsvSection() {
  const { mutateAsync: importCsv, isPending } = useImportCsv()
  const [file, setFile] = useState<File | null>(null)
  const [table, setTable] = useState<CsvTable>('users')
  const [result, setResult] = useState<CSVImportResult | null>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const handleImport = async () => {
    if (!file) return
    try {
      const res = await importCsv({ file, table })
      setResult(res)
      setFile(null)
    } catch {
      // error already toasted
    }
  }

  return (
    <div className="flex flex-col gap-4 p-4 rounded-[var(--radius-lg)] border border-space-border bg-space-card">
      <h2 className="text-sm font-semibold text-text-primary">Import CSV</h2>

      <div className="flex flex-wrap items-end gap-3">
        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-text-muted">Target table</label>
          <select
            value={table}
            onChange={(e) => setTable(e.target.value as CsvTable)}
            className="h-9 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-2 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
          >
            {CSV_TABLES.map((t) => (
              <option key={t} value={t}>
                {t}
              </option>
            ))}
          </select>
        </div>

        <div
          onClick={() => inputRef.current?.click()}
          className="flex items-center gap-2 h-9 px-3 rounded-[var(--radius-md)] border border-dashed border-space-border cursor-pointer hover:border-cosmic-blue/40 transition-colors text-sm text-text-muted"
        >
          {file ? <span className="text-text-primary">{file.name}</span> : <span>Select .csv</span>}
          <input
            ref={inputRef}
            type="file"
            accept=".csv"
            className="hidden"
            onChange={(e) => setFile(e.target.files?.[0] ?? null)}
          />
        </div>

        <button
          onClick={handleImport}
          disabled={!file || isPending}
          className="h-9 px-4 text-sm rounded-[var(--radius-md)] bg-cosmic-blue hover:bg-cosmic-blue/90 text-white transition-colors disabled:opacity-50"
        >
          {isPending ? 'Importing…' : 'Import CSV'}
        </button>
      </div>

      {result && (
        <div className="text-xs rounded p-3 bg-nebula-green/10 text-nebula-green">
          Imported {result.imported_count ?? 0} rows
          {(result.skipped_count ?? 0) > 0 && `, skipped ${result.skipped_count}`}
          {(result.errors ?? []).length > 0 && (
            <ul className="mt-1 list-disc list-inside text-nova-red">
              {result.errors!.map((e, i) => (
                <li key={i}>{e}</li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Section: Reset
// ---------------------------------------------------------------------------
function ResetSection() {
  const { mutateAsync: reset, isPending } = useReset()
  const [opts, setOpts] = useState<ResetBody>({
    accounts: false,
    submissions: false,
    challenges: false,
    pages: false,
    notifications: false,
  })
  const [confirm, setConfirm] = useState('')
  const [step, setStep] = useState<1 | 2>(1)

  const anySelected = Object.values(opts).some(Boolean)
  const canReset = confirm === 'RESET' && anySelected

  const toggle = (k: keyof ResetBody) => setOpts((prev) => ({ ...prev, [k]: !prev[k] }))

  const handleReset = async () => {
    if (!canReset) return
    try {
      await reset(opts)
      setOpts({
        accounts: false,
        submissions: false,
        challenges: false,
        pages: false,
        notifications: false,
      })
      setConfirm('')
      setStep(1)
    } catch {
      // error already toasted
    }
  }

  const RESET_OPTIONS: { key: keyof ResetBody; label: string; danger?: boolean }[] = [
    { key: 'submissions', label: 'Submissions & solves' },
    { key: 'notifications', label: 'Notifications' },
    { key: 'pages', label: 'Pages' },
    { key: 'challenges', label: 'Challenges, hints & files', danger: true },
    { key: 'accounts', label: 'Users & teams', danger: true },
  ]

  return (
    <div className="flex flex-col gap-4 p-4 rounded-[var(--radius-lg)] border border-nova-red/30 bg-nova-red/5">
      <div className="flex items-center gap-2">
        <svg
          className="w-4 h-4 text-nova-red shrink-0"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
          />
        </svg>
        <h2 className="text-sm font-semibold text-nova-red">Danger Zone - Reset</h2>
      </div>

      {step === 1 && (
        <>
          <p className="text-xs text-text-muted">
            Select which data to reset. This action is irreversible.
          </p>

          <div className="flex flex-col gap-2">
            {RESET_OPTIONS.map(({ key, label, danger }) => (
              <label key={key} className="flex items-center gap-2 cursor-pointer select-none">
                <input
                  type="checkbox"
                  checked={opts[key] ?? false}
                  onChange={() => toggle(key)}
                  className={`w-4 h-4 rounded ${danger ? 'accent-nova-red' : 'accent-cosmic-blue'}`}
                />
                <span className={`text-sm ${danger ? 'text-nova-red' : 'text-text-secondary'}`}>
                  {label}
                </span>
              </label>
            ))}
          </div>

          <button
            onClick={() => setStep(2)}
            disabled={!anySelected}
            className="self-start px-4 py-1.5 text-sm rounded-[var(--radius-md)] border border-nova-red text-nova-red hover:bg-nova-red/10 transition-colors disabled:opacity-40"
          >
            Continue…
          </button>
        </>
      )}

      {step === 2 && (
        <>
          <p className="text-xs text-nova-red font-medium">
            You are about to reset:{' '}
            {RESET_OPTIONS.filter(({ key }) => opts[key])
              .map(({ label }) => label)
              .join(', ')}
          </p>
          <p className="text-xs text-text-muted">
            Type <span className="font-mono font-bold text-nova-red">RESET</span> to confirm:
          </p>
          <input
            type="text"
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            placeholder="RESET"
            className="h-9 w-48 rounded-[var(--radius-md)] border border-nova-red/50 bg-space-dark px-3 text-sm font-mono text-nova-red placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-nova-red/40"
          />
          <div className="flex gap-3">
            <button
              onClick={() => {
                setStep(1)
                setConfirm('')
              }}
              className="px-4 py-1.5 text-sm rounded-[var(--radius-md)] border border-space-border text-text-secondary hover:text-text-primary transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={handleReset}
              disabled={!canReset || isPending}
              className="px-4 py-1.5 text-sm rounded-[var(--radius-md)] bg-nova-red hover:bg-nova-red/90 text-white transition-colors disabled:opacity-40"
            >
              {isPending ? 'Resetting…' : 'Reset now'}
            </button>
          </div>
        </>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------
export function AdminBackupPage() {
  return (
    <div className="flex flex-col gap-5">
      <h1
        className="text-2xl font-black text-text-primary"
        style={{ fontFamily: 'var(--font-display)' }}
      >
        Backup &amp; Restore
      </h1>

      <ExportSection />
      <ImportZipSection />
      <ImportCsvSection />
      <ResetSection />
    </div>
  )
}
