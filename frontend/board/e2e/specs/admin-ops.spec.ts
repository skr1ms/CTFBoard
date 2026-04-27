/**
 * Admin operations spec - competition settings, configs, settings toggles, scoreboard admin.
 */
import { expect, test } from '../fixtures'

test.describe('Admin: competition', () => {
  test('competition page loads and shows active status', async ({ adminPage: page }) => {
    await page.goto('/admin/competition')
    await expect(page.getByText(/active|running/i).first()).toBeVisible({ timeout: 10_000 })
  })

  test('competition name field is visible', async ({ adminPage: page }) => {
    await page.goto('/admin/competition')
    const nameInput = page.getByLabel(/name/i).first()
    await expect(nameInput).toBeVisible({ timeout: 8_000 })
    await expect(nameInput).toHaveValue('E2E CTF')
  })
})

test.describe('Admin: configs', () => {
  test('configs page loads', async ({ adminPage: page }) => {
    await page.goto('/admin/configs')
    await expect(page.locator('h1').first()).toBeVisible({ timeout: 10_000 })
  })

  test('visibility panel renders select dropdowns', async ({ adminPage: page }) => {
    await page.goto('/admin/configs')
    // Should have visibility config selects
    await expect(page.locator('select').first()).toBeVisible({ timeout: 8_000 })
  })
})

test.describe('Admin: settings', () => {
  test('settings page loads', async ({ adminPage: page }) => {
    await page.goto('/admin/settings')
    await expect(page.locator('h1').first()).toBeVisible({ timeout: 10_000 })
  })

  test('registration toggle is present', async ({ adminPage: page }) => {
    await page.goto('/admin/settings')
    const toggle = page
      .getByLabel(/registration/i)
      .or(page.locator('[class*="toggle"]').first())
    await expect(toggle).toBeVisible({ timeout: 8_000 })
  })
})

test.describe('Admin: scoreboard', () => {
  test('admin scoreboard page loads', async ({ adminPage: page }) => {
    await page.goto('/admin/scoreboard')
    await expect(page.locator('h1, h2').first()).toBeVisible({ timeout: 10_000 })
  })
})

test.describe('Admin: submissions', () => {
  test('submissions page loads', async ({ adminPage: page }) => {
    await page.goto('/admin/submissions')
    await expect(page.locator('h1, h2, table, [class*="empty"]').first()).toBeVisible({
      timeout: 10_000,
    })
  })
})
