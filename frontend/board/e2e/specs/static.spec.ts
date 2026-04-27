/**
 * Static pages spec - published pages, draft pages, ToS, Privacy.
 */
import { expect, test } from '@playwright/test'

test.describe('Static pages', () => {
  test.use({ storageState: { cookies: [], origins: [] } })

  test('/pages/rules renders published page', async ({ page }) => {
    await page.goto('/pages/rules')
    // The page is published, should render content
    await expect(page.getByText(/Rules/i).first()).toBeVisible({ timeout: 8_000 })
  })

  test('/pages/draft-me returns not found for guests', async ({ page }) => {
    await page.goto('/pages/draft-me')
    // Draft pages are not publicly visible
    await expect(page.getByText(/not found|404|does not exist/i).first()).toBeVisible({
      timeout: 5_000,
    })
  })

  test('/tos renders terms of service', async ({ page }) => {
    await page.goto('/tos')
    await expect(page.getByText(/terms|service/i).first()).toBeVisible({ timeout: 5_000 })
  })

  test('/privacy renders privacy policy', async ({ page }) => {
    await page.goto('/privacy')
    await expect(page.getByText(/privacy/i).first()).toBeVisible({ timeout: 5_000 })
  })
})
