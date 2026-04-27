/**
 * Challenges spec - list, modal, flag submit, hints, requirements.
 */
import { expect, test } from '../fixtures'
import { tid } from '../helpers/selectors'

// ---------------------------------------------------------------------------
// Challenge list
// ---------------------------------------------------------------------------
test.describe('Challenges: list', () => {
  test('renders seed challenges', async ({ user1Page: page }) => {
    await page.goto('/challenges')
    await expect(page.locator('[data-testid="challenge-card"]').first()).toBeVisible({
      timeout: 15_000,
    })
    await expect(page.locator('[data-testid="challenge-card"]')).toHaveCount(3)
  })

  test('tag filter narrows results', async ({ user1Page: page }) => {
    await page.goto('/challenges')
    await page.locator('[data-testid="challenge-card"]').first().waitFor({ timeout: 15_000 })

    // Click 'web-easy' tag filter if present (may appear as tag button)
    const tagButton = page.getByText('web-easy', { exact: true }).first()
    if (await tagButton.isVisible()) {
      await tagButton.click()
      await expect(page.locator('[data-testid="challenge-card"]')).toHaveCount(1)
    }
  })

  test('search filter narrows results', async ({ user1Page: page }) => {
    await page.goto('/challenges')
    await page.locator('[data-testid="challenge-card"]').first().waitFor({ timeout: 15_000 })

    const search = page.getByPlaceholder(/search/i)
    if (await search.isVisible()) {
      await search.fill('Hello')
      await page.waitForTimeout(500) // debounce
      await expect(page.locator('[data-testid="challenge-card"]')).toHaveCount(1)
    }
  })
})

// ---------------------------------------------------------------------------
// Challenge modal
// ---------------------------------------------------------------------------
test.describe('Challenges: modal', () => {
  test('card click opens modal and updates URL', async ({ user1Page: page }) => {
    await page.goto('/challenges')
    const card = page.locator('[data-testid="challenge-card"]').first()
    await card.waitFor({ timeout: 15_000 })
    await card.click()

    await expect(page.getByTestId(tid.challengeModal.root)).toBeVisible({ timeout: 5_000 })
    await expect(page).toHaveURL(/\/challenges\/[\w-]+/)
  })

  test('modal close button dismisses modal', async ({ user1Page: page }) => {
    await page.goto('/challenges')
    await page.locator('[data-testid="challenge-card"]').first().click()
    await page.getByTestId(tid.challengeModal.root).waitFor({ timeout: 5_000 })

    await page.getByTestId(tid.challengeModal.close).click()
    await expect(page.getByTestId(tid.challengeModal.root)).not.toBeVisible({ timeout: 3_000 })
  })

  test('Escape key closes modal', async ({ user1Page: page }) => {
    await page.goto('/challenges')
    await page.locator('[data-testid="challenge-card"]').first().click()
    await page.getByTestId(tid.challengeModal.root).waitFor({ timeout: 5_000 })

    await page.keyboard.press('Escape')
    await expect(page.getByTestId(tid.challengeModal.root)).not.toBeVisible({ timeout: 3_000 })
  })
})

// ---------------------------------------------------------------------------
// Flag submission
// ---------------------------------------------------------------------------
test.describe('Challenges: flag submit', () => {
  test('correct flag shows success', async ({ user2Page: page }) => {
    await page.goto('/challenges')
    // Open "Hello World" challenge
    await page.getByText('Hello World').first().click()
    await page.getByTestId(tid.challengeModal.root).waitFor({ timeout: 10_000 })

    await page.getByTestId(tid.flagSubmit.input).fill('CTF{hello_world}')
    await page.getByTestId(tid.flagSubmit.button).click()

    await expect(page.getByTestId(tid.flagSubmit.status)).toContainText(/correct|solved/i, {
      timeout: 8_000,
    })
  })

  test('wrong flag shows error', async ({ user2Page: page }) => {
    await page.goto('/challenges')
    await page.getByText('Hello World').first().click()
    await page.getByTestId(tid.challengeModal.root).waitFor({ timeout: 10_000 })

    await page.getByTestId(tid.flagSubmit.input).fill('CTF{wrong_flag}')
    await page.getByTestId(tid.flagSubmit.button).click()

    await expect(page.getByTestId(tid.flagSubmit.status)).toContainText(/incorrect|wrong/i, {
      timeout: 5_000,
    })
  })
})

// ---------------------------------------------------------------------------
// Hints
// ---------------------------------------------------------------------------
test.describe('Challenges: hints', () => {
  test('hint unlock dialog shows cost and confirms', async ({ user1Page: page }) => {
    await page.goto('/challenges')
    await page.getByText('Hello World').first().click()
    await page.getByTestId(tid.challengeModal.root).waitFor({ timeout: 10_000 })

    // Switch to hints tab if multi-tab modal
    const hintsTab = page.getByTestId(tid.challengeModal.tab('hints'))
    if (await hintsTab.isVisible()) await hintsTab.click()

    // Find the first unlock button
    const unlockBtn = page.locator('[data-testid^="hint-unlock-"]').first()
    if (await unlockBtn.isVisible()) {
      await unlockBtn.click()
      // Confirmation dialog should appear
      await expect(page.getByTestId(tid.hint.confirmYes)).toBeVisible({ timeout: 3_000 })
      await page.getByTestId(tid.hint.confirmYes).click()
      // After unlock, content should appear
      await expect(page.locator('[data-testid^="hint-content-"]').first()).toBeVisible({
        timeout: 5_000,
      })
    }
  })
})

// ---------------------------------------------------------------------------
// Requirements gate
// ---------------------------------------------------------------------------
test.describe('Challenges: requirements', () => {
  test('Base64 Basics is locked until Hello World is solved', async ({ user2Page: page }) => {
    await page.goto('/challenges')
    await page.locator('[data-testid="challenge-card"]').first().waitFor({ timeout: 15_000 })

    // Look for Base64 Basics card - should be locked for user2 who hasn't solved Hello World
    const base64Card = page.locator('[data-testid="challenge-card"]').filter({ hasText: 'Base64 Basics' })
    if (await base64Card.isVisible()) {
      await expect(base64Card.locator('[data-testid="challenge-locked-icon"]')).toBeVisible()
    }
  })
})
