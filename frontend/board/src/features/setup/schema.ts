import { type SetupFormData, type SetupStep } from './types'

type ValidationError = Partial<Record<keyof SetupFormData, string>>

function isValidDateTimeLocal(s: string): boolean {
  if (!s) return true
  const d = new Date(s)
  if (Number.isNaN(d.getTime())) return false
  const [date] = s.split('T')
  if (!date) return false
  const [y, m, day] = date.split('-').map(Number)
  return d.getFullYear() === y && d.getMonth() + 1 === m && d.getDate() === day
}

export function validateStep(step: SetupStep, data: SetupFormData): ValidationError {
  const errors: ValidationError = {}

  switch (step) {
    case 0: // General
      if (!data.ctf_name.trim()) errors.ctf_name = 'CTF name is required'
      else if (data.ctf_name.length > 100)
        errors.ctf_name = 'CTF name must be 100 characters or fewer'
      break

    case 1: // Mode
      if (!['teams_only', 'solo_only', 'flexible'].includes(data.mode)) {
        errors.mode = 'Please select a valid mode'
      }
      if (data.mode !== 'solo_only') {
        if (!/^\d+$/.test(data.max_team_size)) {
          errors.max_team_size = 'Team size must be a positive integer'
        } else {
          const n = parseInt(data.max_team_size, 10)
          if (n < 1 || n > 100) {
            errors.max_team_size = 'Team size must be between 1 and 100'
          }
        }
      }
      break

    case 2: // Settings
      break // all have defaults, nothing mandatory

    case 3: // Administration
      if (!data.admin_username.trim()) errors.admin_username = 'Username is required'
      else if (data.admin_username.length < 3)
        errors.admin_username = 'Username must be at least 3 characters'
      else if (!/^[a-zA-Z0-9_.-]+$/.test(data.admin_username)) {
        errors.admin_username =
          'Username may only contain letters, numbers, underscores, hyphens, and dots'
      }
      if (!data.admin_email.trim()) errors.admin_email = 'Email is required'
      else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(data.admin_email)) {
        errors.admin_email = 'Please enter a valid email address'
      }
      if (data.admin_password.length < 8)
        errors.admin_password = 'Password must be at least 8 characters'
      break

    case 4: // Style - stubs, no validation
      break

    case 5: // Date & Time - all optional
      if (!isValidDateTimeLocal(data.start_time)) {
        errors.start_time = 'Invalid date'
      }
      if (!isValidDateTimeLocal(data.end_time)) {
        errors.end_time = 'Invalid date'
      }
      if (!isValidDateTimeLocal(data.freeze_time)) {
        errors.freeze_time = 'Invalid date'
      }
      if (!errors.start_time && !errors.end_time && data.start_time && data.end_time) {
        if (new Date(data.end_time) <= new Date(data.start_time)) {
          errors.end_time = 'End time must be after start time'
        }
      }
      if (!errors.freeze_time && !errors.end_time && data.freeze_time && data.end_time) {
        if (new Date(data.freeze_time) >= new Date(data.end_time)) {
          errors.freeze_time = 'Freeze time must be before end time'
        }
      }
      break
  }

  return errors
}
