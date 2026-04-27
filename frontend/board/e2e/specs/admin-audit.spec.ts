/**
 * Admin audit spec - statistics, unlocks, appeals, backup.
 */
import { expect, test } from '../fixtures'

test.describe('Admin: statistics', () => {
  test('statistics page loads with cards', async ({ adminPage: page }) => {
    await page.goto('/admin/statistics')
    await expect(page.locator('h1').first()).toBeVisible({ timeout: 10_000 })
    // Stats cards
    await expect(
      page.locator('[class*="card"], [class*="stat"]').first(),
    ).toBeVisible({ timeout: 8_000 })
  })
})

test.describe('Admin: unlocks', () => {
  test('unlocks page loads', async ({ adminPage: page }) => {
    await page.goto('/admin/unlocks')
    await expect(
      page.locator('h1, table, [class*="empty"]').first(),
    ).toBeVisible({ timeout: 10_000 })
  })
})

test.describe('Admin: appeals', () => {
  test('appeals page loads', async ({ adminPage: page }) => {
    // Appeals/appeals-related admin path
    const paths = ['/admin/appeals', '/admin/bans']
    for (const p of paths) {
      await page.goto(p)
      if (page.url().includes(p.split('/').pop()!)) {
        await expect(
          page.locator('h1, table, [class*="empty"]').first(),
        ).toBeVisible({ timeout: 8_000 })
        return
      }
    }
  })
})

test.describe('Admin: backup', () => {
  test('backup page loads', async ({ adminPage: page }) => {
    await page.goto('/admin/backup')
    await expect(page.locator('h1').first()).toBeVisible({ timeout: 10_000 })
  })

  test('export buttons are visible', async ({ adminPage: page }) => {
    await page.goto('/admin/backup')
    await expect(
      page.getByRole('button', { name: /export|download/i }).first(),
    ).toBeVisible({ timeout: 8_000 })
  })
})
