/**
 * Playwright global setup - runs once before all tests.
 * 1. Seeds backend data.
 * 2. Obtains refresh tokens for 4 roles via the backend API.
 * 3. Writes Playwright storage-state files (localStorage only - no cookies).
 *
 * Using the API directly avoids flaky browser login flows in global-setup.
 * The authStore's hydrate() function will exchange the refresh token for a
 * new access token on first page load in each test.
 */
import { execSync } from 'child_process'
import * as fs from 'fs'
import * as path from 'path'
import { fileURLToPath } from 'url'
import type { FullConfig } from '@playwright/test'
import { ADMIN_EMAIL, ADMIN_PASSWORD } from './env'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

const BACKEND_PORT = process.env['VITE_BACKEND_PORT'] ?? '8090'
const API = `http://localhost:${BACKEND_PORT}/api/v1`
const FRONTEND_ORIGIN = 'http://localhost:4173'

interface UserCreds {
  role: string
  email: string
  password: string
}

const USERS: UserCreds[] = [
  { role: 'admin', email: ADMIN_EMAIL, password: ADMIN_PASSWORD },
  { role: 'user1', email: 'user1@example.com', password: 'Password1' },
  { role: 'user2', email: 'user2@example.com', password: 'Password1' },
  { role: 'banned', email: 'banned@example.com', password: 'Password1' },
]

async function apiLogin(email: string, password: string): Promise<string> {
  const res = await fetch(`${API}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  })
  if (!res.ok) {
    const body = await res.text().catch(() => '')
    throw new Error(`login ${email} failed ${res.status}: ${body}`)
  }
  const data = (await res.json()) as { refresh_token?: string }
  if (!data.refresh_token) throw new Error(`no refresh_token for ${email}`)
  return data.refresh_token
}

export default async function globalSetup(_config: FullConfig): Promise<void> {
  // -------------------------------------------------------------------------
  // 1. Seed backend data
  // -------------------------------------------------------------------------
  console.log('[setup] seeding e2e data…')
  execSync(`VITE_BACKEND_PORT=${BACKEND_PORT} bun e2e/seed.ts`, {
    stdio: 'inherit',
    env: { ...process.env, E2E_EMAIL: ADMIN_EMAIL, E2E_PASSWORD: ADMIN_PASSWORD },
  })

  const authDir = path.join(__dirname, '.auth')
  fs.mkdirSync(authDir, { recursive: true })

  // -------------------------------------------------------------------------
  // 2. Obtain refresh tokens and write storage-state files
  // -------------------------------------------------------------------------
  for (const user of USERS) {
    const refreshToken = await apiLogin(user.email, user.password)

    // Playwright storage-state format - the app reads ctf_refresh_token from
    // localStorage to hydrate the auth store on first render.
    const storageState = {
      cookies: [],
      origins: [
        {
          origin: FRONTEND_ORIGIN,
          localStorage: [{ name: 'ctf_refresh_token', value: refreshToken }],
        },
      ],
    }

    const statePath = path.join(authDir, `${user.role}.json`)
    fs.writeFileSync(statePath, JSON.stringify(storageState, null, 2))
    console.log(`[setup] saved auth state for ${user.role}`)
  }

  console.log('[setup] global setup complete.')
}
