import { useAuthStore } from '@/shared/stores/authStore'
import { render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'
import { MemoryRouter, Route, Routes } from 'react-router'
import { AdminGuard } from './AdminGuard'
import { AuthGuard } from './AuthGuard'
import { TeamGuard } from './TeamGuard'

// ── helpers ──────────────────────────────────────────────────────────────────

function setAuth(overrides: Partial<ReturnType<typeof useAuthStore.getState>>) {
  useAuthStore.setState({
    user: null,
    accessToken: null,
    isAdmin: false,
    isAuthenticated: false,
    isBanned: false,
    hydrating: false,
    ...overrides,
  })
}

function makeUser(overrides: Record<string, unknown> = {}) {
  return {
    id: 'u1',
    username: 'alice',
    role: 'user',
    ban_status: null,
    team_id: null,
    ...overrides,
  } as never
}

// ── setup ─────────────────────────────────────────────────────────────────────

beforeEach(() => {
  setAuth({})
})

// ── AuthGuard ─────────────────────────────────────────────────────────────────

describe('AuthGuard', () => {
  it('shows spinner while hydrating - does not render children', () => {
    setAuth({ hydrating: true })
    render(
      <MemoryRouter>
        <AuthGuard>
          <div>protected</div>
        </AuthGuard>
      </MemoryRouter>,
    )
    expect(screen.queryByText('protected')).not.toBeInTheDocument()
  })

  it('redirects to /login when not authenticated', () => {
    setAuth({ isAuthenticated: false, hydrating: false })
    render(
      <MemoryRouter initialEntries={['/challenges']}>
        <Routes>
          <Route path="/login" element={<div>login-page</div>} />
          <Route
            path="/challenges"
            element={
              <AuthGuard>
                <div>protected</div>
              </AuthGuard>
            }
          />
        </Routes>
      </MemoryRouter>,
    )
    expect(screen.getByText('login-page')).toBeInTheDocument()
    expect(screen.queryByText('protected')).not.toBeInTheDocument()
  })

  it('redirects banned user to /banned', () => {
    setAuth({ isAuthenticated: true, isBanned: true, hydrating: false })
    render(
      <MemoryRouter initialEntries={['/challenges']}>
        <Routes>
          <Route path="/banned" element={<div>banned-page</div>} />
          <Route
            path="/challenges"
            element={
              <AuthGuard>
                <div>protected</div>
              </AuthGuard>
            }
          />
        </Routes>
      </MemoryRouter>,
    )
    expect(screen.getByText('banned-page')).toBeInTheDocument()
    expect(screen.queryByText('protected')).not.toBeInTheDocument()
  })

  it('renders children for authenticated non-banned user', () => {
    setAuth({ isAuthenticated: true, isBanned: false, hydrating: false })
    render(
      <MemoryRouter>
        <AuthGuard>
          <div>protected</div>
        </AuthGuard>
      </MemoryRouter>,
    )
    expect(screen.getByText('protected')).toBeInTheDocument()
  })

  it('allows banned user to access /banned without redirect loop', () => {
    setAuth({ isAuthenticated: true, isBanned: true, hydrating: false })
    render(
      <MemoryRouter initialEntries={['/banned']}>
        <Routes>
          <Route
            path="/banned"
            element={
              <AuthGuard>
                <div>banned-content</div>
              </AuthGuard>
            }
          />
        </Routes>
      </MemoryRouter>,
    )
    expect(screen.getByText('banned-content')).toBeInTheDocument()
  })
})

// ── TeamGuard ─────────────────────────────────────────────────────────────────

describe('TeamGuard', () => {
  it('shows spinner when authenticated but user not loaded yet', () => {
    setAuth({ isAuthenticated: true, user: null })
    render(
      <MemoryRouter>
        <TeamGuard>
          <div>team-content</div>
        </TeamGuard>
      </MemoryRouter>,
    )
    expect(screen.queryByText('team-content')).not.toBeInTheDocument()
  })

  it('redirects to /team/enroll when user has no team', () => {
    setAuth({ isAuthenticated: true, user: makeUser({ team_id: null }) })
    render(
      <MemoryRouter initialEntries={['/team']}>
        <Routes>
          <Route path="/team/enroll" element={<div>enroll-page</div>} />
          <Route
            path="/team"
            element={
              <TeamGuard>
                <div>team-content</div>
              </TeamGuard>
            }
          />
        </Routes>
      </MemoryRouter>,
    )
    expect(screen.getByText('enroll-page')).toBeInTheDocument()
    expect(screen.queryByText('team-content')).not.toBeInTheDocument()
  })

  it('renders children when user has a team', () => {
    setAuth({ isAuthenticated: true, user: makeUser({ team_id: 'team-uuid' }) })
    render(
      <MemoryRouter>
        <TeamGuard>
          <div>team-content</div>
        </TeamGuard>
      </MemoryRouter>,
    )
    expect(screen.getByText('team-content')).toBeInTheDocument()
  })
})

// ── AdminGuard ────────────────────────────────────────────────────────────────

describe('AdminGuard', () => {
  it('shows spinner while hydrating', () => {
    setAuth({ isAdmin: true, hydrating: true })
    render(
      <MemoryRouter>
        <AdminGuard>
          <div>admin-content</div>
        </AdminGuard>
      </MemoryRouter>,
    )
    expect(screen.queryByText('admin-content')).not.toBeInTheDocument()
  })

  it('renders children for admin user', () => {
    setAuth({ isAuthenticated: true, isAdmin: true, hydrating: false })
    render(
      <MemoryRouter>
        <AdminGuard>
          <div>admin-content</div>
        </AdminGuard>
      </MemoryRouter>,
    )
    expect(screen.getByText('admin-content')).toBeInTheDocument()
  })

  it('renders 403 Forbidden for non-admin (does not render children)', () => {
    setAuth({ isAuthenticated: true, isAdmin: false, hydrating: false })
    render(
      <MemoryRouter>
        <AdminGuard>
          <div>admin-content</div>
        </AdminGuard>
      </MemoryRouter>,
    )
    expect(screen.queryByText('admin-content')).not.toBeInTheDocument()
    expect(screen.getByText('Access Denied')).toBeInTheDocument()
  })
})
