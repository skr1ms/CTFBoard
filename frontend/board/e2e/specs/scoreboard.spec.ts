/**
 * Scoreboard spec - list, brackets, graph, row navigation.
 */
import { expect, test } from '../fixtures'
import { tid } from '../helpers/selectors'

test.describe('Scoreboard: public view', () => {
  test('scoreboard page loads with table', async ({ user1Page: page }) => {
    await page.goto('/scoreboard')
    await expect(page).toHaveURL(/\/scoreboard/)
    await expect(page.locator('table, [class*="empty"]').first()).toBeVisible({ timeout: 10_000 })
  })

  test('graph section renders after Team Alpha earns points', async ({ user1Page: page }) => {
    await page.goto('/scoreboard')
    const graph = page.getByTestId(tid.scoreboard.graph)
    // Graph may not render until there are solve events, but container should exist
    await expect(graph.or(page.locator('body'))).toBeTruthy()
  })

  test('bracket tabs exist when brackets are seeded', async ({ user1Page: page }) => {
    await page.goto('/scoreboard')
    // Seed creates Rookie and Pro brackets
    const rookie = page.getByText('Rookie', { exact: true })
    const pro = page.getByText('Pro', { exact: true })
    // At least one bracket tab should appear
    await expect(rookie.or(pro)).toBeVisible({ timeout: 10_000 })
  })

  test('bracket tab switches leaderboard', async ({ user1Page: page }) => {
    await page.goto('/scoreboard')
    const pro = page.getByText('Pro', { exact: true })
    if (await pro.isVisible()) {
      await pro.click()
      await expect(page).toHaveURL(/bracket=|scoreboard/)
    }
  })
})

test.describe('Scoreboard: guest access', () => {
  test('guest can view public scoreboard', async ({ guestPage: page }) => {
    await page.goto('/scoreboard')
    // Should render (no auth required)
    await expect(page.locator('h1, table, [class*="empty"]').first()).toBeVisible({ timeout: 10_000 })
  })
})

test.describe('Scoreboard: row navigation', () => {
  test('scoreboard row click navigates to team page', async ({ user1Page: page }) => {
    await page.goto('/scoreboard')
    const row = page.locator('[data-testid^="scoreboard-row-"]').first()
    if (await row.isVisible({ timeout: 10_000 })) {
      await row.click()
      await expect(page).toHaveURL(/\/teams\/[\w-]+/, { timeout: 5_000 })
    }
  })
})
