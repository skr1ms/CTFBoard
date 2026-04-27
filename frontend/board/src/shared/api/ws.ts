import { queryClient } from '@/app/providers/QueryProvider'
import type {
  NotificationPayload,
  ScoreboardUpdatePayload,
  WsEventEnvelope,
} from '@/shared/stores/wsStore'
import { useWsStore } from '@/shared/stores/wsStore'
import { toast } from 'sonner'

// ---------------------------------------------------------------------------
// Level -> toast colour mapping
// ---------------------------------------------------------------------------
const LEVEL_TOAST: Record<string, (msg: string) => void> = {
  info: (m) => toast.info(m),
  success: (m) => toast.success(m),
  warning: (m) => toast.warning(m),
  error: (m) => toast.error(m),
}

function handleScoreboardUpdate(payload: ScoreboardUpdatePayload) {
  queryClient.invalidateQueries({ queryKey: ['scoreboard'] })

  if (payload.type === 'first_blood') {
    toast(`🩸 First Blood! ${payload.challenge}`, {
      description: `${payload.points} pts`,
      style: {
        background: 'var(--color-solar-yellow-dim, #1a1600)',
        border: '1px solid var(--color-solar-yellow, #f59e0b)',
        color: 'var(--color-solar-yellow, #f59e0b)',
      },
      duration: 6_000,
    })
  }
}

function handleNotification(payload: NotificationPayload) {
  const fn = LEVEL_TOAST[payload.level] ?? ((m: string) => toast.info(m))
  fn(payload.message)
  queryClient.invalidateQueries({ queryKey: ['notifications'] })
}

function handleEvent(event: WsEventEnvelope) {
  switch (event.type) {
    case 'scoreboard_update': {
      const p = event.payload as ScoreboardUpdatePayload | null
      if (p) handleScoreboardUpdate(p)
      break
    }
    case 'notification': {
      const p = event.payload as NotificationPayload | null
      if (p) handleNotification(p)
      break
    }
    default:
      if (import.meta.env.DEV) {
        console.warn('[ws] unhandled event type:', event.type)
      }
      break
  }
}

// ---------------------------------------------------------------------------
// Subscribe to wsStore events and auth state changes.
// Call once at app startup (after QueryProvider is mounted).
// Returns an unsubscribe function.
// ---------------------------------------------------------------------------
export function subscribeWsEvents(): () => void {
  let lastEvent: WsEventEnvelope | null = null

  const unsubEvents = useWsStore.subscribe((state) => {
    const ev = state.lastEvent
    if (!ev || ev === lastEvent) return
    lastEvent = ev
    handleEvent(ev)
  })

  // Connect/disconnect WS in response to auth state changes.
  // Lazy-import authStore to avoid module init cycles.
  let unsubAuth: (() => void) | undefined

  import('@/shared/stores/authStore')
    .then(({ useAuthStore }) => {
      // Connect immediately if already authenticated
      const { isAuthenticated, accessToken } = useAuthStore.getState()
      if (isAuthenticated && accessToken) {
        useWsStore.getState().connect()
      }

      // Watch future auth changes - also handles F5 hydration where isAuthenticated
      // is set optimistically before accessToken is available (stays true but
      // accessToken transitions null -> value)
      unsubAuth = useAuthStore.subscribe((state, prev) => {
        if (
          state.isAuthenticated &&
          state.accessToken &&
          (!prev.isAuthenticated || !prev.accessToken)
        ) {
          useWsStore.getState().connect()
        } else if (!state.isAuthenticated && prev.isAuthenticated) {
          useWsStore.getState().disconnect()
        }
      })
    })
    .catch(() => undefined)

  return () => {
    unsubEvents()
    unsubAuth?.()
  }
}
