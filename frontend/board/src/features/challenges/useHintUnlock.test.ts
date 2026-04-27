import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, waitFor } from '@testing-library/react'
import { createElement } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useHintUnlock } from './useHintUnlock'

// ── mocks ─────────────────────────────────────────────────────────────────────

vi.mock('@/shared/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/shared/api/client')>()
  return {
    ...actual,
    api: { POST: vi.fn() },
  }
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

// ── tests ─────────────────────────────────────────────────────────────────────

describe('useHintUnlock', () => {
  let qc: QueryClient

  beforeEach(() => {
    qc = makeClient()
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it('calls POST with correct path params', async () => {
    const { api } = await import('@/shared/api/client')
    vi.mocked(api.POST).mockResolvedValue({ data: {}, error: undefined } as never)

    const { result } = renderHook(() => useHintUnlock('ch-1'), { wrapper: wrapper(qc) })

    result.current.mutate('hint-42')

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(api.POST).toHaveBeenCalledWith('/challenges/{challengeID}/hints/{hintID}/unlock', {
      params: { path: { challengeID: 'ch-1', hintID: 'hint-42' } },
    })
  })

  it('shows success toast on unlock', async () => {
    const { api } = await import('@/shared/api/client')
    const { toast } = await import('sonner')
    vi.mocked(api.POST).mockResolvedValue({ data: {}, error: undefined } as never)

    const { result } = renderHook(() => useHintUnlock('ch-1'), { wrapper: wrapper(qc) })

    result.current.mutate('hint-1')

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(toast.success).toHaveBeenCalledWith('Hint unlocked')
  })

  it('invalidates challenge, scoreboard and my-team on success', async () => {
    const { api } = await import('@/shared/api/client')
    vi.mocked(api.POST).mockResolvedValue({ data: {}, error: undefined } as never)

    const invalidateSpy = vi.spyOn(qc, 'invalidateQueries')

    const { result } = renderHook(() => useHintUnlock('ch-99'), { wrapper: wrapper(qc) })

    result.current.mutate('h-1')

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(invalidateSpy).toHaveBeenCalledWith(
      expect.objectContaining({ queryKey: ['challenge', 'ch-99'] }),
    )
    expect(invalidateSpy).toHaveBeenCalledWith(
      expect.objectContaining({ queryKey: ['scoreboard'] }),
    )
    expect(invalidateSpy).toHaveBeenCalledWith(expect.objectContaining({ queryKey: ['my-team'] }))
  })

  it('shows insufficient-points error toast', async () => {
    const { api } = await import('@/shared/api/client')
    const { toast } = await import('sonner')
    vi.mocked(api.POST).mockResolvedValue({
      data: undefined,
      error: { code: 'INSUFFICIENT_POINTS', message: 'not enough' },
    } as never)

    const { result } = renderHook(() => useHintUnlock('ch-1'), { wrapper: wrapper(qc) })

    result.current.mutate('hint-1')

    await waitFor(() => expect(result.current.isError).toBe(true))

    expect(toast.error).toHaveBeenCalledWith('Not enough points to unlock this hint')
  })

  it('shows generic error toast for non-INSUFFICIENT_POINTS errors', async () => {
    const { api } = await import('@/shared/api/client')
    const { toast } = await import('sonner')
    vi.mocked(api.POST).mockResolvedValue({
      data: undefined,
      error: { code: 'COMPETITION_FROZEN', message: 'competition is frozen' },
    } as never)

    const { result } = renderHook(() => useHintUnlock('ch-1'), { wrapper: wrapper(qc) })

    result.current.mutate('hint-1')

    await waitFor(() => expect(result.current.isError).toBe(true))

    expect(toast.error).toHaveBeenCalledWith('Failed to unlock hint')
  })

  it('does not invalidate on error', async () => {
    const { api } = await import('@/shared/api/client')
    vi.mocked(api.POST).mockResolvedValue({
      data: undefined,
      error: { code: 'INSUFFICIENT_POINTS', message: 'not enough' },
    } as never)

    const invalidateSpy = vi.spyOn(qc, 'invalidateQueries')

    const { result } = renderHook(() => useHintUnlock('ch-1'), { wrapper: wrapper(qc) })

    result.current.mutate('hint-1')

    await waitFor(() => expect(result.current.isError).toBe(true))

    expect(invalidateSpy).not.toHaveBeenCalled()
  })
})
