import { api, baseClient, registerTokenStore, type TokenStore } from '@/shared/api/client'
import type { components } from '@/shared/api/schema.d'
import { create } from 'zustand'

type MeResponse = components['schemas']['MeResponse']

// Guard against HMR / React Strict-Mode double-invoke
let _hydrateStarted = false

/**
 * Non-sensitive sentinel flag in localStorage that mirrors whether the browser
 * is *expected* to have a refresh cookie. The actual refresh token lives in an
 * httpOnly cookie that is invisible to JS, so without a hint the SPA would
 * blindly POST /auth/refresh on every load and get a 401 for guest sessions.
 * The flag is set on login(), cleared on logout() and on any hydrate failure.
 */
const SESSION_HINT_KEY = 'ctf_has_session'

function setSessionHint(): void {
  try {
    localStorage.setItem(SESSION_HINT_KEY, '1')
  } catch {
    // ignore quota / privacy mode
  }
}

function clearSessionHint(): void {
  try {
    localStorage.removeItem(SESSION_HINT_KEY)
  } catch {
    // ignore
  }
}

function hasSessionHint(): boolean {
  try {
    return localStorage.getItem(SESSION_HINT_KEY) === '1'
  } catch {
    return false
  }
}

export interface AuthState {
  user: MeResponse | null
  accessToken: string | null
  isAdmin: boolean
  isAuthenticated: boolean
  isBanned: boolean
  /** True while the initial token refresh + /auth/me is in flight. Guards show a
   * spinner instead of redirecting so there are no flash-of-redirect artefacts. */
  hydrating: boolean

  login(accessToken: string, user: MeResponse): void
  logout(): Promise<void>
  setUser(user: MeResponse): void
  setAccessToken(token: string): void
  hydrate(): void
}

export const useAuthStore = create<AuthState>((set, get) => ({
  user: null,
  accessToken: null,
  isAdmin: false,
  isAuthenticated: false,
  isBanned: false,
  hydrating: true,

  login(accessToken, user) {
    setSessionHint()
    set({
      accessToken,
      user,
      isAdmin: user.role === 'admin',
      isBanned: user.ban_status?.is_banned ?? false,
      isAuthenticated: true,
      hydrating: false,
    })
  },

  async logout() {
    if (!get().isAuthenticated) return

    // Best-effort call - the server clears the httpOnly refresh cookie.
    try {
      await baseClient.POST('/auth/logout')
    } catch {
      // ignore
    }

    // Clear query cache BEFORE resetting state so a fast re-login cannot have
    // its freshly loaded data wiped by a deferred clear() firing afterward.
    try {
      const { queryClient } = await import('@/app/providers/QueryProvider')
      queryClient.clear()
    } catch {
      // ignore
    }

    clearSessionHint()
    set({
      accessToken: null,
      user: null,
      isAdmin: false,
      isBanned: false,
      isAuthenticated: false,
      hydrating: false,
    })
  },

  setUser(user) {
    set({ user, isAdmin: user.role === 'admin', isBanned: user.ban_status?.is_banned ?? false })
  },

  setAccessToken(token) {
    set({ accessToken: token })
  },

  hydrate() {
    if (_hydrateStarted) return
    _hydrateStarted = true

    const clearState = () => {
      clearSessionHint()
      set({
        user: null,
        accessToken: null,
        isAdmin: false,
        isBanned: false,
        isAuthenticated: false,
        hydrating: false,
      })
    }

    // Skip the refresh probe if the browser has never successfully logged in.
    // The httpOnly refresh cookie is invisible to JS, so we track a non-sensitive
    // localStorage hint instead of probing blindly.
    if (!hasSessionHint()) {
      set({ hydrating: false })
      return
    }

    void (async () => {
      try {
        const { data: tokens, error: refreshErr } = await baseClient.POST('/auth/refresh')
        if (refreshErr || !tokens?.access_token) {
          clearState()
          return
        }
        set({ accessToken: tokens.access_token })

        const { data: me, error: meErr } = await api.GET('/auth/me')
        if (meErr || !me) {
          await get().logout()
          return
        }
        set({
          user: me,
          isAdmin: me.role === 'admin',
          isBanned: me.ban_status?.is_banned ?? false,
          isAuthenticated: true,
          hydrating: false,
        })
      } catch {
        clearState()
      }
    })()
  },
}))

// ---------------------------------------------------------------------------
// Wire the token store into the API client (called once at module load)
// ---------------------------------------------------------------------------

const tokenStore: TokenStore = {
  getAccessToken: () => useAuthStore.getState().accessToken,
  hasSession: () => useAuthStore.getState().isAuthenticated,
  setAccessToken(accessToken) {
    useAuthStore.getState().setAccessToken(accessToken)
  },
}

registerTokenStore(tokenStore)
