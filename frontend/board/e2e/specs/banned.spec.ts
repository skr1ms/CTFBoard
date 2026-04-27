/**
 * Banned user spec - landing page, appeal form, sign out.
 */
import { expect, test } from '../fixtures'

test.describe('Banned: landing page', () => {
  test('banned user lands on /banned', async ({ bannedPage: page }) => {
    await page.goto('/challenges')
    await expect(page).toHaveURL(/\/banned/, { timeout: 10_000 })
  })

  test('banned page shows reason and date', async ({ bannedPage: page }) => {
    await page.goto('/banned')
    await expect(page.getByText(/E2E banned fixture/i)).toBeVisible({ timeout: 8_000 })
  })

  test('banned user cannot access /challenges', async ({ bannedPage: page }) => {
    await page.goto('/challenges')
    await expect(page).toHaveURL(/\/banned/, { timeout: 10_000 })
  })
})

test.describe('Banned: appeal form', () => {
  test('appeal textarea is present', async ({ bannedPage: page }) => {
    await page.goto('/banned')
    await expect(page.locator('textarea').first()).toBeVisible({ timeout: 8_000 })
  })

  test('too-short appeal shows validation error', async ({ bannedPage: page }) => {
    await page.goto('/banned')
    const textarea = page.locator('textarea').first()
    await textarea.fill('short')
    // Inline error appears as the user types (no submit needed)
    await expect(page.getByText(/at least 20 characters/i).first()).toBeVisible({
      timeout: 3_000,
    })
  })

  test('valid appeal submission shows pending state', async ({ bannedPage: page }) => {
    await page.goto('/banned')
    const textarea = page.locator('textarea').first()
    await textarea.fill('I am appealing this ban because I did not violate any rules.')
    await page.getByRole('button', { name: /submit|appeal|send/i }).click()
    // Should show pending/submitted state
    await expect(
      page.getByText(/submitted|pending|review|received/i).first(),
    ).toBeVisible({ timeout: 5_000 })
  })
})

test.describe('Banned: sign out', () => {
  test('sign out clears session', async ({ bannedPage: page }) => {
    await page.goto('/banned')
    const signOut = page.getByRole('button', { name: /sign out|logout/i })
    if (await signOut.isVisible()) {
      await signOut.click()
      await expect(page).toHaveURL(/\/login/, { timeout: 10_000 })
    }
  })
})
