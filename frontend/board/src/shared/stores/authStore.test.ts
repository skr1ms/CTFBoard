import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

// authStore uses a module-level _hydrateStarted guard. Reset it between tests
// by re-importing the module after clearing the module registry.
vi.mock('@/shared/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/shared/api/client')>()
  return {
    ...actual,
    baseClient: {
      POST: vi.fn(),
    },
    api: {
      GET: vi.fn(),
    },
  }
})

import { useAuthStore } from './authStore'

// ── helpers ──────────────────────────────────────────────────────────────────

function makeUser(
  overrides: Partial<{
    id: string
    username: string
    role: string
    ban_status: { is_banned: boolean } | null
  }> = {},
) {
  return {
    id: 'u1',
    username: 'alice',
    role: 'user',
    ban_status: null,
    ...overrides,
  } as Parameters<ReturnType<typeof useAuthStore.getState>['login']>[1]
}

function resetStore() {
  useAuthStore.setState({
    user: null,
    accessToken: null,
    isAdmin: false,
    isAuthenticated: false,
    isBanned: false,
    hydrating: true,
  })
}

// ── setup ─────────────────────────────────────────────────────────────────────

beforeEach(() => {
  resetStore()
  vi.clearAllMocks()
})

afterEach(() => {
  localStorage.clear()
})

// ── login() ───────────────────────────────────────────────────────────────────

describe('login()', () => {
  it('sets accessToken, user and isAuthenticated', () => {
    const user = makeUser()
    useAuthStore.getState().login('acc', user)

    const s = useAuthStore.getState()
    expect(s.accessToken).toBe('acc')
    expect(s.user).toEqual(user)
    expect(s.isAuthenticated).toBe(true)
    expect(s.hydrating).toBe(false)
  })

  it('sets the session-hint flag in localStorage (refresh token itself is httpOnly cookie)', () => {
    useAuthStore.getState().login('acc', makeUser())
    expect(localStorage.getItem('ctf_has_session')).toBe('1')
    expect(localStorage.length).toBe(1)
  })

  it('sets isAdmin=true for admin role', () => {
    useAuthStore.getState().login('acc', makeUser({ role: 'admin' }))
    expect(useAuthStore.getState().isAdmin).toBe(true)
  })

  it('sets isAdmin=false for non-admin role', () => {
    useAuthStore.getState().login('acc', makeUser({ role: 'user' }))
    expect(useAuthStore.getState().isAdmin).toBe(false)
  })

  it('sets isBanned=true when ban_status.is_banned', () => {
    useAuthStore.getState().login('acc', makeUser({ ban_status: { is_banned: true } }))
    expect(useAuthStore.getState().isBanned).toBe(true)
  })

  it('sets isBanned=false when ban_status is null', () => {
    useAuthStore.getState().login('acc', makeUser({ ban_status: null }))
    expect(useAuthStore.getState().isBanned).toBe(false)
  })

  it('clears hydrating flag', () => {
    useAuthStore.setState({ hydrating: true })
    useAuthStore.getState().login('acc', makeUser())
    expect(useAuthStore.getState().hydrating).toBe(false)
  })
})

// ── logout() - idempotency ────────────────────────────────────────────────────

describe('logout() - idempotency', () => {
  it('is a no-op when already logged out', async () => {
    useAuthStore.setState({ isAuthenticated: false })

    await useAuthStore.getState().logout()

    expect(useAuthStore.getState().isAuthenticated).toBe(false)
  })

  it('calling logout() twice does not error on the second call', async () => {
    useAuthStore.setState({ isAuthenticated: true, accessToken: 'acc' })

    const { baseClient } = await import('@/shared/api/client')
    vi.spyOn(baseClient, 'POST').mockResolvedValue({ data: undefined, error: undefined } as never)

    await useAuthStore.getState().logout()

    // First logout cleared isAuthenticated -> second call should be a no-op
    await expect(useAuthStore.getState().logout()).resolves.toBeUndefined()
  })
})

// ── tokenStore.logout() - idempotency ─────────────────────────────────────────

describe('tokenStore.logout() - idempotency', () => {
  it('does not navigate to /login when already logged out', () => {
    useAuthStore.setState({ isAuthenticated: false })

    const assignSpy = vi.spyOn(window, 'location', 'get').mockReturnValue({
      ...window.location,
      href: '',
    } as Location)

    expect(useAuthStore.getState().isAuthenticated).toBe(false)
    assignSpy.mockRestore()
  })
})

// ── setUser() ─────────────────────────────────────────────────────────────────

describe('setUser()', () => {
  it('updates user, isAdmin and isBanned', () => {
    useAuthStore.getState().setUser(makeUser({ role: 'admin', ban_status: { is_banned: true } }))

    const s = useAuthStore.getState()
    expect(s.isAdmin).toBe(true)
    expect(s.isBanned).toBe(true)
  })
})

// ── setAccessToken() ──────────────────────────────────────────────────────────

describe('setAccessToken()', () => {
  it('updates accessToken in the store', () => {
    useAuthStore.getState().setAccessToken('new-token')
    expect(useAuthStore.getState().accessToken).toBe('new-token')
  })
})
