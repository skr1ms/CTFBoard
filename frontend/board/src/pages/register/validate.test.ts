import type { FieldResponse } from '@/features/auth/useFields'
import { describe, expect, it } from 'vitest'
import { validateRegisterFields } from './index'

const ok = {
  username: 'alice_123',
  email: 'alice@example.com',
  password: 'Password1',
  confirm: 'Password1',
  fields: [] as FieldResponse[],
  values: {} as Record<string, string>,
}

const run = (overrides: Partial<typeof ok> = {}) => {
  const p = { ...ok, ...overrides }
  return validateRegisterFields(p.username, p.email, p.password, p.confirm, p.fields, p.values)
}

describe('validateRegisterFields - happy path', () => {
  it('returns empty errors for valid input', () => {
    expect(run()).toEqual({})
  })
})

describe('username', () => {
  it('required', () => expect(run({ username: '' }).username).toBeTruthy())
  it('too long (51 chars)', () => expect(run({ username: 'a'.repeat(51) }).username).toBeTruthy())
  it('invalid chars (space)', () => expect(run({ username: 'alice smith' }).username).toBeTruthy())
  it('accepts letters/digits/underscore/dash', () =>
    expect(run({ username: 'Alice-1_2' }).username).toBeUndefined())
  it('accepts dot and plus (john.doe, a+b)', () => {
    expect(run({ username: 'john.doe' }).username).toBeUndefined()
    expect(run({ username: 'a+b' }).username).toBeUndefined()
  })
  it('exactly 1 char is valid', () => expect(run({ username: 'a' }).username).toBeUndefined())
  it('exactly 50 chars', () => expect(run({ username: 'a'.repeat(50) }).username).toBeUndefined())
})

describe('email', () => {
  it('required', () => expect(run({ email: '' }).email).toBeTruthy())
  it('rejects missing @', () => expect(run({ email: 'notanemail' }).email).toBeTruthy())
  it('accepts valid email', () => expect(run({ email: 'x@y.io' }).email).toBeUndefined())
})

describe('password', () => {
  it('required', () => expect(run({ password: '', confirm: '' }).password).toBeTruthy())
  it('too short (5 chars)', () =>
    expect(run({ password: 'Ab1cd', confirm: 'Ab1cd' }).password).toBeTruthy())
  it('too short (7 chars)', () =>
    expect(run({ password: 'Ab1cdef', confirm: 'Ab1cdef' }).password).toBeTruthy())
  it('missing uppercase', () =>
    expect(run({ password: 'password1', confirm: 'password1' }).password).toBeTruthy())
  it('missing lowercase', () =>
    expect(run({ password: 'PASSWORD1', confirm: 'PASSWORD1' }).password).toBeTruthy())
  it('missing digit', () =>
    expect(run({ password: 'PasswordX', confirm: 'PasswordX' }).password).toBeTruthy())
  it('accepts 8-char password with all char classes', () =>
    expect(run({ password: 'Ab1cdefg', confirm: 'Ab1cdefg' }).password).toBeUndefined())
  it('accepts longer password', () =>
    expect(run({ password: 'Password1', confirm: 'Password1' }).password).toBeUndefined())
})

describe('confirm', () => {
  it('required', () => expect(run({ confirm: '' }).confirm).toBeTruthy())
  it('mismatch', () => expect(run({ confirm: 'other_pass' }).confirm).toBeTruthy())
  it('passes when matches', () => expect(run({ confirm: ok.password }).confirm).toBeUndefined())
})

describe('custom fields', () => {
  const req = { id: 'f1', name: 'City', required: true, field_type: 'text' as const }

  it('errors when required field is empty', () => {
    const errs = run({ fields: [req], values: {} })
    expect(errs['f1']).toBeTruthy()
  })

  it('passes when required field is filled', () => {
    const errs = run({ fields: [req], values: { f1: 'London' } })
    expect(errs['f1']).toBeUndefined()
  })

  it('ignores optional fields', () => {
    const opt = { ...req, required: false }
    expect(run({ fields: [opt], values: {} })['f1']).toBeUndefined()
  })
})
