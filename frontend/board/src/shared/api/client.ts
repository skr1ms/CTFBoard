import { env } from '@/shared/config/env'
import createClient, { type Middleware } from 'openapi-fetch'
import type { paths } from './schema.d'

export type { paths }

export interface ApiError {
  code: string
  message: string
}

export function isApiError(value: unknown): value is ApiError {
  return (
    typeof value === 'object' &&
    value !== null &&
    'code' in value &&
    'message' in value &&
    typeof (value as Record<string, unknown>).code === 'string'
  )
}

/** Returns the API error message, a plain Error message, or the provided fallback. */
export function apiErrorMessage(err: unknown, fallback = 'Something went wrong.'): string {
  if (isApiError(err)) return err.message
  if (err instanceof Error) return err.message
  return fallback
}

// Injected at runtime by authStore to avoid circular deps.
export interface TokenStore {
  getAccessToken(): string | null
  /** True when the browser is expected to have a valid httpOnly refresh cookie. */
  hasSession(): boolean
  setAccessToken(accessToken: string): void
}

let _tokenStore: TokenStore | null = null

export function registerTokenStore(store: TokenStore): void {
  _tokenStore = store
}

// No auth middleware - used for the refresh call itself.
// credentials: 'include' is required so the browser sends the httpOnly refresh cookie.
export const baseClient = createClient<paths>({ baseUrl: env.apiBaseUrl, credentials: 'include' })

// One in-flight refresh at a time.
let refreshPromise: Promise<boolean> | null = null

async function attemptRefresh(): Promise<boolean> {
  // The httpOnly refresh cookie is sent automatically; no Authorization header needed.
  const { data, error } = await baseClient.POST('/auth/refresh')

  if (error || !data?.access_token) return false

  _tokenStore?.setAccessToken(data.access_token)
  return true
}

function doRefresh(): Promise<boolean> {
  if (!refreshPromise) {
    refreshPromise = attemptRefresh().finally(() => {
      refreshPromise = null
    })
  }
  return refreshPromise
}

/** Exposed for non-middleware callers (e.g. SSE transport) that need to refresh
 * outside the openapi-fetch middleware chain. Shares the singleflight guard. */
export function refreshTokens(): Promise<boolean> {
  return doRefresh()
}

// Clones are stored before the first fetch so the body is available for retry.
// (After fetch() completes, request.bodyUsed === true and the body cannot be re-read.)
const requestClones = new WeakMap<Request, Request>()

// Embeds HTTP status into the JSON body so downstream handlers (QueryProvider,
// hooks) can call extractStatus(error) without access to the raw Response object.
async function withEmbeddedStatus(response: Response): Promise<Response> {
  if (response.ok) return response
  try {
    const body = (await response.clone().json()) as Record<string, unknown>
    const extra: Record<string, unknown> = { __httpStatus: response.status }
    const retryAfter = response.headers.get('Retry-After')
    if (retryAfter !== null) {
      const parsed = Number(retryAfter)
      extra['retryAfter'] = Number.isFinite(parsed) ? parsed : retryAfter
    }
    return new Response(JSON.stringify({ ...body, ...extra }), {
      status: response.status,
      headers: response.headers,
    })
  } catch {
    return response
  }
}

const authMiddleware: Middleware = {
  async onRequest({ request }) {
    requestClones.set(request, request.clone())
    const token = _tokenStore?.getAccessToken()
    if (token) {
      request.headers.set('Authorization', `Bearer ${token}`)
    }
    return request
  },

  async onResponse({ request, response }) {
    // For non-401 errors embed status immediately so callers can inspect it.
    if (response.status !== 401) return withEmbeddedStatus(response)

    // Don't retry refresh endpoint itself
    if (new URL(request.url).pathname.endsWith('/auth/refresh')) {
      return withEmbeddedStatus(response)
    }

    // If there's no active session the user was never logged in - don't redirect.
    if (!_tokenStore?.hasSession()) {
      return withEmbeddedStatus(response)
    }

    const ok = await doRefresh()
    if (!ok) {
      // Logout is handled by QueryProvider.queryCache.onError / mutations.onError
      // when this 401 propagates to React Query. Calling it here as well would
      // cause a race: two concurrent logout() invocations fighting over queryClient.clear().
      return withEmbeddedStatus(response)
    }

    // Retry using the pre-fetch clone so the body is still available
    const clone = requestClones.get(request)
    requestClones.delete(request)
    if (!clone) return withEmbeddedStatus(response)

    const newToken = _tokenStore?.getAccessToken()
    if (newToken) clone.headers.set('Authorization', `Bearer ${newToken}`)

    // NOTE: intentionally bypasses middleware chain on retry - a second openapi-fetch
    // call would re-enter onRequest and potentially loop. Raw fetch is correct here.
    return withEmbeddedStatus(await fetch(clone))
  },
}

export const api = createClient<paths>({ baseUrl: env.apiBaseUrl, credentials: 'include' })
api.use(authMiddleware)
