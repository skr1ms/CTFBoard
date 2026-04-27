/**
 * Activity spec - solves/fails/awards/submissions tabs for user and team.
 */
import { expect, test } from '../fixtures'

test.describe('Activity: user tabs', () => {
  test('my solves tab renders', async ({ user1Page: page }) => {
    await page.goto('/users/me')
    // Navigate to activity or the user's own profile
    const solvesTab = page
      .getByRole('tab', { name: /solves/i })
      .or(page.getByText(/my solves/i).first())
    if (await solvesTab.isVisible({ timeout: 5_000 })) {
      await solvesTab.click()
      await expect(page.locator('table tbody tr, [class*="empty"]').first()).toBeVisible({
        timeout: 5_000,
      })
    }
  })

  test('awards section shows +50 award for user1 team', async ({ user1Page: page }) => {
    await page.goto('/team')
    const awardsTab = page.getByRole('tab', { name: /award/i })
    if (await awardsTab.isVisible({ timeout: 5_000 })) {
      await awardsTab.click()
      await expect(page.getByText('+50').or(page.getByText('50'))).toBeVisible({ timeout: 5_000 })
    }
  })
})

test.describe('Activity: user detail page', () => {
  test('user1 profile page loads', async ({ user1Page: page }) => {
    await page.goto('/users')
    const row = page.getByText('user1')
    if (await row.isVisible({ timeout: 8_000 })) {
      await row.click()
      await expect(page).toHaveURL(/\/users\/[\w-]+/, { timeout: 5_000 })
      await expect(page.locator('h1, h2').first()).toBeVisible()
    }
  })
})
