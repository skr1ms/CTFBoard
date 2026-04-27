/**
 * OAuth callback spec - error paths and success path for the /auth/callback page.
 *
 * The backend redirects to /auth/callback with either:
 *   - ?error=<code>          (provider denied access)
 *   - #access_token=...&refresh_token=...  (success)
 *
 * The frontend OAuthCallbackPage strips the hash immediately and processes
 * the tokens or error, then navigates to /challenges or /login.
 */
import { expect, test } from '@playwright/test'

const BACKEND_PORT = process.env['VITE_BACKEND_PORT'] ?? '8090'
const API = `http://localhost:${BACKEND_PORT}/api/v1`

async function loginApi(email: string, password: string): Promise<{ access_token: string; refresh_token: string }> {
  const res = await fetch(`${API}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  })
  if (!res.ok) {
    const body = await res.text().catch(() => '')
    throw new Error(`login failed ${res.status}: ${body}`)
  }
  return res.json() as Promise<{ access_token: string; refresh_token: string }>
}

// All callback tests start unauthenticated - the page handles auth itself.
test.describe('OAuth callback', () => {
  test.use({ storageState: { cookies: [], origins: [] } })

  test('?error=access_denied redirects to /login', async ({ page }) => {
    await page.goto('/auth/callback?error=access_denied')
    await expect(page).toHaveURL(/\/login/, { timeout: 10_000 })
  })

  test('?error=OAUTH_PROVIDER_DISABLED redirects to /login', async ({ page }) => {
    await page.goto('/auth/callback?error=OAUTH_PROVIDER_DISABLED')
    await expect(page).toHaveURL(/\/login/, { timeout: 10_000 })
  })

  test('no hash and no error redirects to /login', async ({ page }) => {
    await page.goto('/auth/callback')
    await expect(page).toHaveURL(/\/login/, { timeout: 10_000 })
  })

  test('hash with only access_token (no refresh_token) redirects to /login', async ({ page }) => {
    await page.goto('/auth/callback#access_token=fake-token-only')
    await expect(page).toHaveURL(/\/login/, { timeout: 10_000 })
  })

  test('success: valid tokens in hash -> user lands on /challenges', async ({ page }) => {
    // Obtain real tokens via the login API so /auth/me succeeds.
    const { access_token, refresh_token } = await loginApi('user2@example.com', 'Password1')

    // Playwright cannot set hash via page.goto() directly when there is a redirect,
    // so we navigate to the page and then set the hash via evaluate before React mounts.
    // Instead, build the full URL and navigate - Playwright will follow without stripping the hash.
    await page.goto(
      `/auth/callback#access_token=${encodeURIComponent(access_token)}&refresh_token=${encodeURIComponent(refresh_token)}`,
    )

    // OAuthCallbackPage processes the hash, calls /auth/me, then navigates to /challenges.
    await expect(page).toHaveURL(/\/challenges/, { timeout: 15_000 })
  })

  test('hash is cleared from browser history after processing', async ({ page }) => {
    const { access_token, refresh_token } = await loginApi('user2@example.com', 'Password1')

    await page.goto(
      `/auth/callback#access_token=${encodeURIComponent(access_token)}&refresh_token=${encodeURIComponent(refresh_token)}`,
    )
    await expect(page).toHaveURL(/\/challenges/, { timeout: 15_000 })

    // The hash must have been stripped by replaceState before login() was called.
    const hash = await page.evaluate(() => window.location.hash)
    expect(hash).toBe('')
  })
})
