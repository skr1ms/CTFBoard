import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { backoff, useWsStore } from './wsStore'

const BASE = 1_000
const MAX = 30_000

// ── backoff ───────────────────────────────────────────────────────────────────

describe('backoff', () => {
  it('returns BASE_DELAY on attempt 0', () => {
    expect(backoff(0)).toBe(BASE)
  })

  it('doubles each attempt', () => {
    expect(backoff(1)).toBe(2_000)
    expect(backoff(2)).toBe(4_000)
    expect(backoff(3)).toBe(8_000)
  })

  it('caps at MAX_DELAY', () => {
    expect(backoff(5)).toBe(MAX) // 2^5 * 1000 = 32000 > 30000
    expect(backoff(10)).toBe(MAX)
    expect(backoff(100)).toBe(MAX)
  })

  it('reaches cap exactly at the boundary', () => {
    // 2^4 * 1000 = 16000, 2^5 * 1000 = 32000 -> capped to 30000
    expect(backoff(4)).toBe(16_000)
    expect(backoff(5)).toBe(MAX)
  })
})

// ── initial store state ───────────────────────────────────────────────────────

describe('useWsStore - initial state', () => {
  it('starts disconnected with zero reconnect attempts', () => {
    const state = useWsStore.getState()
    expect(state.connected).toBe(false)
    expect(state.reconnectAttempt).toBe(0)
    expect(state.usingSse).toBe(false)
    expect(state.lastEvent).toBeNull()
  })
})

// ── disconnect() ──────────────────────────────────────────────────────────────

describe('useWsStore - disconnect()', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
    useWsStore.setState({ connected: false, reconnectAttempt: 0, usingSse: false, lastEvent: null })
  })

  it('resets connected, reconnectAttempt and usingSse', () => {
    useWsStore.setState({ connected: true, reconnectAttempt: 3, usingSse: true })
    useWsStore.getState().disconnect()

    const state = useWsStore.getState()
    expect(state.connected).toBe(false)
    expect(state.reconnectAttempt).toBe(0)
    expect(state.usingSse).toBe(false)
  })
})

// ── connect() / disconnect() - observable state transitions ──────────────────

describe('useWsStore - state transitions', () => {
  afterEach(() => {
    useWsStore.setState({ connected: false, reconnectAttempt: 0, usingSse: false, lastEvent: null })
  })

  it('disconnect() is idempotent - calling it twice does not error', () => {
    useWsStore.setState({ connected: true, reconnectAttempt: 2 })
    useWsStore.getState().disconnect()
    expect(() => useWsStore.getState().disconnect()).not.toThrow()
    expect(useWsStore.getState().reconnectAttempt).toBe(0)
  })

  it('reconnectAttempt can be read from store state', () => {
    useWsStore.setState({ reconnectAttempt: 5 })
    expect(useWsStore.getState().reconnectAttempt).toBe(5)
  })

  it('usingSse flag reflects SSE fallback mode', () => {
    useWsStore.setState({ usingSse: true })
    expect(useWsStore.getState().usingSse).toBe(true)
    useWsStore.setState({ usingSse: false })
    expect(useWsStore.getState().usingSse).toBe(false)
  })
})
