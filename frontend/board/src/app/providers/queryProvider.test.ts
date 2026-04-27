import { describe, expect, it } from 'vitest'
import { extractStatus, is4xx } from './QueryProvider'

// Tests for is4xx and extractStatus are in queryHelpers.test.ts.
// This file tests additional helpers and documents expected behaviour.

describe('extractStatus - edge cases', () => {
  it('extracts status from an embedded error body', () => {
    // withEmbeddedStatus merges { __httpStatus: httpCode } into the JSON body.
    // extractStatus must read it back correctly.
    const embeddedError = { code: 'NOT_FOUND', message: 'not found', __httpStatus: 404 }
    expect(extractStatus(embeddedError)).toBe(404)
  })

  it('returns undefined for a non-error object (no status field)', () => {
    expect(extractStatus({ code: 'OK' })).toBeUndefined()
  })
})

describe('is4xx - classification', () => {
  it('covers the full 4xx band', () => {
    for (let s = 400; s < 500; s++) {
      expect(is4xx(s)).toBe(true)
    }
  })

  it('does not classify 3xx as client error', () => {
    expect(is4xx(301)).toBe(false)
    expect(is4xx(302)).toBe(false)
  })

  it('does not classify 5xx as client error', () => {
    expect(is4xx(500)).toBe(false)
    expect(is4xx(503)).toBe(false)
  })
})
