/**
 * E2E credential resolution - single source of truth.
 *
 * Priority chain for admin email/password:
 *   1. E2E_EMAIL / E2E_PASSWORD env vars (explicit override)
 *   2. Project-root .env.local ADMIN_EMAIL / ADMIN_PASSWORD (local-compose)
 *   3. Defaults 'admin@example.com' / 'password' (matches docker-compose.e2e.yml)
 *
 * This lets e2e tests run against either stack without per-invocation env vars.
 */
import * as fs from 'fs'
import * as path from 'path'
import { fileURLToPath } from 'url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

function loadEnvFile(filePath: string): Record<string, string> {
  try {
    const env: Record<string, string> = {}
    for (const line of fs.readFileSync(filePath, 'utf-8').split('\n')) {
      const trimmed = line.trim()
      if (!trimmed || trimmed.startsWith('#')) continue
      const eqIdx = trimmed.indexOf('=')
      if (eqIdx === -1) continue
      const key = trimmed.slice(0, eqIdx).trim()
      const val = trimmed.slice(eqIdx + 1).trim().replace(/^['"]|['"]$/g, '')
      if (key) env[key] = val
    }
    return env
  } catch {
    return {}
  }
}

const localEnv = loadEnvFile(path.resolve(__dirname, '../../../.env.local'))

export const ADMIN_EMAIL =
  process.env['E2E_EMAIL'] ?? localEnv['ADMIN_EMAIL'] ?? 'admin@example.com'
export const ADMIN_PASSWORD =
  process.env['E2E_PASSWORD'] ?? localEnv['ADMIN_PASSWORD'] ?? 'password'
