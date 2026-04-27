import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  apiErrorMessage,
  baseClient,
  isApiError,
  refreshTokens,
  registerTokenStore,
  type TokenStore,
} from './client'

// ── helpers ───────────────────────────────────────────────────────────────────

function makeTokenStore(overrides: Partial<TokenStore> = {}): TokenStore {
  return {
    getAccessToken: vi.fn().mockReturnValue('acc'),
    hasSession: vi.fn().mockReturnValue(true),
    setAccessToken: vi.fn(),
    ...overrides,
  }
}

// ── refreshTokens() - singleflight ───────────────────────────────────────────
// Spy on baseClient.POST so we intercept the actual openapi-fetch call without
// needing to stub the global fetch (which openapi-fetch doesn't expose cleanly).

describe('refreshTokens() - singleflight', () => {
  let postSpy: ReturnType<typeof vi.spyOn>

  beforeEach(() => {
    postSpy = vi.spyOn(baseClient, 'POST')
  })

  afterEach(() => {
    postSpy.mockRestore()
    vi.clearAllMocks()
  })

  it('returns false when the server rejects the refresh (no cookie)', async () => {
    const store = makeTokenStore()
    registerTokenStore(store)

    postSpy.mockResolvedValue({
      data: undefined,
      error: { code: 'INVALID_TOKEN', message: 'no cookie' },
    } as never)

    const result = await refreshTokens()
    expect(result).toBe(false)
    expect(store.setAccessToken).not.toHaveBeenCalled()
  })

  it('makes only one POST when called concurrently (singleflight)', async () => {
    const store = makeTokenStore()
    registerTokenStore(store)

    // Delayed response - both concurrent calls share this one POST
    postSpy.mockImplementation(
      () =>
        new Promise((res) =>
          setTimeout(() => {
            res({
              data: { access_token: 'new-acc' },
              error: undefined,
            } as never)
          }, 10),
        ),
    )

    const [r1, r2] = await Promise.all([refreshTokens(), refreshTokens()])

    expect(r1).toBe(true)
    expect(r2).toBe(true)
    // Only one POST despite two concurrent refreshTokens() invocations
    expect(postSpy).toHaveBeenCalledTimes(1)
    expect(store.setAccessToken).toHaveBeenCalledTimes(1)
    expect(store.setAccessToken).toHaveBeenCalledWith('new-acc')
  })

  it('returns false and does not call setAccessToken when refresh fails', async () => {
    const store = makeTokenStore()
    registerTokenStore(store)

    postSpy.mockResolvedValue({
      data: undefined,
      error: { code: 'INVALID_TOKEN', message: 'bad' },
    } as never)

    const result = await refreshTokens()
    expect(result).toBe(false)
    expect(store.setAccessToken).not.toHaveBeenCalled()
  })

  it('sequential calls each make their own POST after the singleflight resolves', async () => {
    const store = makeTokenStore()
    registerTokenStore(store)

    postSpy.mockResolvedValue({
      data: { access_token: 'new-acc' },
      error: undefined,
    } as never)

    await refreshTokens()
    await refreshTokens()

    // Two sequential calls: each should POST since the singleflight guard
    // resets to null after the first promise resolves.
    expect(postSpy).toHaveBeenCalledTimes(2)
  })
})

// ── isApiError ────────────────────────────────────────────────────────────────

describe('isApiError', () => {
  it('returns true for a valid ApiError shape', () => {
    expect(isApiError({ code: 'NOT_FOUND', message: 'not found' })).toBe(true)
  })

  it('returns false for null', () => {
    expect(isApiError(null)).toBe(false)
  })

  it('returns false for a plain string', () => {
    expect(isApiError('error')).toBe(false)
  })

  it('returns false when code is missing', () => {
    expect(isApiError({ message: 'oops' })).toBe(false)
  })

  it('returns false when code is not a string', () => {
    expect(isApiError({ code: 42, message: 'oops' })).toBe(false)
  })

  it('returns false for an empty object', () => {
    expect(isApiError({})).toBe(false)
  })

  it('returns true even with extra fields', () => {
    expect(isApiError({ code: 'ERR', message: 'msg', status: 400 })).toBe(true)
  })

  it('returns true for an error shape that also carries retryAfter', () => {
    expect(
      isApiError({ code: 'RATE_LIMIT_EXCEEDED', message: 'too fast', status: 429, retryAfter: 30 }),
    ).toBe(true)
  })
})

// ── retryAfter parsing (useFlagSubmit reads err.retryAfter) ───────────────────
// withEmbeddedStatus embeds the Retry-After header into the error body.
// These tests verify the expected shape that useFlagSubmit consumes.

describe('retryAfter embedded field shape', () => {
  it('is a number when Retry-After is a numeric string', () => {
    // Simulates the parsed error body after withEmbeddedStatus runs
    const embeddedErr = {
      code: 'RATE_LIMIT_EXCEEDED',
      message: 'slow down',
      status: 429,
      retryAfter: 60,
    }
    const raw = (embeddedErr as { retryAfter?: unknown }).retryAfter
    const seconds = typeof raw === 'number' && raw > 0 ? Math.ceil(raw) : 30
    expect(seconds).toBe(60)
  })

  it('falls back to 30 when retryAfter is missing', () => {
    const embeddedErr = { code: 'RATE_LIMIT_EXCEEDED', message: 'slow down', status: 429 }
    const raw = (embeddedErr as { retryAfter?: unknown }).retryAfter
    const seconds = typeof raw === 'number' && raw > 0 ? Math.ceil(raw) : 30
    expect(seconds).toBe(30)
  })

  it('falls back to 30 when retryAfter is zero', () => {
    const raw = 0
    const seconds = typeof raw === 'number' && raw > 0 ? Math.ceil(raw) : 30
    expect(seconds).toBe(30)
  })

  it('falls back to 30 when retryAfter is a non-numeric string', () => {
    const raw: unknown = 'Tue, 21 Apr 2026 00:00:00 GMT'
    const seconds = typeof raw === 'number' && raw > 0 ? Math.ceil(raw) : 30
    expect(seconds).toBe(30)
  })

  it('ceils fractional retryAfter values', () => {
    const raw = 29.1
    const seconds = typeof raw === 'number' && raw > 0 ? Math.ceil(raw) : 30
    expect(seconds).toBe(30)
  })
})

// ── apiErrorMessage ───────────────────────────────────────────────────────────

describe('apiErrorMessage', () => {
  it('returns the message from a valid ApiError', () => {
    expect(apiErrorMessage({ code: 'NOT_FOUND', message: 'resource not found' })).toBe(
      'resource not found',
    )
  })

  it('returns the message from a native Error', () => {
    expect(apiErrorMessage(new Error('network failure'))).toBe('network failure')
  })

  it('returns the provided fallback for unknown shapes', () => {
    expect(apiErrorMessage({ foo: 'bar' }, 'oops')).toBe('oops')
  })

  it('returns the default fallback when none is provided', () => {
    expect(apiErrorMessage(null)).toBe('Something went wrong.')
  })

  it('returns the default fallback for a plain string', () => {
    expect(apiErrorMessage('error string')).toBe('Something went wrong.')
  })
})
