/**
 * API helper for e2e specs - standalone fetch, no Vite/src imports.
 */
import { ADMIN_EMAIL, ADMIN_PASSWORD } from '../env'

const BASE = `http://localhost:${process.env['VITE_BACKEND_PORT'] ?? '8090'}`

const tokenCache = new Map<string, string>()

export async function apiLogin(email: string, password: string): Promise<string> {
  const cached = tokenCache.get(email)
  if (cached) return cached

  const res = await fetch(`${BASE}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  })
  if (!res.ok) {
    const body = await res.text().catch(() => '')
    throw new Error(`login(${email}) -> ${res.status}: ${body}`)
  }
  const data = (await res.json()) as { access_token: string }
  tokenCache.set(email, data.access_token)
  return data.access_token
}

export function clearTokenCache() {
  tokenCache.clear()
}

export async function apiFetch(
  path: string,
  token: string,
  init?: RequestInit,
): Promise<unknown> {
  const res = await fetch(`${BASE}/api/v1${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
      ...init?.headers,
    },
  })
  if (!res.ok) {
    const body = await res.text().catch(() => '')
    throw new Error(`${init?.method ?? 'GET'} ${path} -> ${res.status}: ${body}`)
  }
  const text = await res.text()
  return text ? (JSON.parse(text) as unknown) : null
}

export async function getAdminToken(): Promise<string> {
  return apiLogin(ADMIN_EMAIL, ADMIN_PASSWORD)
}

export async function adminFetch(path: string, init?: RequestInit): Promise<unknown> {
  const token = await getAdminToken()
  return apiFetch(path, token, init)
}

export async function setConfig(
  key: string,
  value: string,
  valueType = 'string',
): Promise<void> {
  await adminFetch(`/admin/configs/${key}`, {
    method: 'PUT',
    body: JSON.stringify({ value, value_type: valueType }),
  })
}

export async function getConfig(key: string): Promise<unknown> {
  return adminFetch(`/admin/configs/${key}`)
}

export async function banUser(userId: string, reason: string): Promise<void> {
  await adminFetch(`/admin/users/${userId}/ban`, {
    method: 'POST',
    body: JSON.stringify({ reason }),
  })
}

export async function unbanUser(userId: string): Promise<void> {
  await adminFetch(`/admin/users/${userId}/ban`, { method: 'DELETE' })
}

export async function getUserByEmail(
  email: string,
): Promise<{ id: string; username: string; [k: string]: unknown }> {
  const res = (await adminFetch(`/admin/users?q=${encodeURIComponent(email)}&per_page=5`)) as {
    data?: Array<{ id: string; username: string; email?: string; [k: string]: unknown }>
  }
  const users = res.data ?? (res as unknown as Array<{ id: string; username: string }>)
  const match = (Array.isArray(users) ? users : []).find(
    (u: { email?: string }) => u.email === email,
  )
  if (!match) throw new Error(`User not found: ${email}`)
  return match as { id: string; username: string; [k: string]: unknown }
}
