/**
 * Visibility spec - challenge_visibility, score_visibility, account_visibility configs.
 * Tests run serially because they mutate global config.
 */
import { expect, test } from '../fixtures'
import { setConfig } from '../helpers/api'

test.describe('Visibility: challenge_visibility', () => {
  test.describe.configure({ mode: 'serial' })

  test('public - guests can see challenge list', async ({ guestPage: page }) => {
    await setConfig('challenge_visibility', 'public')
    await page.goto('/challenges')
    await expect(page.locator('[data-testid="challenge-card"]').first()).toBeVisible({
      timeout: 15_000,
    })
  })

  test('private - guests see sign-in card, not challenges', async ({ guestPage: page }) => {
    await setConfig('challenge_visibility', 'private')
    await page.goto('/challenges')
    await expect(
      page.getByText(/sign in|login|register/i).first(),
    ).toBeVisible({ timeout: 8_000 })
    await expect(page.locator('[data-testid="challenge-card"]')).toHaveCount(0)
  })

  test.afterAll(async () => {
    // Reset to default
    await setConfig('challenge_visibility', 'public')
  })
})

test.describe('Visibility: score_visibility', () => {
  test.describe.configure({ mode: 'serial' })

  test('public - scoreboard visible to guests', async ({ guestPage: page }) => {
    await setConfig('score_visibility', 'public')
    await page.goto('/scoreboard')
    await expect(page.locator('table, [class*="empty"]').first()).toBeVisible({ timeout: 10_000 })
  })

  test('hidden - guests see restricted card', async ({ guestPage: page }) => {
    await setConfig('score_visibility', 'hidden')
    await page.goto('/scoreboard')
    await expect(
      page.getByText(/sign in|login|hidden|private/i).first(),
    ).toBeVisible({ timeout: 8_000 })
  })

  test.afterAll(async () => {
    await setConfig('score_visibility', 'public')
  })
})
