import { describe, expect, it } from 'vitest'
import { getBannerConfig } from './index'

const make = (status: string, extra: Record<string, unknown> = {}) => ({
  status,
  ...extra,
})

describe('getBannerConfig', () => {
  it('returns null when data is null/undefined', () => {
    expect(getBannerConfig(null)).toBeNull()
    expect(getBannerConfig(undefined)).toBeNull()
  })

  it('returns null for unknown status', () => {
    expect(getBannerConfig(make('unknown'))).toBeNull()
  })

  it('not_started -> blue banner', () => {
    const cfg = getBannerConfig(make('not_started'))
    expect(cfg).not.toBeNull()
    expect(cfg?.variant).toBe('blue')
    expect(cfg?.show).toBe(true)
  })

  it('not_started with start_time includes date in message', () => {
    const cfg = getBannerConfig(make('not_started', { start_time: '2030-01-01T12:00:00Z' }))
    expect(cfg?.message(make('not_started', { start_time: '2030-01-01T12:00:00Z' }))).toContain(
      'starts',
    )
  })

  it('not_started without start_time uses fallback message', () => {
    const data = make('not_started')
    const cfg = getBannerConfig(data)
    expect(cfg?.message(data)).toBe('Competition has not started yet')
  })

  it('active without freeze_time -> null (no banner)', () => {
    expect(getBannerConfig(make('active'))).toBeNull()
  })

  it('active with future freeze_time -> null (not frozen yet)', () => {
    const future = new Date(Date.now() + 3_600_000).toISOString()
    expect(getBannerConfig(make('active', { freeze_time: future }))).toBeNull()
  })

  it('active with past freeze_time -> orange frozen banner', () => {
    const past = new Date(Date.now() - 3_600_000).toISOString()
    const cfg = getBannerConfig(make('active', { freeze_time: past }))
    expect(cfg?.variant).toBe('orange')
    expect(cfg?.icon).toBe('🧊')
  })

  it('paused -> yellow banner', () => {
    const cfg = getBannerConfig(make('paused'))
    expect(cfg?.variant).toBe('yellow')
  })

  it('frozen -> orange banner', () => {
    const cfg = getBannerConfig(make('frozen'))
    expect(cfg?.variant).toBe('orange')
    expect(cfg?.icon).toBe('🧊')
  })

  it('ended -> gray banner', () => {
    const cfg = getBannerConfig(make('ended'))
    expect(cfg?.variant).toBe('gray')
    expect(cfg?.icon).toBe('🏁')
  })

  it('ended with end_time includes date in message', () => {
    const data = make('ended', { end_time: '2024-01-01T10:00:00Z' })
    const cfg = getBannerConfig(data)
    expect(cfg?.message(data)).toContain('ended')
  })
})
