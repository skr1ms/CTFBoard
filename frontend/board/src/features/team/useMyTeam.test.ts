import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { createElement } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useKickMember, useLeaveTeam, useMyTeam } from './useMyTeam'

// ── mocks ─────────────────────────────────────────────────────────────────────

vi.mock('@/shared/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/shared/api/client')>()
  return {
    ...actual,
    api: { GET: vi.fn(), POST: vi.fn(), DELETE: vi.fn() },
  }
})

vi.mock('@/shared/stores/authStore', () => {
  const mockState = { isAuthenticated: true, setUser: vi.fn() }
  const useAuthStore = Object.assign(
    vi.fn((selector?: (s: typeof mockState) => unknown) =>
      selector ? selector(mockState) : mockState,
    ),
    { getState: () => mockState },
  )
  return { useAuthStore }
})

vi.mock('sonner', () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}))

// ── helpers ───────────────────────────────────────────────────────────────────

function makeClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
}

function wrapper(qc: QueryClient) {
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: qc }, children)
}

const mockTeam = {
  id: 't1',
  name: 'Alpha Team',
  captain_id: 'u1',
  members: [
    { id: 'u1', username: 'captain', role: 'user' },
    { id: 'u2', username: 'member', role: 'user' },
  ],
}

// ── useMyTeam ─────────────────────────────────────────────────────────────────

describe('useMyTeam', () => {
  let qc: QueryClient

  beforeEach(() => {
    qc = makeClient()
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it('returns team with members on success', async () => {
    const { api } = await import('@/shared/api/client')
    vi.mocked(api.GET).mockResolvedValue({ data: mockTeam, error: undefined } as never)

    const { result } = renderHook(() => useMyTeam(), { wrapper: wrapper(qc) })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(result.current.data?.id).toBe('t1')
    expect(result.current.data?.members).toHaveLength(2)
  })

  it('handles team with no members (members field undefined)', async () => {
    const { api } = await import('@/shared/api/client')
    vi.mocked(api.GET).mockResolvedValue({
      data: { ...mockTeam, members: undefined },
      error: undefined,
    } as never)

    const { result } = renderHook(() => useMyTeam(), { wrapper: wrapper(qc) })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    // Consuming code does `team.members ?? []` - verify data passes through without throw
    expect(result.current.data?.members).toBeUndefined()
  })

  it('surfaces error when API call fails', async () => {
    const { api } = await import('@/shared/api/client')
    vi.mocked(api.GET).mockResolvedValue({
      data: undefined,
      error: { code: 'NOT_IN_TEAM', message: 'not in team' },
    } as never)

    const { result } = renderHook(() => useMyTeam(), { wrapper: wrapper(qc) })

    await waitFor(() => expect(result.current.isError).toBe(true))
  })
})

// ── useKickMember ─────────────────────────────────────────────────────────────

describe('useKickMember', () => {
  let qc: QueryClient

  beforeEach(() => {
    qc = makeClient()
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it('calls DELETE with the correct member ID', async () => {
    const { api } = await import('@/shared/api/client')
    vi.mocked(api.DELETE).mockResolvedValue({ error: undefined } as never)

    const { result } = renderHook(() => useKickMember(), { wrapper: wrapper(qc) })

    result.current.mutate('u2')

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(api.DELETE).toHaveBeenCalledWith('/teams/members/{ID}', {
      params: { path: { ID: 'u2' } },
    })
  })

  it('invalidates my-team query on success', async () => {
    const { api } = await import('@/shared/api/client')
    vi.mocked(api.DELETE).mockResolvedValue({ error: undefined } as never)

    const invalidateSpy = vi.spyOn(qc, 'invalidateQueries')

    const { result } = renderHook(() => useKickMember(), { wrapper: wrapper(qc) })

    result.current.mutate('u2')

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(invalidateSpy).toHaveBeenCalledWith(expect.objectContaining({ queryKey: ['my-team'] }))
  })

  it('shows error toast when kick fails', async () => {
    const { api } = await import('@/shared/api/client')
    const { toast } = await import('sonner')
    vi.mocked(api.DELETE).mockResolvedValue({
      error: { code: 'FORBIDDEN', message: 'not captain' },
    } as never)

    const { result } = renderHook(() => useKickMember(), { wrapper: wrapper(qc) })

    result.current.mutate('u2')

    await waitFor(() => expect(result.current.isError).toBe(true))

    expect(toast.error).toHaveBeenCalledWith('Failed to kick member')
  })
})

// ── useLeaveTeam ──────────────────────────────────────────────────────────────

describe('useLeaveTeam', () => {
  let qc: QueryClient

  beforeEach(() => {
    qc = makeClient()
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it('invalidates my-team query and syncs user store on success', async () => {
    const { api } = await import('@/shared/api/client')
    vi.mocked(api.POST).mockResolvedValue({ error: undefined } as never)
    vi.mocked(api.GET).mockResolvedValue({ data: null, error: undefined } as never)

    const invalidateSpy = vi.spyOn(qc, 'invalidateQueries')

    const { result } = renderHook(() => useLeaveTeam(), { wrapper: wrapper(qc) })

    result.current.mutate()

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(invalidateSpy).toHaveBeenCalledWith(expect.objectContaining({ queryKey: ['my-team'] }))
    expect(invalidateSpy).not.toHaveBeenCalledWith(expect.objectContaining({ queryKey: ['me'] }))
  })

  it('shows error toast when leave fails', async () => {
    const { api } = await import('@/shared/api/client')
    const { toast } = await import('sonner')
    vi.mocked(api.POST).mockResolvedValue({
      error: { code: 'MUST_TRANSFER_CAPTAIN', message: 'transfer captain first' },
    } as never)

    const { result } = renderHook(() => useLeaveTeam(), { wrapper: wrapper(qc) })

    result.current.mutate()

    await waitFor(() => expect(result.current.isError).toBe(true))

    expect(toast.error).toHaveBeenCalledWith('Failed to leave team')
  })
})
