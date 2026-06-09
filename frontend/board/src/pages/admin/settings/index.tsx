import { api, isApiError } from '@/shared/api/client'
import type { operations } from '@/shared/api/schema'
import { Button } from '@/shared/ui/button'
import { Skeleton } from '@/shared/ui/skeleton'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { toast } from 'sonner'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------
type Settings = NonNullable<
  operations['PutAdminSettings']['requestBody']['content']['application/json']
>

// ---------------------------------------------------------------------------
// Hooks
// ---------------------------------------------------------------------------
function useAdminSettings() {
  return useQuery({
    queryKey: ['admin', 'settings'],
    queryFn: async () => {
      const { data, error } = await api.GET('/admin/settings')
      if (error) throw error
      return (data ?? {}) as Settings
    },
    staleTime: 0,
  })
}

function useUpdateSettings() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (body: Settings) => {
      const { data, error } = await api.PUT('/admin/settings', { body })
      if (error) throw error
      return data
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['admin', 'settings'] })
      void qc.invalidateQueries({ queryKey: ['configs', 'public'] })
    },
  })
}

// ---------------------------------------------------------------------------
// UI helpers
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
      <div className="sm:w-56 flex-shrink-0">
        <p className="text-sm font-medium text-text-primary">{label}</p>
        {hint && <p className="text-xs text-text-muted mt-0.5 leading-relaxed">{hint}</p>}
      </div>
      <div className="flex-1">{children}</div>
    </div>
  )
}

function Toggle({ checked, onChange }: { checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      onClick={() => onChange(!checked)}
      className={`relative inline-flex w-10 h-6 rounded-full transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cosmic-blue/50
        ${checked ? 'bg-cosmic-blue' : 'bg-space-border'}`}
    >
      <span
        className={`absolute top-0.5 left-0.5 w-5 h-5 rounded-full bg-white shadow transition-transform
          ${checked ? 'translate-x-4' : 'translate-x-0'}`}
      />
    </button>
  )
}

const inputCls =
  'h-9 w-full rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40'

const numberInputCls =
  'h-9 w-28 rounded-[var(--radius-md)] border border-space-border bg-space-dark px-3 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-cosmic-blue/40'

function NumericField({ value, onChange }: { value: number; onChange: (v: number) => void }) {
  return (
    <input
      type="number"
      min={0}
      value={value}
      onChange={(e) => onChange(Number(e.target.value))}
      className={numberInputCls}
    />
  )
}

