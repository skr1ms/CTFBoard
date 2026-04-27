// In production VITE_API_BASE_URL must be explicitly set via Docker build args.
// Failing loudly at startup prevents silent misconfiguration where all requests
// hit same-origin /api/v1 instead of the real API host, which breaks OAuth
// redirects and CORS-protected endpoints.
const rawApiBase = import.meta.env['VITE_API_BASE_URL'] as string | undefined

if (!rawApiBase && import.meta.env.PROD) {
  throw new Error('[env] VITE_API_BASE_URL must be set in production.')
}

export const env = {
  apiBaseUrl: rawApiBase || '/api/v1',
  wsUrl:
    import.meta.env['VITE_WS_URL'] ||
    `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/api/v1/ws`,
  sseUrl: import.meta.env['VITE_SSE_URL'] || '/api/v1/sse',
  appName: import.meta.env['VITE_APP_NAME'] || 'CTF Platform',
} as const
