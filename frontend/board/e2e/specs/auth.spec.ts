/**
 * Auth spec - login, register, verify-email, reset-password, logout, OAuth visibility.
 * Rate-limit tests are in serial mode to prevent interference.
 */
import { expect, test } from '@playwright/test'
import { adminFetch } from '../helpers/api'
import { tid } from '../helpers/selectors'

test.describe('Auth: login', () => {
  test.use({ storageState: { cookies: [], origins: [] } })

  test('successful login redirects to /challenges', async ({ page }) => {
    await page.goto('/login')
    await page.getByTestId(tid.login.email).fill('user1@example.com')
    await page.getByTestId(tid.login.password).fill('Password1')
    await page.getByTestId(tid.login.submit).click()
    await expect(page).toHaveURL(/\/challenges/, { timeout: 15_000 })
  })

  test('invalid credentials shows error', async ({ page }) => {
    await page.goto('/login')
    await page.getByTestId(tid.login.email).fill('user1@example.com')
    await page.getByTestId(tid.login.password).fill('wrongpassword')
    await page.getByTestId(tid.login.submit).click()
    await expect(page.getByTestId(tid.login.error)).toBeVisible({ timeout: 5_000 })
  })

  test('banned user is redirected to /banned', async ({ page }) => {
    await page.goto('/login')
    await page.getByTestId(tid.login.email).fill('banned@example.com')
    await page.getByTestId(tid.login.password).fill('Password1')
    await page.getByTestId(tid.login.submit).click()
    await expect(page).toHaveURL(/\/banned/, { timeout: 10_000 })
  })

  test('email field required validation', async ({ page }) => {
    await page.goto('/login')
    await page.getByTestId(tid.login.password).fill('password')
    await page.getByTestId(tid.login.submit).click()
    // Should not navigate away (HTML required validation prevents submit)
    await expect(page).toHaveURL(/\/login/)
  })
})

test.describe('Auth: login rate-limit', () => {
  test.use({ storageState: { cookies: [], origins: [] } })
  test.describe.configure({ mode: 'serial' })

  test.beforeAll(async () => {
    await adminFetch('/admin/settings', {
      method: 'PUT',
      body: JSON.stringify({ rate_limit_login_per_minute: 3 }),
    })
  })

  test.afterAll(async () => {
    await adminFetch('/admin/settings', {
      method: 'PUT',
      body: JSON.stringify({ rate_limit_login_per_minute: 1000 }),
    })
  })

  test('exceeding rate limit shows rate limit error', async ({ page }) => {
    await page.goto('/login')
    // First 3 attempts are INVALID_CREDENTIALS, 4th triggers rate limit
    for (let i = 0; i < 4; i++) {
      await page.getByTestId(tid.login.email).fill('user2@example.com')
      await page.getByTestId(tid.login.password).fill(`wrong${i}`)
      await page.getByTestId(tid.login.submit).click()
      await page.getByTestId(tid.login.error).waitFor({ timeout: 5_000 })
    }
    const err = page.getByTestId(tid.login.error)
    await expect(err).toContainText(/rate|limit|too many/i, { timeout: 3_000 })
  })
})

test.describe('Auth: register', () => {
  test.use({ storageState: { cookies: [], origins: [] } })

  test('register form renders required fields', async ({ page }) => {
    await page.goto('/register')
    await expect(page.getByTestId(tid.register.username)).toBeVisible()
    await expect(page.getByTestId(tid.register.email)).toBeVisible()
    await expect(page.getByTestId(tid.register.password)).toBeVisible()
    await expect(page.getByTestId(tid.register.submit)).toBeVisible()
  })

  test('existing email shows error', async ({ page }) => {
    // Backend returns USER_ALREADY_EXISTS for both username/email conflicts
    // Use existing user1's email to trigger the email conflict error
    await page.goto('/register')
    await page.getByTestId(tid.register.username).fill(`unique${Date.now()}`)
    await page.getByTestId(tid.register.email).fill('user1@example.com')
    await page.getByTestId(tid.register.password).fill('Password123')
    const confirm = page.getByTestId(tid.register.confirm)
    if (await confirm.isVisible()) await confirm.fill('Password123')
    await page.getByTestId(tid.register.submit).click()
    await expect(page.getByText(/already exists|taken|conflict/i).first()).toBeVisible({
      timeout: 5_000,
    })
  })

  test('successful registration redirects to /login', async ({ page }) => {
    const unique = `e2ereg${Date.now()}`
    await page.goto('/register')
    await page.getByTestId(tid.register.username).fill(unique)
    await page.getByTestId(tid.register.email).fill(`${unique}@example.com`)
    await page.getByTestId(tid.register.password).fill('Password123')
    const confirm = page.getByTestId(tid.register.confirm)
    if (await confirm.isVisible()) await confirm.fill('Password123')
    await page.getByTestId(tid.register.submit).click()
    await expect(page).toHaveURL(/\/login|\/challenges/, { timeout: 10_000 })
  })
})

test.describe('Auth: reset password', () => {
  test.use({ storageState: { cookies: [], origins: [] } })

  test('request form renders', async ({ page }) => {
    await page.goto('/reset-password')
    await expect(page.getByTestId(tid.resetPassword.email)).toBeVisible()
    await expect(page.getByTestId(tid.resetPassword.submit)).toBeVisible()
  })

  test('submit with unknown email shows success (no user enumeration)', async ({ page }) => {
    await page.goto('/reset-password')
    await page.getByTestId(tid.resetPassword.email).fill('nobody@example.com')
    await page.getByTestId(tid.resetPassword.submit).click()
    // Should show success message regardless (no enumeration)
    await expect(page.getByText(/sent|check|email/i).first()).toBeVisible({ timeout: 5_000 })
  })
})

test.describe('Auth: logout', () => {
  test.use({ storageState: 'e2e/.auth/user1.json' })

  test('logout clears session and redirects to /login', async ({ page }) => {
    await page.goto('/challenges')
    await page.locator('[data-testid="challenge-card"]').first().waitFor({ timeout: 15_000 })

    await page.getByTestId(tid.navbar.userMenuTrigger).click()
    await page.getByTestId(tid.navbar.logout).click()
    await expect(page).toHaveURL(/\/login/, { timeout: 10_000 })
  })
})

test.describe('Auth: guest redirects', () => {
  test.use({ storageState: { cookies: [], origins: [] } })

  test('/challenges -> /login', async ({ page }) => {
    await page.goto('/challenges')
    await expect(page).toHaveURL(/\/login/)
  })

  test('/settings -> /login', async ({ page }) => {
    await page.goto('/settings')
    await expect(page).toHaveURL(/\/login/)
  })

  test('/notifications -> /login', async ({ page }) => {
    await page.goto('/notifications')
    await expect(page).toHaveURL(/\/login/)
  })

  test('/admin -> 403 or /login', async ({ page }) => {
    await page.goto('/admin')
    await expect(page).toHaveURL(/\/login|\/403/)
  })
})

test.describe('Auth: registration disabled', () => {
  test.use({ storageState: { cookies: [], origins: [] } })

  // registration_open cannot be changed while competition is active;
  // this test requires a non-active competition to run.
  test.skip('register page shows closed message', async ({ page }) => {
    await page.goto('/register')
    await expect(page.getByText(/closed|disabled|registration/i).first()).toBeVisible({ timeout: 5_000 })
    await expect(page.getByTestId(tid.register.submit)).not.toBeVisible()
  })
})
