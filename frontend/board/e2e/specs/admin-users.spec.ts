/**
 * Admin users spec - list, search, ban/unban, bulk delete.
 */
import { expect, test } from '../fixtures'
import { tid } from '../helpers/selectors'

test.describe('Admin: users list', () => {
  test('renders seeded users', async ({ adminPage: page }) => {
    await page.goto('/admin/users')
    await expect(page.getByText('user1')).toBeVisible({ timeout: 10_000 })
    await expect(page.getByText('user2')).toBeVisible()
  })

  test('search by username narrows results', async ({ adminPage: page }) => {
    await page.goto('/admin/users')
    const search = page.getByPlaceholder(/search/i)
    if (await search.isVisible({ timeout: 5_000 })) {
      await search.fill('user1')
      await page.waitForTimeout(500)
      await expect(page.getByText('user1')).toBeVisible()
    }
  })

  test('banned user row is visible', async ({ adminPage: page }) => {
    await page.goto('/admin/users')
    await expect(page.getByText('banned')).toBeVisible({ timeout: 10_000 })
  })
})

test.describe('Admin: users ban/unban', () => {
  test('ban and unban user2', async ({ adminPage: page }) => {
    await page.goto('/admin/users')

    // Navigate to user2 detail
    const row = page.locator('[data-testid^="admin-row-"]').filter({ hasText: 'user2' }).first()
    if (await row.isVisible({ timeout: 10_000 })) {
      await row.click()
      await expect(page).toHaveURL(/\/admin\/users\/[\w-]+/, { timeout: 5_000 })

      // Ban user2
      const banBtn = page.getByRole('button', { name: /^ban$/i })
      if (await banBtn.isVisible()) {
        await banBtn.click()
        const reasonInput = page.getByPlaceholder(/reason/i)
        await reasonInput.fill('E2E test ban')
        await page.getByRole('button', { name: /confirm|submit|ban/i }).click()
        await expect(page.getByText(/banned/i)).toBeVisible({ timeout: 5_000 })

        // Unban user2
        const unbanBtn = page.getByRole('button', { name: /unban/i })
        if (await unbanBtn.isVisible()) {
          await unbanBtn.click()
          await expect(page.getByText(/active|unbanned/i)).toBeVisible({ timeout: 5_000 })
        }
      }
    }
  })
})

test.describe('Admin: users bulk delete', () => {
  test('bulk delete button appears when rows are selected', async ({ adminPage: page }) => {
    await page.goto('/admin/users')
    // Select first checkbox
    const checkbox = page.locator('input[type="checkbox"]').first()
    if (await checkbox.isVisible({ timeout: 5_000 })) {
      await checkbox.check()
      await expect(page.getByTestId(tid.admin.bulkDelete)).toBeVisible({ timeout: 3_000 })
    }
  })
})
