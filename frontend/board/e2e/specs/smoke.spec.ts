/**
 * Smoke tests - verify key user flows render without JS errors.
 *
 * Unauthenticated describes clear storage explicitly.
 * Authenticated tests share a single browser context (one login, one hydration).
 *
 * Run: make test-e2e
 */
import { BrowserContext, Page, expect, test } from '@playwright/test'
import path from 'path'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const AUTH_STATE = path.join(__dirname, '../.auth/admin.json')

// ---------------------------------------------------------------------------
// Unauthenticated pages
// ---------------------------------------------------------------------------

test.describe('Smoke: unauthenticated pages', () => {
  test.use({ storageState: { cookies: [], origins: [] } })

  test('login page renders', async ({ page }) => {
    await page.goto('/login')
    await expect(page.getByRole('heading', { name: /sign in/i })).toBeVisible()
    await expect(page.getByTestId('login-email')).toBeVisible()
    await expect(page.getByTestId('login-password')).toBeVisible()
  })

  test('guest redirect: /challenges -> /login', async ({ page }) => {
    await page.goto('/challenges')
    await expect(page).toHaveURL(/\/login/)
  })

  test('404 page renders for unknown route', async ({ page }) => {
    await page.goto('/nonexistent-route-xyz')
    await expect(page.getByText(/lost in space|not found|404/i).first()).toBeVisible()
  })
})

// ---------------------------------------------------------------------------
// Authenticated tests - single context, one hydration cycle
// ---------------------------------------------------------------------------

test.describe('Smoke: authenticated', () => {
  let ctx: BrowserContext
  let pg: Page

  test.beforeAll(async ({ browser }) => {
    ctx = await browser.newContext({ storageState: AUTH_STATE })
    pg = await ctx.newPage()

    // Navigate once - triggers hydrate() which rotates refresh token.
    // All sub-tests reuse this page/context so hydrate runs only once.
    await pg.goto('/challenges')
    await pg.locator('[data-testid="challenge-card"]').first().waitFor({ timeout: 15_000 })
  })

  test.afterAll(async () => {
    await ctx.close()
  })

  test('challenges page loads', async () => {
    await expect(pg).toHaveURL(/\/challenges/)
    await expect(pg.locator('[data-testid="challenge-card"]').first()).toBeVisible()
  })

  test('challenge modal opens on card click', async () => {
    const card = pg.locator('[data-testid="challenge-card"]').first()
    await card.click()

    const dialog = pg.getByRole('dialog')
    await expect(dialog).toBeVisible({ timeout: 5_000 })

    await pg.keyboard.press('Escape')
    await expect(dialog).not.toBeVisible({ timeout: 3_000 })
  })

  test('scoreboard page loads', async () => {
    await pg.goto('/scoreboard')
    await expect(pg).toHaveURL(/\/scoreboard/, { timeout: 10_000 })
    await expect(pg.locator('h1, table, [class*="empty"]').first()).toBeVisible({ timeout: 8_000 })
  })

  test('settings page loads', async () => {
    await pg.goto('/settings')
    await expect(pg).toHaveURL(/\/settings/, { timeout: 10_000 })
    await expect(pg.locator('h1, h2').first()).toBeVisible({ timeout: 8_000 })
  })

  test('notifications page loads', async () => {
    await pg.goto('/notifications')
    await expect(pg).toHaveURL(/\/notifications/, { timeout: 10_000 })
    await expect(pg.locator('h1, [role="tablist"]').first()).toBeVisible({ timeout: 8_000 })
  })

  test('navbar links are present', async () => {
    await pg.goto('/challenges')
    await expect(pg.getByRole('link', { name: /challenges/i }).first()).toBeVisible()
    await expect(pg.getByRole('link', { name: /scoreboard/i }).first()).toBeVisible()
  })

  test('users list page loads', async () => {
    await pg.goto('/users')
    await expect(pg.locator('h1, table, [class*="empty"]').first()).toBeVisible({ timeout: 8_000 })
  })

  test('teams list page loads', async () => {
    await pg.goto('/teams')
    await expect(pg.locator('h1, table, [class*="empty"]').first()).toBeVisible({ timeout: 8_000 })
  })
})
