import { refreshTokens } from '@/shared/api/client'
import { env } from '@/shared/config/env'
import { useAuthStore } from '@/shared/stores/authStore'
import { create } from 'zustand'

// ---------------------------------------------------------------------------
// WS event types (from OpenAPI /ws description)
// ---------------------------------------------------------------------------

export interface WsEventEnvelope {
  type: 'connected' | 'scoreboard_update' | 'notification'
  payload: ScoreboardUpdatePayload | NotificationPayload | null
  timestamp: string
}

export interface ScoreboardUpdatePayload {
  type: 'solve' | 'first_blood'
  team_id: string
  challenge: string
  points: number
  timestamp: string
}

export interface NotificationPayload {
  type: 'notification'
  message: string
  level: 'info' | 'warning' | 'error' | 'success'
  timestamp: string
}

// ---------------------------------------------------------------------------
// Reconnect config
// ---------------------------------------------------------------------------

const BASE_DELAY_MS = 1_000
const MAX_DELAY_MS = 30_000
// After this many failed WS attempts, fall back to SSE
const SSE_FALLBACK_THRESHOLD = 3
const MAX_WS_ATTEMPTS = 10

export function backoff(attempt: number): number {
  return Math.min(BASE_DELAY_MS * 2 ** attempt, MAX_DELAY_MS)
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export interface WsState {
  connected: boolean
  lastEvent: WsEventEnvelope | null
  reconnectAttempt: number
  /** true when connected via SSE fallback instead of WebSocket */
  usingSse: boolean

  connect(): void
  disconnect(): void
}

let socket: WebSocket | null = null
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let shouldReconnect = true
let sseAbort: AbortController | null = null
// Tracks whether the current socket has ever received an onopen event.
// Used to detect early-close auth rejections before any data is exchanged.
let wsHadOpen = false

export const useWsStore = create<WsState>((set, get) => ({
  connected: false,
  lastEvent: null,
  reconnectAttempt: 0,
  usingSse: false,

  connect() {
    if (socket && socket.readyState === WebSocket.OPEN) return
    if (sseAbort) return // already on SSE

    // Cancel any pending reconnect timer so we don't get two parallel open attempts
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }

    shouldReconnect = true
    openSocket(set, get)
  },

  disconnect() {
    shouldReconnect = false
    wsHadOpen = false
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    socket?.close(1000, 'user logout')
    socket = null

    sseAbort?.abort()
    sseAbort = null

    set({ connected: false, reconnectAttempt: 0, usingSse: false })
  },
}))

// ---------------------------------------------------------------------------
// Token helper - always reads the freshest access token from authStore
// ---------------------------------------------------------------------------

function getToken(): string | null {
  return useAuthStore.getState().accessToken ?? null
}

// ---------------------------------------------------------------------------
// WS message type guard
// ---------------------------------------------------------------------------

function isWsEventEnvelope(obj: unknown): obj is WsEventEnvelope {
  return (
    typeof obj === 'object' &&
    obj !== null &&
    'type' in obj &&
    typeof (obj as Record<string, unknown>).type === 'string' &&
    'timestamp' in obj &&
    typeof (obj as Record<string, unknown>).timestamp === 'string'
  )
}

// ---------------------------------------------------------------------------
// WebSocket transport
// ---------------------------------------------------------------------------

