import { test as base, type Page } from '@playwright/test'
import { ADMIN_EMAIL, ADMIN_PASSWORD } from './env'

const BACKEND_PORT = process.env['VITE_BACKEND_PORT'] ?? '8090'
const API = `http://localhost:${BACKEND_PORT}/api/v1`
const FRONTEND_ORIGIN = 'http://localhost:4173'

interface RoleCreds {
  email: string
  password: string
}

const CREDS: Record<string, RoleCreds> = {
  admin: { email: ADMIN_EMAIL, password: ADMIN_PASSWORD },
  user1: { email: 'user1@example.com', password: 'Password1' },
  user2: { email: 'user2@example.com', password: 'Password1' },
  banned: { email: 'banned@example.com', password: 'Password1' },
}

async function freshStorageState(role: string) {
  const creds = CREDS[role]
  if (!creds) throw new Error(`Unknown role: ${role}`)

  const res = await fetch(`${API}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(creds),
  })
  if (!res.ok) {
    const body = await res.text().catch(() => '')
    throw new Error(`freshStorageState(${role}) login failed ${res.status}: ${body}`)
  }
  const data = (await res.json()) as { refresh_token?: string }
  if (!data.refresh_token) throw new Error(`no refresh_token for ${role}`)

  return {
    cookies: [],
    origins: [
      {
        origin: FRONTEND_ORIGIN,
        localStorage: [{ name: 'ctf_refresh_token', value: data.refresh_token }],
      },
    ],
  }
}

interface Fixtures {
  adminPage: Page
  user1Page: Page
  user2Page: Page
  bannedPage: Page
  guestPage: Page
}

export const test = base.extend<Fixtures>({
  adminPage: async ({ browser }, use) => {
    const ctx = await browser.newContext({ storageState: await freshStorageState('admin') })
    const page = await ctx.newPage()
    await use(page)
    await ctx.close()
  },
  user1Page: async ({ browser }, use) => {
    const ctx = await browser.newContext({ storageState: await freshStorageState('user1') })
    const page = await ctx.newPage()
    await use(page)
    await ctx.close()
  },
  user2Page: async ({ browser }, use) => {
    const ctx = await browser.newContext({ storageState: await freshStorageState('user2') })
    const page = await ctx.newPage()
    await use(page)
    await ctx.close()
  },
  bannedPage: async ({ browser }, use) => {
    const ctx = await browser.newContext({ storageState: await freshStorageState('banned') })
    const page = await ctx.newPage()
    await use(page)
    await ctx.close()
  },
  guestPage: async ({ browser }, use) => {
    const ctx = await browser.newContext({ storageState: { cookies: [], origins: [] } })
    const page = await ctx.newPage()
    await use(page)
    await ctx.close()
  },
})

export { expect } from '@playwright/test'
