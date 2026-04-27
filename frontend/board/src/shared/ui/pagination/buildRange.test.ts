import { describe, expect, it } from 'vitest'
import { buildRange } from './index'

describe('buildRange', () => {
  it('returns all pages when total <= 7', () => {
    expect(buildRange(1, 5)).toEqual([1, 2, 3, 4, 5])
    expect(buildRange(3, 7)).toEqual([1, 2, 3, 4, 5, 6, 7])
  })

  it('adds trailing ellipsis when current is near start', () => {
    const r = buildRange(1, 10)
    expect(r[0]).toBe(1)
    expect(r).toContain('...')
    expect(r[r.length - 1]).toBe(10)
  })

  it('adds leading ellipsis when current is near end', () => {
    const r = buildRange(10, 10)
    expect(r[0]).toBe(1)
    expect(r).toContain('...')
    expect(r[r.length - 1]).toBe(10)
  })

  it('adds both ellipses when current is in the middle', () => {
    const r = buildRange(5, 10)
    expect(r[0]).toBe(1)
    expect(r[r.length - 1]).toBe(10)
    expect(r.filter((x) => x === '...').length).toBe(2)
  })

  it('always includes first and last page', () => {
    for (const cur of [1, 3, 5, 8, 10]) {
      const r = buildRange(cur, 10)
      expect(r[0]).toBe(1)
      expect(r[r.length - 1]).toBe(10)
    }
  })

  it('includes current page and its neighbours', () => {
    const r = buildRange(5, 10)
    expect(r).toContain(4)
    expect(r).toContain(5)
    expect(r).toContain(6)
  })
})
