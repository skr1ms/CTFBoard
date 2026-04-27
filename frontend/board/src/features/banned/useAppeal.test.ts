import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { createElement } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useCreateAppeal, useMyAppeals } from './useAppeal'

// ── mocks ─────────────────────────────────────────────────────────────────────

vi.mock('@/shared/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/shared/api/client')>()
  return {
    ...actual,
    api: {
      GET: vi.fn(),
      POST: vi.fn(),
    },
  }
})

vi.mock('@/shared/stores/authStore', () => {
  const mockState = { isAuthenticated: true }
  const useAuthStore = Object.assign(
    vi.fn((selector?: (s: typeof mockState) => unknown) =>
      selector ? selector(mockState) : mockState,
    ),
    { getState: () => mockState },
  )
  return { useAuthStore }
})

vi.mock('sonner', () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

// ── helpers ───────────────────────────────────────────────────────────────────

function makeClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } })
}

function wrapper(qc: QueryClient) {
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: qc }, children)
}

// ── useMyAppeals ──────────────────────────────────────────────────────────────

describe('useMyAppeals', () => {
  let qc: QueryClient

  beforeEach(() => {
    qc = makeClient()
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it('returns appeal list on success', async () => {
    const { api } = await import('@/shared/api/client')
    vi.mocked(api.GET).mockResolvedValue({
      data: {
        appeals: [
          {
            id: 'a1',
            message: 'Please unban me',
            status: 'pending',
            created_at: '2026-04-20T10:00:00Z',
          },
        ],
      },
      error: undefined,
    } as never)

    const { result } = renderHook(() => useMyAppeals(), { wrapper: wrapper(qc) })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(result.current.data).toHaveLength(1)
    expect(result.current.data?.[0]?.id).toBe('a1')
  })

  it('returns empty array when appeals is null/undefined', async () => {
    const { api } = await import('@/shared/api/client')
    vi.mocked(api.GET).mockResolvedValue({
      data: { appeals: null },
      error: undefined,
    } as never)

    const { result } = renderHook(() => useMyAppeals(), { wrapper: wrapper(qc) })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(result.current.data).toEqual([])
  })

  it('throws when API returns an error', async () => {
    const { api } = await import('@/shared/api/client')
    vi.mocked(api.GET).mockResolvedValue({
      data: undefined,
      error: { code: 'UNAUTHORIZED', message: 'not authenticated' },
    } as never)

    const { result } = renderHook(() => useMyAppeals(), { wrapper: wrapper(qc) })

    await waitFor(() => expect(result.current.isError).toBe(true))
  })
})

// ── useCreateAppeal ───────────────────────────────────────────────────────────

describe('useCreateAppeal', () => {
  let qc: QueryClient

  beforeEach(() => {
    qc = makeClient()
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it('calls POST /appeals with the provided message', async () => {
    const { api } = await import('@/shared/api/client')
    vi.mocked(api.POST).mockResolvedValue({
      data: { id: 'a1', message: 'unban me please' },
      error: undefined,
    } as never)

    const { result } = renderHook(() => useCreateAppeal(), { wrapper: wrapper(qc) })

    result.current.mutate('unban me please')

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(api.POST).toHaveBeenCalledWith('/appeals', { body: { message: 'unban me please' } })
  })

  it('shows success toast on successful submission', async () => {
    const { api } = await import('@/shared/api/client')
    const { toast } = await import('sonner')
    vi.mocked(api.POST).mockResolvedValue({ data: {}, error: undefined } as never)

    const { result } = renderHook(() => useCreateAppeal(), { wrapper: wrapper(qc) })

    result.current.mutate('valid appeal message here')

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(toast.success).toHaveBeenCalledWith('Appeal submitted successfully')
  })

  it('invalidates ["appeals", "me"] query on success', async () => {
    const { api } = await import('@/shared/api/client')
    vi.mocked(api.POST).mockResolvedValue({ data: {}, error: undefined } as never)

    const invalidateSpy = vi.spyOn(qc, 'invalidateQueries')

    const { result } = renderHook(() => useCreateAppeal(), { wrapper: wrapper(qc) })

    result.current.mutate('valid appeal message')

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(invalidateSpy).toHaveBeenCalledWith(
      expect.objectContaining({ queryKey: ['appeals', 'me'] }),
    )
  })

  it('shows error toast when mutation fails', async () => {
    const { api } = await import('@/shared/api/client')
    const { toast } = await import('sonner')
    vi.mocked(api.POST).mockResolvedValue({
      data: undefined,
      error: { code: 'ALREADY_APPEALED', message: 'Appeal already exists' },
    } as never)

    const { result } = renderHook(() => useCreateAppeal(), { wrapper: wrapper(qc) })

    result.current.mutate('appeal message')

    await waitFor(() => expect(result.current.isError).toBe(true))

    expect(toast.error).toHaveBeenCalledWith('Appeal already exists')
  })

  it('shows fallback error message when error has no message field', async () => {
    const { api } = await import('@/shared/api/client')
    const { toast } = await import('sonner')
    vi.mocked(api.POST).mockResolvedValue({
      data: undefined,
      error: {},
    } as never)

    const { result } = renderHook(() => useCreateAppeal(), { wrapper: wrapper(qc) })

    result.current.mutate('appeal message')

    await waitFor(() => expect(result.current.isError).toBe(true))

    expect(toast.error).toHaveBeenCalledWith('Failed to submit appeal')
  })
})
