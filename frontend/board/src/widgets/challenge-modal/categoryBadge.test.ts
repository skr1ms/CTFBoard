import { describe, expect, it } from 'vitest'
import { categoryBadgeVariant } from './ChallengeModal'

describe('categoryBadgeVariant', () => {
  it('returns default for undefined', () => {
    expect(categoryBadgeVariant(undefined)).toBe('default')
  })

  it('returns default for unknown category', () => {
    expect(categoryBadgeVariant('steganography')).toBe('default')
  })

  it('is case-insensitive', () => {
    expect(categoryBadgeVariant('Web')).toBe('blue')
    expect(categoryBadgeVariant('WEB')).toBe('blue')
    expect(categoryBadgeVariant('web')).toBe('blue')
  })

  it.each([
    ['web', 'blue'],
    ['network', 'blue'],
    ['crypto', 'purple'],
    ['blockchain', 'purple'],
    ['pwn', 'red'],
    ['rev', 'yellow'],
    ['forensics', 'cyan'],
    ['osint', 'cyan'],
    ['misc', 'green'],
    ['mobile', 'green'],
  ])('%s -> %s', (cat, expected) => {
    expect(categoryBadgeVariant(cat)).toBe(expected)
  })
})
