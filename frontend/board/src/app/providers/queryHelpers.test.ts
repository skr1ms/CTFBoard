import { describe, expect, it } from 'vitest'
import { extractStatus, is4xx } from './QueryProvider'

describe('is4xx', () => {
  it('returns true for 400', () => expect(is4xx(400)).toBe(true))
  it('returns true for 404', () => expect(is4xx(404)).toBe(true))
  it('returns true for 499', () => expect(is4xx(499)).toBe(true))
  it('returns false for 500', () => expect(is4xx(500)).toBe(false))
  it('returns false for 399', () => expect(is4xx(399)).toBe(false))
  it('returns false for a string', () => expect(is4xx('404')).toBe(false))
  it('returns false for null', () => expect(is4xx(null)).toBe(false))
})

describe('extractStatus', () => {
  it('extracts numeric status from object', () => {
    expect(extractStatus({ __httpStatus: 404 })).toBe(404)
  })

  it('returns undefined when status is absent', () => {
    expect(extractStatus({ code: 'ERR' })).toBeUndefined()
  })

  it('returns undefined when __httpStatus is a string', () => {
    expect(extractStatus({ __httpStatus: '404' })).toBeUndefined()
  })

  it('returns undefined for null', () => {
    expect(extractStatus(null)).toBeUndefined()
  })

  it('returns undefined for a plain string', () => {
    expect(extractStatus('error')).toBeUndefined()
  })
})
