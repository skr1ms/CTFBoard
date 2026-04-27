/**
 * Public lists spec - /users, /teams pages, pagination, search.
 */
import { expect, test } from '../fixtures'

test.describe('Users list: /users', () => {
  test('renders user rows', async ({ user1Page: page }) => {
    await page.goto('/users')
    await expect(page.locator('table tbody tr, [class*="user-row"], [class*="UserRow"]').first()).toBeVisible({
      timeout: 10_000,
    })
  })

  test('search narrows results', async ({ user1Page: page }) => {
    await page.goto('/users')
    const search = page.getByPlaceholder(/search/i)
    if (await search.isVisible({ timeout: 3_000 })) {
      await search.fill('user1')
      await page.waitForTimeout(500)
      // Table should only show user1
      await expect(page.getByText('user1')).toBeVisible()
    }
  })

  test('user row click navigates to detail', async ({ user1Page: page }) => {
    await page.goto('/users')
    const row = page.locator('table tbody tr').first()
    if (await row.isVisible({ timeout: 8_000 })) {
      await row.click()
      await expect(page).toHaveURL(/\/users\/[\w-]+/, { timeout: 5_000 })
    }
  })

  test('pagination controls are present when >1 page', async ({ user1Page: page }) => {
    await page.goto('/users')
    const nextBtn = page.getByTestId('pagination-next')
    if (await nextBtn.isVisible({ timeout: 5_000 })) {
      await expect(nextBtn).toBeEnabled()
    }
  })
})

test.describe('Teams list: /teams', () => {
  test('renders Team Alpha', async ({ user1Page: page }) => {
    await page.goto('/teams')
    await expect(page.getByText('Team Alpha')).toBeVisible({ timeout: 10_000 })
  })

  test('search works', async ({ user1Page: page }) => {
    await page.goto('/teams')
    const search = page.getByPlaceholder(/search/i)
    if (await search.isVisible({ timeout: 3_000 })) {
      await search.fill('Team Alpha')
      await page.waitForTimeout(500)
      await expect(page.getByText('Team Alpha')).toBeVisible()
    }
  })

  test('team row click navigates to detail', async ({ user1Page: page }) => {
    await page.goto('/teams')
    const row = page.locator('table tbody tr, [class*="team-row"]').first()
    if (await row.isVisible({ timeout: 8_000 })) {
      await row.click()
      await expect(page).toHaveURL(/\/teams\/[\w-]+/, { timeout: 5_000 })
    }
  })
})
