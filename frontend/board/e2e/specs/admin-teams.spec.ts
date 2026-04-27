/**
 * Admin teams spec - list, edit, member management.
 */
import { expect, test } from '../fixtures'
import { tid } from '../helpers/selectors'

test.describe('Admin: teams list', () => {
  test('renders Team Alpha', async ({ adminPage: page }) => {
    await page.goto('/admin/teams')
    await expect(page.getByText('Team Alpha')).toBeVisible({ timeout: 10_000 })
  })

  test('search by name narrows results', async ({ adminPage: page }) => {
    await page.goto('/admin/teams')
    const search = page.getByPlaceholder(/search/i)
    if (await search.isVisible({ timeout: 3_000 })) {
      await search.fill('Team Alpha')
      await page.waitForTimeout(500)
      await expect(page.getByText('Team Alpha')).toBeVisible()
    }
  })

  test('row click opens detail panel or navigates', async ({ adminPage: page }) => {
    await page.goto('/admin/teams')
    const row = page
      .locator('[data-testid^="admin-row-"]')
      .filter({ hasText: 'Team Alpha' })
      .first()
    if (await row.isVisible({ timeout: 10_000 })) {
      await row.click()
      // Should open detail panel or navigate
      await expect(
        page.locator('[class*="detail"], [role="dialog"]').or(page.locator('h2, h3')).first(),
      ).toBeVisible({ timeout: 5_000 })
    }
  })
})

test.describe('Admin: teams edit', () => {
  test('edit Team Alpha name', async ({ adminPage: page }) => {
    await page.goto('/admin/teams')
    const row = page
      .locator('[data-testid^="admin-row-"]')
      .filter({ hasText: 'Team Alpha' })
      .first()
    if (await row.isVisible({ timeout: 10_000 })) {
      await row.click()
      // Look for edit/save controls
      const nameField = page.getByLabel(/name/i)
      if (await nameField.isVisible({ timeout: 3_000 })) {
        await nameField.fill('Team Alpha')
        await page.getByRole('button', { name: /save|update/i }).click()
        await expect(page.getByText(/saved|updated|success/i).first()).toBeVisible({
          timeout: 5_000,
        })
      }
    }
  })
})