function openSocket(set: (partial: Partial<WsState>) => void, get: () => WsState): void {
  const token = getToken()
  if (!token) return

  // Pass the JWT via Sec-WebSocket-Protocol: bearer, <token> instead of ?token=
  // in the query string (which would appear in server access logs).
  socket = new WebSocket(env.wsUrl, ['bearer', token])

  socket.onopen = () => {
    wsHadOpen = true
    set({ connected: true, reconnectAttempt: 0, usingSse: false })
  }

  socket.onmessage = (ev: MessageEvent) => {
    try {
      const parsed: unknown = JSON.parse(ev.data as string)
      if (isWsEventEnvelope(parsed)) {
        set({ lastEvent: parsed })
      } else if (import.meta.env.DEV) {
        console.warn('[wsStore] unrecognised WS frame:', parsed)
      }
    } catch {
      // ignore malformed frames
    }
  }

  socket.onclose = (ev) => {
    const hadOpen = wsHadOpen
    wsHadOpen = false
    set({ connected: false })
    socket = null

    if (!shouldReconnect || ev.code === 1000) return

    const attempt = get().reconnectAttempt

    // Early close without ever reaching onopen on the first attempt is a
    // strong signal the server rejected the handshake (expired access token).
    // Proactively refresh so the next reconnect uses a valid credential instead
    // of burning 3 backoff rounds before the SSE fallback path handles it.
    if (!hadOpen && attempt === 0 && useAuthStore.getState().isAuthenticated) {
      void (async () => {
        const ok = await refreshTokens()
        if (!ok) {
          useAuthStore.getState().logout()
          return
        }
        openSocket(set, get)
      })()
      return
    }

    const nextAttempt = attempt + 1

    // After SSE_FALLBACK_THRESHOLD failed attempts, switch to SSE
    if (nextAttempt >= SSE_FALLBACK_THRESHOLD) {
      set({ reconnectAttempt: nextAttempt })
      connectSse(set, get)
      return
    }

    set({ reconnectAttempt: nextAttempt })
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null
      openSocket(set, get)
    }, backoff(attempt))
  }

  socket.onerror = () => {
    // onclose fires after onerror - reconnect handled there
  }
}

// ---------------------------------------------------------------------------
// SSE transport (fetch + ReadableStream, supports Authorization header)
// ---------------------------------------------------------------------------

function connectSse(set: (partial: Partial<WsState>) => void, get: () => WsState): void {
  if (sseAbort) sseAbort.abort()
  sseAbort = new AbortController()
  const { signal } = sseAbort

  set({ usingSse: true })

  void (async () => {
    try {
      const token = getToken()
      if (!token) return

      const res = await fetch(env.sseUrl, {
        headers: { Authorization: `Bearer ${token}` },
        signal,
      })

      if (res.status === 401) {
        const ok = await refreshTokens()
        if (!ok) {
          useAuthStore.getState().logout()
          return
        }
        // Reconnect with the refreshed token
        connectSse(set, get)
        return
      }

      if (!res.ok || !res.body) {
        throw new Error(`SSE HTTP ${res.status}`)
      }

      set({ connected: true, reconnectAttempt: 0 })

      const reader = res.body.getReader()
      const decoder = new TextDecoder()
      let buf = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buf += decoder.decode(value, { stream: true })
        const lines = buf.split('\n')
        buf = lines.pop() ?? ''

        for (const line of lines) {
          if (!line.startsWith('data:')) continue
          const payload = line.slice(5).trim()
          if (!payload) continue
          try {
            const parsed: unknown = JSON.parse(payload)
            if (isWsEventEnvelope(parsed)) {
              set({ lastEvent: parsed })
            } else if (import.meta.env.DEV) {
              console.warn('[wsStore] unrecognised SSE frame:', parsed)
            }
          } catch {
            // ignore malformed SSE data
          }
        }
      }
    } catch (err) {
      if ((err as { name?: string }).name === 'AbortError') return
    } finally {
      set({ connected: false, usingSse: false })
      sseAbort = null
    }

    if (!shouldReconnect) return

    const attempt = get().reconnectAttempt
    const nextAttempt = attempt + 1

    if (nextAttempt >= MAX_WS_ATTEMPTS) return

    set({ reconnectAttempt: nextAttempt })
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null
      // If we already exceeded the WS threshold, stay on SSE path instead of
      // bouncing back to WS and immediately falling over again.
      if (get().reconnectAttempt >= SSE_FALLBACK_THRESHOLD) {
        connectSse(set, get)
      } else {
        openSocket(set, get)
      }
    }, backoff(attempt))
  })()
}

// ---------------------------------------------------------------------------
// Token refresh subscription - bounce the socket immediately when the access
// token is replaced so the server sees the new credential on reconnect.
// (The server validates the token only at handshake time, not on every frame.)
// ---------------------------------------------------------------------------

useAuthStore.subscribe((state, prev) => {
  if (state.accessToken === prev.accessToken || !state.accessToken) return
  if (!socket || socket.readyState !== WebSocket.OPEN) return
  // Reset attempt counter so this intentional bounce doesn't count toward the
  // SSE fallback threshold.
  useWsStore.setState({ reconnectAttempt: 0 })
  socket.close(1001, 'token refresh')
})
