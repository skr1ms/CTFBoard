import type { LoginRequest, TokenPair, MeResponse } from '../types/auth'

const baseUrl = (import.meta.env.VITE_API_URL ?? '').replace(/\/$/, '')
const apiPrefix = `${baseUrl}/api/v1`

export async function login(email: string, password: string): Promise<TokenPair> {
  const res = await fetch(`${apiPrefix}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password } as LoginRequest),
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({})) as { error?: string; code?: string }
    const msg = body.error ?? res.statusText
    const friendly =
      msg === 'invalid credentials' ? 'Invalid email or password' :
      msg === 'email not verified' ? 'Email not verified. Check your inbox.' :
      msg
    throw new Error(friendly)
  }
  return res.json() as Promise<TokenPair>
}

export async function getMe(accessToken: string): Promise<MeResponse> {
  const res = await fetch(`${apiPrefix}/auth/me`, {
    headers: { Authorization: `Bearer ${accessToken}` },
  })
  if (!res.ok) throw new Error('Unauthorized')
  return res.json() as Promise<MeResponse>
}
