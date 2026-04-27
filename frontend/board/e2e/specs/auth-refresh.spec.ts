/**
 * Auth refresh spec - silent token refresh on 401.
 *
 * The authMiddleware intercepts a 401 from any API call, exchanges the refresh
 * token for a new access token, then retries the original request transparently.
 * The user must stay on the page and see the data load - no redirect to /login.
 */
import { expect, test } from '@playwright/test'

test.describe('Auth: silent refresh on 401', () => {
  test.use({ storageState: 'e2e/.auth/user1.json' })

  test('401 on challenges triggers refresh+retry, user stays on /challenges', async ({ page }) => {
    // Intercept the challenges list endpoint once. On the first call we return
    // 401 (simulating an expired access token). The authMiddleware should then
    // refresh and retry - the second call passes through unmodified.
    let intercepted = false
    await page.route(/\/api\/v1\/challenges(\?|$)/, async (route) => {
      if (!intercepted) {
        intercepted = true
        await route.fulfill({
          status: 401,
          contentType: 'application/json',
          body: JSON.stringify({ code: 'INVALID_TOKEN', message: 'token expired' }),
        })
      } else {
        await route.continue()
      }
    })

    await page.goto('/challenges')

    // Challenge cards must appear - meaning the retry succeeded after refresh.
    await expect(page.locator('[data-testid="challenge-card"]').first()).toBeVisible({
      timeout: 20_000,
    })
    // User was not kicked to /login
    await expect(page).toHaveURL(/\/challenges/)
  })

  test('logout happens when refresh fails after 401', async ({ page }) => {
    // Return 401 for every challenges request AND every refresh attempt.
    // The middleware will try to refresh, fail, and call logout() which redirects to /login.
    await page.route(/\/api\/v1\/challenges(\?|$)/, (route) =>
      route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({ code: 'INVALID_TOKEN', message: 'token expired' }),
      }),
    )
    await page.route(/\/api\/v1\/auth\/refresh/, (route) =>
      route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({ code: 'INVALID_TOKEN', message: 'refresh failed' }),
      }),
    )

    await page.goto('/challenges')

    await expect(page).toHaveURL(/\/login/, { timeout: 15_000 })
  })
})
