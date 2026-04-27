import { useAuthStore } from '@/shared/stores/authStore'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import { createElement } from 'react'
import { MemoryRouter, Route, Routes } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { OAuthCallbackPage } from './index'

// ── mocks ─────────────────────────────────────────────────────────────────────

const mockNavigate = vi.fn()

vi.mock('react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router')>()
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  }
})

vi.mock('@/shared/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/shared/api/client')>()
  return {
    ...actual,
    api: { GET: vi.fn(), POST: vi.fn() },
    baseClient: { GET: vi.fn(), POST: vi.fn() },
  }
})

vi.mock('sonner', () => ({
  toast: { error: vi.fn(), success: vi.fn() },
}))

// ── helpers ───────────────────────────────────────────────────────────────────

function makeClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } })
}

function renderPage(initialPath = '/auth/callback') {
  const qc = makeClient()
  return render(
    createElement(
      QueryClientProvider,
      { client: qc },
      createElement(
        MemoryRouter,
        { initialEntries: [initialPath] },
        createElement(
          Routes,
          null,
          createElement(Route, {
            path: '/auth/callback',
            element: createElement(OAuthCallbackPage),
          }),
          createElement(Route, {
            path: '/login',
            element: createElement('div', null, 'login-page'),
          }),
          createElement(Route, {
            path: '/challenges',
            element: createElement('div', null, 'challenges-page'),
          }),
        ),
      ),
    ),
  )
}

// ── setup ─────────────────────────────────────────────────────────────────────

beforeEach(() => {
  vi.clearAllMocks()
  mockNavigate.mockReset()

  useAuthStore.setState({
    user: null,
    accessToken: null,
    isAuthenticated: false,
    isBanned: false,
    isAdmin: false,
    hydrating: false,
  })
})

// ── error in query param ──────────────────────────────────────────────────────

describe('OAuthCallbackPage - query param errors', () => {
  it('shows toast and navigates to /login when ?error=access_denied', async () => {
    const { toast } = await import('sonner')

    renderPage('/auth/callback?error=access_denied')

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith('OAuth login was cancelled')
      expect(mockNavigate).toHaveBeenCalledWith('/login', { replace: true })
    })
  })

  it('shows toast and navigates to /login for unknown error codes', async () => {
    const { toast } = await import('sonner')

    renderPage('/auth/callback?error=SOME_UNKNOWN_ERROR')

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith('OAuth login failed: SOME_UNKNOWN_ERROR')
      expect(mockNavigate).toHaveBeenCalledWith('/login', { replace: true })
    })
  })
})

// ── missing code ──────────────────────────────────────────────────────────────

describe('OAuthCallbackPage - missing exchange code', () => {
  it('shows error and navigates to /login when ?code is absent', async () => {
    const { toast } = await import('sonner')

    renderPage('/auth/callback')

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith('OAuth login failed: missing exchange code')
      expect(mockNavigate).toHaveBeenCalledWith('/login', { replace: true })
    })
  })
})

// ── exchange failure ──────────────────────────────────────────────────────────

describe('OAuthCallbackPage - exchange failure', () => {
  it('shows error and navigates to /login when exchange returns 404', async () => {
    const { baseClient } = await import('@/shared/api/client')
    const { toast } = await import('sonner')

    vi.mocked(baseClient.POST).mockResolvedValue({
      data: null,
      error: { code: 'TOKEN_NOT_FOUND', message: 'code expired' },
    } as never)

    renderPage('/auth/callback?code=expired-code')

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith('OAuth login failed: code expired or already used')
      expect(mockNavigate).toHaveBeenCalledWith('/login', { replace: true })
    })
  })
})

// ── successful login ──────────────────────────────────────────────────────────

describe('OAuthCallbackPage - successful login', () => {
  const mockMe = {
    id: 'u1',
    username: 'alice',
    role: 'user',
    ban_status: null,
  }

  beforeEach(async () => {
    const { baseClient, api } = await import('@/shared/api/client')

    vi.mocked(baseClient.POST).mockResolvedValue({
      data: { access_token: 'acc123', refresh_expires_at: 0, access_expires_at: 0 },
      error: undefined,
    } as never)

    vi.mocked(api.GET).mockResolvedValue({ data: mockMe, error: undefined } as never)
  })

  it('calls /auth/oauth/exchange with the code from query params', async () => {
    const { baseClient } = await import('@/shared/api/client')

    renderPage('/auth/callback?code=one-time-code')

    await waitFor(() => {
      expect(baseClient.POST).toHaveBeenCalledWith('/auth/oauth/exchange', {
        body: { code: 'one-time-code' },
      })
    })
  })

  it('calls /auth/me with the access token after successful exchange', async () => {
    const { api } = await import('@/shared/api/client')

    renderPage('/auth/callback?code=one-time-code')

    await waitFor(() => {
      expect(api.GET).toHaveBeenCalledWith('/auth/me')
    })
  })

  it('logs the user in and navigates to /challenges', async () => {
    renderPage('/auth/callback?code=one-time-code')

    await waitFor(() => {
      const state = useAuthStore.getState()
      expect(state.isAuthenticated).toBe(true)
      expect(state.accessToken).toBe('acc123')
      expect(mockNavigate).toHaveBeenCalledWith('/challenges', { replace: true })
    })
  })

  it('shows a spinner while processing', () => {
    renderPage('/auth/callback?code=one-time-code')
    expect(screen.getByText(/completing sign-in/i)).toBeInTheDocument()
  })
})

// ── /auth/me failure ──────────────────────────────────────────────────────────

describe('OAuthCallbackPage - /auth/me failure', () => {
  beforeEach(async () => {
    const { baseClient } = await import('@/shared/api/client')

    vi.mocked(baseClient.POST).mockResolvedValue({
      data: { access_token: 'acc123', refresh_expires_at: 0, access_expires_at: 0 },
      error: undefined,
    } as never)
  })

  it('shows error and navigates to /login when /auth/me fails', async () => {
    const { api } = await import('@/shared/api/client')
    const { toast } = await import('sonner')
    vi.mocked(api.GET).mockResolvedValue({ data: null, error: { code: 'UNAUTHORIZED' } } as never)

    renderPage('/auth/callback?code=one-time-code')

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith('OAuth login failed: could not fetch user info')
      expect(mockNavigate).toHaveBeenCalledWith('/login', { replace: true })
    })
  })

  it('clears accessToken from store when /auth/me fails', async () => {
    const { api } = await import('@/shared/api/client')
    vi.mocked(api.GET).mockResolvedValue({
      data: undefined,
      error: { code: 'INTERNAL_ERROR' },
    } as never)

    renderPage('/auth/callback?code=one-time-code')

    await waitFor(() => {
      expect(useAuthStore.getState().accessToken).toBeNull()
    })
  })
})
