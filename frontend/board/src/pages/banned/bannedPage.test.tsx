import { useAuthStore } from '@/shared/stores/authStore'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { MemoryRouter } from 'react-router'
import { BannedPage } from './index'

// ── mocks ─────────────────────────────────────────────────────────────────────

// Stub useMyAppeals and useCreateAppeal so the page doesn't need a live API
vi.mock('@/features/banned/useAppeal', () => ({
  useMyAppeals: () => ({ data: [], isLoading: false }),
  useCreateAppeal: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
}))

// ── helpers ───────────────────────────────────────────────────────────────────

function makeClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } })
}

function renderPage() {
  return render(
    <QueryClientProvider client={makeClient()}>
      <MemoryRouter>
        <BannedPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

// ── setup ─────────────────────────────────────────────────────────────────────

beforeEach(() => {
  useAuthStore.setState({
    user: {
      id: 'u1',
      username: 'banned_user',
      role: 'user',
      ban_status: {
        is_banned: true,
        reason: 'E2E banned fixture',
        banned_at: new Date().toISOString(),
        can_appeal: true,
        has_pending_appeal: false,
      },
    } as never,
    isAuthenticated: true,
    isBanned: true,
    isAdmin: false,
    accessToken: 'acc',
    hydrating: false,
  })
})

// ── inline validation error ───────────────────────────────────────────────────

describe('BannedPage - appeal form validation', () => {
  it('does not show error when textarea is empty', () => {
    renderPage()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('shows "at least 20 characters" error when message is too short', async () => {
    const user = userEvent.setup()
    renderPage()

    const textarea = screen.getByPlaceholderText(/explain why/i)
    await user.type(textarea, 'short')

    expect(screen.getByRole('alert')).toBeInTheDocument()
    expect(screen.getByText(/at least 20 characters/i)).toBeInTheDocument()
  })

  it('hides error once message reaches 20 characters', async () => {
    const user = userEvent.setup()
    renderPage()

    const textarea = screen.getByPlaceholderText(/explain why/i)
    await user.type(textarea, 'A'.repeat(20))

    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('submit button is disabled when message is too short', async () => {
    const user = userEvent.setup()
    renderPage()

    const textarea = screen.getByPlaceholderText(/explain why/i)
    await user.type(textarea, 'too short')

    const btn = screen.getByRole('button', { name: /submit appeal/i })
    expect(btn).toBeDisabled()
  })

  it('submit button is enabled when message is long enough', async () => {
    const user = userEvent.setup()
    renderPage()

    const textarea = screen.getByPlaceholderText(/explain why/i)
    await user.type(textarea, 'This is a valid appeal message.')

    const btn = screen.getByRole('button', { name: /submit appeal/i })
    expect(btn).not.toBeDisabled()
  })
})

// ── ban card content ──────────────────────────────────────────────────────────

describe('BannedPage - ban card', () => {
  it('shows ban reason', () => {
    renderPage()
    expect(screen.getByText(/E2E banned fixture/i)).toBeInTheDocument()
  })

  it('shows "Your Appeals" section', () => {
    renderPage()
    expect(screen.getByText('Your Appeals')).toBeInTheDocument()
  })

  it('shows "You have not submitted any appeals yet" when list is empty', () => {
    renderPage()
    expect(screen.getByText(/you have not submitted any appeals yet/i)).toBeInTheDocument()
  })

  it('shows "Submit Appeal" heading when can_appeal is true and no pending appeal', () => {
    renderPage()
    expect(screen.getByRole('heading', { name: 'Submit Appeal' })).toBeInTheDocument()
  })
})
