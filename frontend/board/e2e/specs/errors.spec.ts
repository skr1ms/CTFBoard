/**
 * Error pages spec - 403, 404, 429, 500, unknown route, guest redirect.
 */
import { expect, test } from '@playwright/test'

test.describe('Error pages: guest', () => {
  test.use({ storageState: { cookies: [], origins: [] } })

  test('/403 renders forbidden page', async ({ page }) => {
    await page.goto('/403')
    await expect(page.getByText(/403|forbidden|access denied/i).first()).toBeVisible()
  })

  test('/404 renders not found page', async ({ page }) => {
    await page.goto('/404')
    await expect(page.getByText(/404|not found|lost/i).first()).toBeVisible()
  })

  test('/429 renders rate limit page', async ({ page }) => {
    await page.goto('/429')
    await expect(page.getByText(/429|too many|rate limit/i).first()).toBeVisible()
  })

  test('/500 renders server error page', async ({ page }) => {
    await page.goto('/500')
    await expect(page.getByText(/500|server error|unexpected/i).first()).toBeVisible()
  })

  test('unknown route shows 404', async ({ page }) => {
    await page.goto('/this-page-does-not-exist-xyz')
    await expect(page.getByText(/404|not found|lost/i).first()).toBeVisible({ timeout: 5_000 })
  })
})
