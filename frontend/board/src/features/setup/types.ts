export type CompetitionMode = 'teams_only' | 'solo_only' | 'flexible'
export type Visibility = 'public' | 'private' | 'hidden' | 'admins_only' | 'admins'

export interface SetupFormData {
  // General
  ctf_name: string
  ctf_description: string

  // Mode
  mode: CompetitionMode
  max_team_size: string

  // Settings
  challenge_visibility: Visibility
  score_visibility: Visibility
  account_visibility: Visibility
  registration_visibility: Visibility
  email_verification_required: boolean

  // Administration
  setup_token: string
  admin_username: string
  admin_email: string
  admin_password: string

  // Style (stubs - no values submitted)

  // Date & Time
  start_time: string // ISO string or ''
  end_time: string
  freeze_time: string
  timezone: string
}

export const SETUP_DEFAULTS: SetupFormData = {
  ctf_name: '',
  ctf_description: '',
  mode: 'teams_only',
  max_team_size: '10',
  challenge_visibility: 'private',
  score_visibility: 'public',
  account_visibility: 'public',
  registration_visibility: 'public',
  email_verification_required: false,
  setup_token: '',
  admin_username: '',
  admin_email: '',
  admin_password: '',
  start_time: '',
  end_time: '',
  freeze_time: '',
  timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
}

export type SetupStep = 0 | 1 | 2 | 3 | 4 | 5

export const STEP_LABELS: Record<SetupStep, string> = {
  0: 'General',
  1: 'Mode',
  2: 'Settings',
  3: 'Administration',
  4: 'Style',
  5: 'Date & Time',
}
