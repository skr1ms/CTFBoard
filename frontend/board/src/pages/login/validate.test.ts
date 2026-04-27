import { describe, expect, it } from 'vitest'
import { validateLoginFields } from './index'

describe('validateLoginFields', () => {
  it('passes with valid email and password', () => {
    const r = validateLoginFields('user@example.com', 'secret')
    expect(r.emailError).toBeNull()
    expect(r.passwordError).toBeNull()
  })

  it('rejects empty email', () => {
    const r = validateLoginFields('', 'secret')
    expect(r.emailError).toBe('Email is required.')
  })

  it('rejects malformed email', () => {
    const r = validateLoginFields('not-an-email', 'secret')
    expect(r.emailError).toBe('Enter a valid email address.')
  })

  it('rejects empty password', () => {
    const r = validateLoginFields('user@example.com', '')
    expect(r.passwordError).toBe('Password is required.')
  })

  it('reports both errors independently', () => {
    const r = validateLoginFields('', '')
    expect(r.emailError).not.toBeNull()
    expect(r.passwordError).not.toBeNull()
  })

  it('accepts any non-empty password (no length rule at login)', () => {
    const r = validateLoginFields('user@example.com', 'x')
    expect(r.passwordError).toBeNull()
  })
})