// ---------------------------------------------------------------------------
// Form
// ---------------------------------------------------------------------------
function SettingsForm({ settings }: { settings: Settings }) {
  const { mutateAsync: save, isPending } = useUpdateSettings()

  // General
  const [appName, setAppName] = useState(settings.app_name ?? '')
  const [frontendUrl, setFrontendUrl] = useState(settings.frontend_url ?? '')
  const [corsOrigins, setCorsOrigins] = useState(settings.cors_origins ?? '')
  const [maxTeams, setMaxTeams] = useState(settings.max_teams ?? 0)

  // Registration
  const [registrationOpen, setRegistrationOpen] = useState(settings.registration_open ?? true)
  const [verifyEmails, setVerifyEmails] = useState(settings.verify_emails ?? false)
  const [verifyTtlHours, setVerifyTtlHours] = useState(settings.verify_ttl_hours ?? 24)
  const [resetTtlHours, setResetTtlHours] = useState(settings.reset_ttl_hours ?? 24)

  // Scoreboard
  const [scoreboardVisible, setScoreboardVisible] = useState<
    NonNullable<Settings['scoreboard_visible']>
  >(settings.scoreboard_visible ?? 'public')
  const [writeupEnabled, setWriteupEnabled] = useState(settings.writeup_enabled ?? false)

  // OAuth
  const [oauthGithub, setOauthGithub] = useState(settings.oauth_github_enabled ?? false)
  const [oauthGoogle, setOauthGoogle] = useState(settings.oauth_google_enabled ?? false)

  // Pagination
  const [defaultPerPage, setDefaultPerPage] = useState(settings.default_per_page ?? 20)
  const [maxPerPage, setMaxPerPage] = useState(settings.max_per_page ?? 100)
  const [csvExportMaxRows, setCsvExportMaxRows] = useState(settings.csv_export_max_rows ?? 10000)

  // Submit limits
  const [submitLimitPerUser, setSubmitLimitPerUser] = useState(settings.submit_limit_per_user ?? 0)
  const [submitLimitDurationMin, setSubmitLimitDurationMin] = useState(
    settings.submit_limit_duration_min ?? 1,
  )

  // Rate limits
  const [rlGeneralIp, setRlGeneralIp] = useState(settings.rate_limit_general_ip_per_minute ?? 300)
  const [rlLogin, setRlLogin] = useState(settings.rate_limit_login_per_minute ?? 10)
  const [rlLogout, setRlLogout] = useState(settings.rate_limit_logout_per_minute ?? 30)
  const [rlRegister, setRlRegister] = useState(settings.rate_limit_register_per_minute ?? 5)
  const [rlForgotPw, setRlForgotPw] = useState(settings.rate_limit_forgot_password_per_minute ?? 5)
  const [rlResetPw, setRlResetPw] = useState(settings.rate_limit_reset_password_per_minute ?? 10)
  const [rlVerifyEmail, setRlVerifyEmail] = useState(
    settings.rate_limit_verify_email_per_minute ?? 5,
  )
  const [rlRefresh, setRlRefresh] = useState(settings.rate_limit_refresh_per_minute ?? 60)
  const [rlScoreboard, setRlScoreboard] = useState(settings.rate_limit_scoreboard_per_minute ?? 60)
  const [rlOauthCb, setRlOauthCb] = useState(settings.rate_limit_oauth_callback_per_minute ?? 20)
  const [rlOauthRedir, setRlOauthRedir] = useState(
    settings.rate_limit_oauth_redirect_per_minute ?? 20,
  )
  const [rlComment, setRlComment] = useState(settings.rate_limit_comment_per_minute ?? 10)

  const [error, setError] = useState('')
  const [prevSettings, setPrevSettings] = useState(settings)

  if (prevSettings !== settings) {
    setPrevSettings(settings)
    setAppName(settings.app_name ?? '')
    setFrontendUrl(settings.frontend_url ?? '')
    setCorsOrigins(settings.cors_origins ?? '')
    setMaxTeams(settings.max_teams ?? 0)
    setRegistrationOpen(settings.registration_open ?? true)
    setVerifyEmails(settings.verify_emails ?? false)
    setVerifyTtlHours(settings.verify_ttl_hours ?? 24)
    setResetTtlHours(settings.reset_ttl_hours ?? 24)
    setScoreboardVisible(settings.scoreboard_visible ?? 'public')
    setWriteupEnabled(settings.writeup_enabled ?? false)
    setOauthGithub(settings.oauth_github_enabled ?? false)
    setOauthGoogle(settings.oauth_google_enabled ?? false)
    setDefaultPerPage(settings.default_per_page ?? 20)
    setMaxPerPage(settings.max_per_page ?? 100)
    setCsvExportMaxRows(settings.csv_export_max_rows ?? 10000)
    setSubmitLimitPerUser(settings.submit_limit_per_user ?? 0)
    setSubmitLimitDurationMin(settings.submit_limit_duration_min ?? 1)
    setRlGeneralIp(settings.rate_limit_general_ip_per_minute ?? 300)
    setRlLogin(settings.rate_limit_login_per_minute ?? 10)
    setRlLogout(settings.rate_limit_logout_per_minute ?? 30)
    setRlRegister(settings.rate_limit_register_per_minute ?? 5)
    setRlForgotPw(settings.rate_limit_forgot_password_per_minute ?? 5)
    setRlResetPw(settings.rate_limit_reset_password_per_minute ?? 10)
    setRlVerifyEmail(settings.rate_limit_verify_email_per_minute ?? 5)
    setRlRefresh(settings.rate_limit_refresh_per_minute ?? 60)
    setRlScoreboard(settings.rate_limit_scoreboard_per_minute ?? 60)
    setRlOauthCb(settings.rate_limit_oauth_callback_per_minute ?? 20)
    setRlOauthRedir(settings.rate_limit_oauth_redirect_per_minute ?? 20)
    setRlComment(settings.rate_limit_comment_per_minute ?? 10)
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    try {
      const body: Settings = {
        app_name: appName,
        frontend_url: frontendUrl,
        cors_origins: corsOrigins,
        max_teams: maxTeams,
        registration_open: registrationOpen,
        verify_emails: verifyEmails,
        verify_ttl_hours: verifyTtlHours,
        reset_ttl_hours: resetTtlHours,
        scoreboard_visible: scoreboardVisible,
        writeup_enabled: writeupEnabled,
        oauth_github_enabled: oauthGithub,
        oauth_google_enabled: oauthGoogle,
        default_per_page: defaultPerPage,
        max_per_page: maxPerPage,
        csv_export_max_rows: csvExportMaxRows,
        submit_limit_per_user: submitLimitPerUser,
        submit_limit_duration_min: submitLimitDurationMin,
        rate_limit_general_ip_per_minute: rlGeneralIp,
        rate_limit_login_per_minute: rlLogin,
        rate_limit_logout_per_minute: rlLogout,
        rate_limit_register_per_minute: rlRegister,
        rate_limit_forgot_password_per_minute: rlForgotPw,
        rate_limit_reset_password_per_minute: rlResetPw,
        rate_limit_verify_email_per_minute: rlVerifyEmail,
        rate_limit_refresh_per_minute: rlRefresh,
        rate_limit_scoreboard_per_minute: rlScoreboard,
        rate_limit_oauth_callback_per_minute: rlOauthCb,
        rate_limit_oauth_redirect_per_minute: rlOauthRedir,
        rate_limit_comment_per_minute: rlComment,
      }
      await save(body)
      toast.success('Settings saved')
    } catch (err) {
      setError(isApiError(err) ? err.message : 'Save failed')
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-5">
      {/* General */}
      <Section title="General">
        <FieldRow label="App name">
          <input
            type="text"
            value={appName}
            onChange={(e) => setAppName(e.target.value)}
            className={inputCls}
          />
        </FieldRow>
        <FieldRow label="Frontend URL" hint="Used in emails and redirects">
          <input
            type="url"
            value={frontendUrl}
            onChange={(e) => setFrontendUrl(e.target.value)}
            className={inputCls}
            placeholder="https://ctf.example.com"
          />
        </FieldRow>
        <FieldRow label="CORS origins" hint="Comma-separated allowed origins">
          <input
            type="text"
            value={corsOrigins}
            onChange={(e) => setCorsOrigins(e.target.value)}
            className={inputCls}
            placeholder="https://ctf.example.com"
          />
        </FieldRow>
        <FieldRow label="Max teams" hint="0 = unlimited">
          <NumericField value={maxTeams} onChange={setMaxTeams} />
        </FieldRow>
      </Section>

      {/* Registration */}
      <Section title="Registration">
        <FieldRow label="Registration open">
          <div className="flex items-center gap-3">
            <Toggle checked={registrationOpen} onChange={setRegistrationOpen} />
            <span className="text-sm text-text-secondary">
              {registrationOpen ? 'Open' : 'Closed'}
            </span>
          </div>
        </FieldRow>
        <FieldRow label="Email verification required">
          <div className="flex items-center gap-3">
            <Toggle checked={verifyEmails} onChange={setVerifyEmails} />
            <span className="text-sm text-text-secondary">
              {verifyEmails ? 'Required' : 'Optional'}
            </span>
          </div>
        </FieldRow>
        <FieldRow label="Verify email TTL" hint="Hours before verification link expires">
          <NumericField value={verifyTtlHours} onChange={setVerifyTtlHours} />
        </FieldRow>
        <FieldRow label="Reset password TTL" hint="Hours before reset link expires">
          <NumericField value={resetTtlHours} onChange={setResetTtlHours} />
        </FieldRow>
      </Section>

      {/* Scoreboard */}
      <Section title="Scoreboard">
        <FieldRow label="Visibility">
          <div className="flex gap-4 flex-wrap">
            {(['public', 'hidden', 'admins_only'] as const).map((v) => (
              <label key={v} className="flex items-center gap-2 cursor-pointer">
                <input
                  type="radio"
                  name="scoreboard_visible"
                  value={v}
                  checked={scoreboardVisible === v}
                  onChange={() => setScoreboardVisible(v)}
                  className="accent-cosmic-blue"
                />
                <span className="text-sm text-text-secondary capitalize">
                  {v.replace('_', ' ')}
                </span>
              </label>
            ))}
          </div>
        </FieldRow>
        <FieldRow label="Writeups / solutions" hint="Show solutions to teams after solving">
          <div className="flex items-center gap-3">
            <Toggle checked={writeupEnabled} onChange={setWriteupEnabled} />
            <span className="text-sm text-text-secondary">
              {writeupEnabled ? 'Enabled' : 'Disabled'}
            </span>
          </div>
        </FieldRow>
      </Section>

      {/* OAuth */}
      <Section title="OAuth">
        <FieldRow label="GitHub OAuth">
          <div className="flex items-center gap-3">
            <Toggle checked={oauthGithub} onChange={setOauthGithub} />
            <span className="text-sm text-text-secondary">
              {oauthGithub ? 'Enabled' : 'Disabled'}
            </span>
          </div>
        </FieldRow>
        <FieldRow label="Google OAuth">
          <div className="flex items-center gap-3">
            <Toggle checked={oauthGoogle} onChange={setOauthGoogle} />
            <span className="text-sm text-text-secondary">
              {oauthGoogle ? 'Enabled' : 'Disabled'}
            </span>
          </div>
        </FieldRow>
      </Section>

      {/* Email / Resend */}
      <Section title="Email (Resend)">
        <FieldRow label="Resend enabled">
          <div className="flex items-center gap-3">
            <span className="text-sm text-text-secondary">
              {settings.resend_enabled ? 'Enabled' : 'Disabled'}
            </span>
          </div>
        </FieldRow>
        <FieldRow label="From email">
          <span className="text-sm text-text-secondary break-all">
            {settings.resend_from_email || '-'}
          </span>
        </FieldRow>
        <FieldRow label="From name">
          <span className="text-sm text-text-secondary">{settings.resend_from_name || '-'}</span>
        </FieldRow>
      </Section>

      {/* Pagination */}
      <Section title="Pagination">
        <FieldRow label="Default page size">
          <NumericField value={defaultPerPage} onChange={setDefaultPerPage} />
        </FieldRow>
        <FieldRow label="Max page size">
          <NumericField value={maxPerPage} onChange={setMaxPerPage} />
        </FieldRow>
        <FieldRow label="CSV export max rows">
          <NumericField value={csvExportMaxRows} onChange={setCsvExportMaxRows} />
        </FieldRow>
        <FieldRow label="Submit limit per user" hint="Max wrong attempts per window (0 = disabled)">
          <NumericField value={submitLimitPerUser} onChange={setSubmitLimitPerUser} />
        </FieldRow>
        <FieldRow label="Submit limit window" hint="Minutes">
          <NumericField value={submitLimitDurationMin} onChange={setSubmitLimitDurationMin} />
        </FieldRow>
      </Section>

      {/* Rate Limits */}
      <Section title="Rate limits (requests / minute)">
        {(
          [
            ['General (per IP)', rlGeneralIp, setRlGeneralIp],
            ['Login', rlLogin, setRlLogin],
            ['Logout', rlLogout, setRlLogout],
            ['Register', rlRegister, setRlRegister],
            ['Forgot password', rlForgotPw, setRlForgotPw],
            ['Reset password', rlResetPw, setRlResetPw],
            ['Verify email', rlVerifyEmail, setRlVerifyEmail],
            ['Token refresh', rlRefresh, setRlRefresh],
            ['Scoreboard', rlScoreboard, setRlScoreboard],
            ['OAuth callback', rlOauthCb, setRlOauthCb],
            ['OAuth redirect', rlOauthRedir, setRlOauthRedir],
            ['Comment', rlComment, setRlComment],
          ] as [string, number, (v: number) => void][]
        ).map(([label, value, setter]) => (
          <FieldRow key={label} label={label}>
            <NumericField value={value} onChange={setter} />
          </FieldRow>
        ))}
      </Section>

      {error && (
        <p className="text-sm text-nova-red bg-nova-red/10 rounded-[var(--radius-md)] px-4 py-3">
          {error}
        </p>
      )}

      <div>
        <Button type="submit" variant="primary" loading={isPending}>
          Save settings
        </Button>
      </div>
    </form>
  )
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------
export function AdminSettingsPage() {
  const { data: settings, isLoading } = useAdminSettings()

  return (
    <div className="flex flex-col gap-5 max-w-2xl">
      <h1
        className="text-2xl font-black text-text-primary"
        style={{ fontFamily: 'var(--font-display)' }}
      >
        Settings
      </h1>

      {isLoading ? (
        <div className="flex flex-col gap-4">
          {[1, 2, 3, 4].map((i) => (
            <Skeleton key={i} className="h-32 w-full" />
          ))}
        </div>
      ) : settings ? (
        <SettingsForm settings={settings} />
      ) : (
        <p className="text-text-muted">No settings data.</p>
      )}
    </div>
  )
}
