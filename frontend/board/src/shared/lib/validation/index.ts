/**
 * Client-side field validators that mirror the backend rules in
 * backend/pkg/validator/validator.go exactly.
 * Each function returns an error message string on failure, or null on success.
 */

// Mirrors passwordRegex in validator.go
const PASSWORD_CHARSET_RE = /^[a-zA-Z0-9!@#$%^&*()_+\-=[\]{};':"|,.<>/?]+$/

// Mirrors usernameRegex in validator.go
const USERNAME_RE = /^[a-zA-Z0-9._%+-]+$/

// Mirrors EmailRegex in validator.go
const EMAIL_RE = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/

// Mirrors teamNameRegex in validator.go
const TEAM_NAME_RE = /^[a-zA-Z0-9\s._-]+$/

export const PASSWORD_MSG =
  'Password must be 8–72 characters, contain at least one uppercase letter, one lowercase letter, and one digit. Allowed special characters: !@#$%^&*()_+-=[]{};\':"|,.<>/?'

export const USERNAME_MSG = 'Username must be 1–50 characters. Allowed: letters, digits, . _ % + -'

export const EMAIL_MSG = 'Enter a valid email address.'

export const TEAM_NAME_MSG =
  'Team name must be 1–50 characters. Allowed: letters, digits, spaces, . _ -'

/** Returns an error message or null. Matches backend `strong_password` tag. */
export function validatePassword(password: string): string | null {
  if (!password) return 'Password is required.'
  if (password.length < 8 || password.length > 72) {
    return PASSWORD_MSG
  }
  if (!PASSWORD_CHARSET_RE.test(password)) {
    return PASSWORD_MSG
  }
  if (!/[a-z]/.test(password) || !/[A-Z]/.test(password) || !/[0-9]/.test(password)) {
    return PASSWORD_MSG
  }
  return null
}

/** Returns an error message or null. Matches backend `custom_username` tag. */
export function validateUsername(username: string): string | null {
  if (!username) return 'Username is required.'
  if (username.length > 50 || !USERNAME_RE.test(username)) return USERNAME_MSG
  return null
}

/** Returns an error message or null. Matches backend `custom_email` tag. */
export function validateEmail(email: string): string | null {
  if (!email) return 'Email is required.'
  if (email.length > 254 || !EMAIL_RE.test(email)) return EMAIL_MSG
  return null
}

/** Returns an error message or null. Matches backend `team_name` tag. */
export function validateTeamName(name: string): string | null {
  if (!name) return 'Team name is required.'
  if (name.length > 50 || !TEAM_NAME_RE.test(name)) return TEAM_NAME_MSG
  return null
}
