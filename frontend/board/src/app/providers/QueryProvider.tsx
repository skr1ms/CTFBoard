import { isApiError } from '@/shared/api/client'
import { useAuthStore } from '@/shared/stores/authStore'
import { QueryCache, QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { type ReactNode } from 'react'

// eslint-disable-next-line react-refresh/only-export-components
export function is4xx(status: unknown): boolean {
  return typeof status === 'number' && status >= 400 && status < 500
}

// eslint-disable-next-line react-refresh/only-export-components
export function extractStatus(error: unknown): number | undefined {
  if (
    typeof error === 'object' &&
    error !== null &&
    '__httpStatus' in error &&
    typeof (error as Record<string, unknown>).__httpStatus === 'number'
  ) {
    return (error as { __httpStatus: number }).__httpStatus
  }
  return undefined
}

// eslint-disable-next-line react-refresh/only-export-components
export const queryClient = new QueryClient({
  queryCache: new QueryCache({
    onError(error) {
      const status = extractStatus(error)
      if (status === 401) {
        useAuthStore
          .getState()
          .logout()
          .catch(() => undefined)
      }
    },
  }),
  defaultOptions: {
    queries: {
      // 30s default; callers can override per-query
      staleTime: 30_000,
      retry(failureCount, error) {
        const status = extractStatus(error)
        // No retries for client errors
        if (status !== undefined && is4xx(status)) return false
        // Up to 3 retries for server errors / network issues
        return failureCount < 3
      },
      refetchOnWindowFocus: false,
    },
    mutations: {
      onError(error) {
        const status = extractStatus(error)
        if (status === 401) {
          useAuthStore
            .getState()
            .logout()
            .catch(() => undefined)
        }
      },
    },
  },
})

// Stale time presets for consumers
// eslint-disable-next-line react-refresh/only-export-components
export const staleTime = {
  /** Live competition data: challenges, scoreboard */
  live: 30_000,
  /** User profile, team data */
  user: 60_000,
  /** Mostly static: pages, tags, brackets */
  static: 10 * 60_000,
} as const

export function QueryProvider({ children }: { children: ReactNode }) {
  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
}

// Re-export for convenience
// eslint-disable-next-line react-refresh/only-export-components
export { isApiError }
