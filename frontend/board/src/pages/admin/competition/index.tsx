import { api, isApiError } from '@/shared/api/client'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { DateTimePicker } from '@/shared/ui/date-time-picker'
import { Skeleton } from '@/shared/ui/skeleton'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { toast } from 'sonner'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------
type CompetitionData = {
  id?: number
  name?: string
  status?: string
  mode?: string
  start_time?: string
  end_time?: string
  freeze_time?: string
  is_paused?: boolean
  is_public?: boolean
  allow_team_switch?: boolean
  keep_scoreboard_frozen_after_end?: boolean
  min_team_size?: number
  max_team_size?: number
  flag_regex?: string
  paused_at?: string
}

type CompetitionMode = 'teams' | 'solo' | 'flexible'

// ---------------------------------------------------------------------------
// Hooks
// ---------------------------------------------------------------------------
function useCompetition() {
  return useQuery({
    queryKey: ['admin', 'competition'],
    queryFn: async () => {
      const { data, error } = await api.GET('/admin/competition')
      if (error) throw error
      return (data ?? {}) as CompetitionData
    },
    staleTime: 0,
  })
}

function useUpdateCompetition() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (body: {
      name: string
      start_time?: string
      end_time?: string
      freeze_time?: string
      clear_end_time?: boolean
      clear_freeze_time?: boolean
      is_paused?: boolean
      is_public?: boolean
      mode?: string
      min_team_size?: number
      max_team_size?: number
      flag_regex?: string
      allow_team_switch?: boolean
      keep_scoreboard_frozen_after_end?: boolean
    }) => {
      const { data, error } = await api.PUT('/admin/competition', { body })
      if (error) throw error
      return data
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'competition'] })
      void qc.invalidateQueries({ queryKey: ['competition'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Status bar
// ---------------------------------------------------------------------------
const STATUS_CONFIG: Record<
  string,
  { label: string; variant: 'green' | 'yellow' | 'blue' | 'default' | 'red' | 'cyan' }
> = {
  active: { label: 'Active', variant: 'green' },
  not_started: { label: 'Not started', variant: 'default' },
  paused: { label: 'Paused', variant: 'yellow' },
  frozen: { label: 'Scoreboard frozen', variant: 'cyan' },
  ended: { label: 'Ended', variant: 'red' },
}

function StatusBar({ comp }: { comp: CompetitionData }) {
  const status = comp.status ?? 'not_started'
  const cfg = STATUS_CONFIG[status] ?? { label: status, variant: 'default' as const }

  return (
    <div className="flex items-center gap-3 px-4 py-3 rounded-[var(--radius-lg)] border border-space-border bg-space-card">
      <div className="flex items-center gap-2 flex-1 flex-wrap">
        <span className="text-sm text-text-muted">Status:</span>
        <Badge variant={cfg.variant}>{cfg.label}</Badge>
        {comp.is_paused && comp.paused_at && (
          <span className="text-xs text-text-muted">
            Paused at {new Date(comp.paused_at).toLocaleString()}
          </span>
        )}
      </div>
      {comp.start_time && (
        <span className="text-xs text-text-muted hidden sm:block">
          Starts: {new Date(comp.start_time).toLocaleString()}
        </span>
      )}
      {comp.end_time && (
        <span className="text-xs text-text-muted hidden sm:block">
          Ends: {new Date(comp.end_time).toLocaleString()}
        </span>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Section wrapper
// ---------------------------------------------------------------------------
function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="rounded-[var(--radius-lg)] border border-space-border bg-space-card p-5 flex flex-col gap-4">
      <h2 className="text-sm font-semibold text-text-muted uppercase tracking-wide">{title}</h2>
      {children}
    </div>
  )
}

function FieldRow({
  label,
  hint,
  children,
}: {
  label: string
  hint?: string
  children: React.ReactNode
}) {
  return (
    <div className="flex flex-col sm:flex-row sm:items-start gap-2 sm:gap-4">
      <div className="sm:w-52 flex-shrink-0">
        <p className="text-sm font-medium text-text-primary">{label}</p>
        {hint && <p className="text-xs text-text-muted mt-0.5">{hint}</p>}
      </div>
      <div className="flex-1">{children}</div>
    </div>
  )
}

function Toggle({
  checked,
  onChange,
  label,
}: {
  checked: boolean
  onChange: (v: boolean) => void
  label: string
}) {
  return (
    <label className="flex items-center gap-2 cursor-pointer w-fit">
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        onClick={() => onChange(!checked)}
        className={`relative w-10 h-5.5 rounded-full transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cosmic-blue/50
          ${checked ? 'bg-cosmic-blue' : 'bg-space-border'}`}
      >
        <span
          className={`absolute top-0.5 left-0.5 w-4.5 h-4.5 rounded-full bg-white shadow transition-transform
            ${checked ? 'translate-x-4.5' : 'translate-x-0'}`}
        />
      </button>
      <span className="text-sm text-text-secondary select-none">{label}</span>
    </label>
  )
}

// ---------------------------------------------------------------------------
// Main form
// ---------------------------------------------------------------------------
function CompetitionForm({ comp }: { comp: CompetitionData }) {
  const { mutateAsync: update, isPending } = useUpdateCompetition()

  // General
  const [name, setName] = useState(comp.name ?? '')
  const [mode, setMode] = useState<CompetitionMode>((comp.mode as CompetitionMode) ?? 'teams')
  const [isPublic, setIsPublic] = useState(comp.is_public ?? false)
  const [isPaused, setIsPaused] = useState(comp.is_paused ?? false)

  // Times
  const [startTime, setStartTime] = useState<Date | undefined>(
    comp.start_time ? new Date(comp.start_time) : undefined,
  )
  const [endTime, setEndTime] = useState<Date | undefined>(
    comp.end_time ? new Date(comp.end_time) : undefined,
  )
  const [freezeTime, setFreezeTime] = useState<Date | undefined>(
    comp.freeze_time ? new Date(comp.freeze_time) : undefined,
  )

  // Team settings
  const [minTeamSize, setMinTeamSize] = useState(comp.min_team_size ?? 1)
  const [maxTeamSize, setMaxTeamSize] = useState(comp.max_team_size ?? 10)
  const [allowTeamSwitch, setAllowTeamSwitch] = useState(comp.allow_team_switch ?? false)

  // Scoreboard
  const [keepFrozen, setKeepFrozen] = useState(comp.keep_scoreboard_frozen_after_end ?? false)

  // Flag
  const [flagRegex, setFlagRegex] = useState(comp.flag_regex ?? '')

  const [error, setError] = useState('')
  const [prevComp, setPrevComp] = useState(comp)

  if (prevComp !== comp) {
    setPrevComp(comp)
    setName(comp.name ?? '')
    setMode((comp.mode as CompetitionMode) ?? 'teams')
    setIsPublic(comp.is_public ?? false)
    setIsPaused(comp.is_paused ?? false)
    setStartTime(comp.start_time ? new Date(comp.start_time) : undefined)
    setEndTime(comp.end_time ? new Date(comp.end_time) : undefined)
    setFreezeTime(comp.freeze_time ? new Date(comp.freeze_time) : undefined)
    setMinTeamSize(comp.min_team_size ?? 1)
    setMaxTeamSize(comp.max_team_size ?? 10)
    setAllowTeamSwitch(comp.allow_team_switch ?? false)
    setKeepFrozen(comp.keep_scoreboard_frozen_after_end ?? false)
    setFlagRegex(comp.flag_regex ?? '')
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    try {
      type Body = Parameters<typeof update>[0]
      const body: Body = {
        name,
        mode,
        is_public: isPublic,
        is_paused: isPaused,
        min_team_size: minTeamSize,
        max_team_size: maxTeamSize,
        allow_team_switch: allowTeamSwitch,
        keep_scoreboard_frozen_after_end: keepFrozen,
      }
      body.flag_regex = flagRegex
      if (startTime) body.start_time = startTime.toISOString()
      if (endTime) {
        body.end_time = endTime.toISOString()
      } else {
        body.clear_end_time = true
      }
      if (freezeTime) {
        body.freeze_time = freezeTime.toISOString()
      } else {
        body.clear_freeze_time = true
      }

      await update(body)
      toast.success('Competition settings saved')
    } catch (err) {
      setError(isApiError(err) ? err.message : 'Save failed')
    }
  }

  const inputCls =
    'h-9 w-full rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40'

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-5">
      {/* General */}
      <Section title="General">
        <FieldRow label="Competition name" hint="Displayed to participants">
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
            className={inputCls}
          />
        </FieldRow>

        <FieldRow label="Mode" hint="How participants compete">
          <div className="flex gap-4 flex-wrap">
            {(['teams', 'solo', 'flexible'] as const).map((m) => (
              <label key={m} className="flex items-center gap-2 cursor-pointer">
                <input
                  type="radio"
                  name="mode"
                  value={m}
                  checked={mode === m}
                  onChange={() => setMode(m)}
                  className="accent-cosmic-blue"
                />
                <span className="text-sm text-text-secondary capitalize">{m}</span>
              </label>
            ))}
          </div>
        </FieldRow>

        <FieldRow label="Visibility">
          <Toggle
            checked={isPublic}
            onChange={setIsPublic}
            label="Public (visible to unauthenticated users)"
          />
        </FieldRow>

        <FieldRow label="Paused">
          <Toggle
            checked={isPaused}
            onChange={setIsPaused}
            label="Pause competition (submissions disabled)"
          />
        </FieldRow>
      </Section>

      {/* Schedule */}
      <Section title="Schedule">
        <FieldRow label="Start time" hint="UTC - leave empty for manual start">
          <DateTimePicker {...(startTime ? { value: startTime } : {})} onChange={setStartTime} />
        </FieldRow>
        <FieldRow label="End time" hint="Leave empty for no end time">
          <div className="flex items-center gap-2">
            <DateTimePicker
              {...(endTime ? { value: endTime } : {})}
              onChange={setEndTime}
              className="flex-1"
            />
            {endTime && (
              <button
                type="button"
                onClick={() => setEndTime(undefined)}
                className="text-xs text-text-muted hover:text-nova-red transition-colors whitespace-nowrap"
              >
                Clear
              </button>
            )}
          </div>
        </FieldRow>
        <FieldRow label="Freeze time" hint="Scoreboard hidden after this time">
          <div className="flex items-center gap-2">
            <DateTimePicker
              {...(freezeTime ? { value: freezeTime } : {})}
              onChange={setFreezeTime}
              className="flex-1"
            />
            {freezeTime && (
              <button
                type="button"
                onClick={() => setFreezeTime(undefined)}
                className="text-xs text-text-muted hover:text-nova-red transition-colors whitespace-nowrap"
              >
                Clear
              </button>
            )}
          </div>
        </FieldRow>
      </Section>

      {/* Team settings */}
      <Section title="Team settings">
        <FieldRow label="Team size" hint="Min and max participants per team">
          <div className="flex items-center gap-3">
            <div className="flex flex-col gap-1">
              <label className="text-xs text-text-muted">Min</label>
              <input
                type="number"
                min={1}
                max={maxTeamSize}
                value={minTeamSize}
                onChange={(e) => setMinTeamSize(Number(e.target.value))}
                className="h-9 w-24 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
              />
            </div>
            <span className="text-text-muted mt-5">-</span>
            <div className="flex flex-col gap-1">
              <label className="text-xs text-text-muted">Max</label>
              <input
                type="number"
                min={minTeamSize}
                value={maxTeamSize}
                onChange={(e) => setMaxTeamSize(Number(e.target.value))}
                className="h-9 w-24 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40"
              />
            </div>
          </div>
        </FieldRow>

        <FieldRow label="Team switching">
          <Toggle
            checked={allowTeamSwitch}
            onChange={setAllowTeamSwitch}
            label="Allow participants to switch teams"
          />
        </FieldRow>
      </Section>

      {/* Scoreboard */}
      <Section title="Scoreboard">
        <FieldRow
          label="Keep frozen after end"
          hint="Scoreboard stays frozen after competition ends"
        >
          <Toggle
            checked={keepFrozen}
            onChange={setKeepFrozen}
            label="Keep scoreboard frozen after end time"
          />
        </FieldRow>
      </Section>

      {/* Flag */}
      <Section title="Flag format">
        <FieldRow label="Flag regex" hint="Optional - validates flag format on submit">
          <input
            type="text"
            value={flagRegex}
            onChange={(e) => setFlagRegex(e.target.value)}
            placeholder="e.g. ^CTF\{.+\}$"
            className={inputCls}
          />
        </FieldRow>
      </Section>

      {error && (
        <p className="text-sm text-nova-red bg-nova-red/10 rounded-[var(--radius-md)] px-4 py-3">
          {error}
        </p>
      )}

      <div className="flex items-center gap-3">
        <Button type="submit" variant="primary" loading={isPending}>
          Save competition settings
        </Button>
      </div>
    </form>
  )
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------
export function AdminCompetitionPage() {
  const { data: comp, isLoading } = useCompetition()

  return (
    <div className="flex flex-col gap-5 max-w-2xl">
      <h1
        className="text-2xl font-black text-text-primary"
        style={{ fontFamily: 'var(--font-display)' }}
      >
        Competition
      </h1>

      {isLoading ? (
        <div className="flex flex-col gap-4">
          <Skeleton className="h-14 w-full" />
          <Skeleton className="h-40 w-full" />
          <Skeleton className="h-40 w-full" />
        </div>
      ) : comp ? (
        <>
          <StatusBar comp={comp} />
          <CompetitionForm comp={comp} />
        </>
      ) : (
        <p className="text-text-muted">No competition data.</p>
      )}
    </div>
  )
}
