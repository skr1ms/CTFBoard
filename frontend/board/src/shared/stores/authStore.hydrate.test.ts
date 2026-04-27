/**
 * hydrate() tests run in an isolated module so the module-level
 * _hydrateStarted guard starts fresh for every test.
 *
 * vi.resetModules() + dynamic import gives each test a clean module copy.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

// Mocks are declared at file scope and configured per-test via vi.mocked().
vi.mock('@/shared/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/shared/api/client')>()
  return {
    ...actual,
    baseClient: { POST: vi.fn() },
    api: { GET: vi.fn() },
  }
})

// ── helpers ───────────────────────────────────────────────────────────────────

const SESSION_HINT_KEY = 'ctf_has_session'

// Re-import authStore fresh for every test so _hydrateStarted resets to false.
async function freshStore() {
  vi.resetModules()
  // vi.doMock (non-hoisted) re-applies the mock after resetModules
  vi.doMock('@/shared/api/client', async (importOriginal) => {
    const actual = await importOriginal<typeof import('@/shared/api/client')>()
    return {
      ...actual,
      baseClient: { POST: vi.fn() },
      api: { GET: vi.fn() },
    }
  })
  const { useAuthStore: store } = await import('./authStore')
  return store
}

// ── tests ─────────────────────────────────────────────────────────────────────

beforeEach(() => {
  localStorage.clear()
  vi.clearAllMocks()
})

afterEach(() => {
  localStorage.clear()
})

describe('hydrate() - no session hint', () => {
  it('skips /auth/refresh and sets hydrating=false immediately when hint is absent', async () => {
    const store = await freshStore()
    const { baseClient: bc } = await import('@/shared/api/client')

    store.getState().hydrate()

    await vi.waitFor(() => expect(store.getState().hydrating).toBe(false))

    expect(bc.POST).not.toHaveBeenCalled()
    expect(store.getState().isAuthenticated).toBe(false)
    expect(localStorage.getItem(SESSION_HINT_KEY)).toBeNull()
  })
})

describe('hydrate() - no refresh token', () => {
  it('sets isAuthenticated=false and clears hint when server rejects cookie', async () => {
    const store = await freshStore()
    localStorage.setItem(SESSION_HINT_KEY, '1')

    const { baseClient: bc } = await import('@/shared/api/client')
    // Server returns 401 - cookie is absent or expired
    vi.mocked(bc.POST).mockResolvedValue({
      data: undefined,
      error: { code: 'INVALID_TOKEN', message: 'no cookie' },
    } as never)

    store.getState().hydrate()

    await vi.waitFor(() => expect(store.getState().hydrating).toBe(false))

    const s = store.getState()
    expect(s.isAuthenticated).toBe(false)
    expect(s.accessToken).toBeNull()
    // hint must be cleared so the next page load does not probe again
    expect(localStorage.getItem(SESSION_HINT_KEY)).toBeNull()
  })
})

describe('hydrate() - refresh succeeds', () => {
  it('sets accessToken, user and clears hydrating after successful refresh+me', async () => {
    const store = await freshStore()
    localStorage.setItem(SESSION_HINT_KEY, '1')

    // Re-import mocked client for this module instance
    const { baseClient: bc, api: a } = await import('@/shared/api/client')

    vi.mocked(bc.POST).mockResolvedValue({
      data: { access_token: 'new-acc' },
      error: undefined,
    } as never)

    vi.mocked(a.GET).mockResolvedValue({
      data: { id: 'u1', username: 'alice', role: 'user', ban_status: null },
      error: undefined,
    } as never)

    store.getState().hydrate()

    // Wait for async hydration to complete
    await vi.waitFor(() => expect(store.getState().hydrating).toBe(false))

    const s = store.getState()
    expect(s.isAuthenticated).toBe(true)
    expect(s.accessToken).toBe('new-acc')
    expect(s.user?.username).toBe('alice')
    // hint remains set - session is active; refresh token itself is an httpOnly cookie
    expect(localStorage.getItem(SESSION_HINT_KEY)).toBe('1')
  })
})

describe('hydrate() - idempotency', () => {
  it('second call is a no-op - no additional network requests', async () => {
    const store = await freshStore()
    localStorage.setItem(SESSION_HINT_KEY, '1')

    const { baseClient: bc, api: a } = await import('@/shared/api/client')
    vi.mocked(bc.POST).mockResolvedValue({
      data: { access_token: 'acc' },
      error: undefined,
    } as never)
    vi.mocked(a.GET).mockResolvedValue({
      data: { id: 'u1', username: 'alice', role: 'user', ban_status: null },
      error: undefined,
    } as never)

    store.getState().hydrate()
    store.getState().hydrate() // second call - must be no-op

    await vi.waitFor(() => expect(store.getState().hydrating).toBe(false))

    // /auth/refresh should only be called once despite two hydrate() invocations
    expect(bc.POST).toHaveBeenCalledTimes(1)
  })
})

describe('hydrate() - refresh fails', () => {
  it('sets isAuthenticated=false and clears hint when refresh throws', async () => {
    const store = await freshStore()
    localStorage.setItem(SESSION_HINT_KEY, '1')

    const { baseClient: bc } = await import('@/shared/api/client')
    vi.mocked(bc.POST).mockResolvedValue({
      data: undefined,
      error: { code: 'INVALID_TOKEN', message: 'bad refresh' },
    } as never)

    store.getState().hydrate()

    await vi.waitFor(() => expect(store.getState().hydrating).toBe(false))

    const s = store.getState()
    expect(s.isAuthenticated).toBe(false)
    expect(s.accessToken).toBeNull()
    expect(localStorage.getItem(SESSION_HINT_KEY)).toBeNull()
  })
})
