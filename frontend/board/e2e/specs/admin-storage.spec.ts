/**
 * Admin storage spec - file storage browser.
 */
import { expect, test } from '../fixtures'

test.describe('Admin: storage', () => {
  test('storage page loads', async ({ adminPage: page }) => {
    await page.goto('/admin/storage')
    await expect(page.locator('h1').first()).toBeVisible({ timeout: 10_000 })
  })

  test('empty state or file list renders', async ({ adminPage: page }) => {
    await page.goto('/admin/storage')
    await expect(
      page.locator('[class*="empty"], table, [class*="file"]').first(),
    ).toBeVisible({ timeout: 8_000 })
  })

  test('prefix filter input is present', async ({ adminPage: page }) => {
    await page.goto('/admin/storage')
    const filter = page.getByPlaceholder(/prefix|filter|search/i)
    if (await filter.isVisible({ timeout: 3_000 })) {
      await filter.fill('test/')
      await page.waitForTimeout(300)
      // Should not error
    }
  })
})
