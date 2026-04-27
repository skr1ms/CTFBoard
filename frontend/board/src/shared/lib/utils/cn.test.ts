import { describe, expect, it } from 'vitest'
import { cn } from './cn'

describe('cn', () => {
  it('joins string classes', () => {
    expect(cn('foo', 'bar')).toBe('foo bar')
  })

  it('ignores falsy values', () => {
    expect(cn('foo', false, null, undefined, '', 'bar')).toBe('foo bar')
  })

  it('includes keys where value is true', () => {
    expect(cn({ active: true, disabled: false, hidden: true })).toBe('active hidden')
  })

  it('combines strings and objects', () => {
    expect(cn('base', { active: true, disabled: false })).toBe('base active')
  })

  it('returns empty string with no truthy args', () => {
    expect(cn(false, null, undefined)).toBe('')
  })

  it('handles empty object', () => {
    expect(cn({})).toBe('')
  })
})
