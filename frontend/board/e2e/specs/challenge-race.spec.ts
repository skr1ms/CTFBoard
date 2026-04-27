/**
 * Challenge race spec - concurrent flag submissions from two users.
 *
 * Two authenticated browser contexts submit the same flag at the same time.
 * Both must receive a status response (correct, already-solved, or error) within
 * a reasonable timeout - no crash, no silent hang.
 *
 * Because user1 (Team Alpha) and user2 are separate scoring entities, both
 * submissions are valid and the backend must handle them concurrently without
 * deadlock or missing one response.
 */
import { expect, test } from '../fixtures'
import { tid } from '../helpers/selectors'

test.describe('Challenge: concurrent flag submissions', () => {
  test('two users submitting the same flag simultaneously both get a status', async ({
    user1Page,
    user2Page,
  }) => {
    // Navigate both contexts to the challenges page in parallel.
    await Promise.all([user1Page.goto('/challenges'), user2Page.goto('/challenges')])

    // Wait for challenge cards on both pages.
    await Promise.all([
      user1Page.locator('[data-testid="challenge-card"]').first().waitFor({ timeout: 20_000 }),
      user2Page.locator('[data-testid="challenge-card"]').first().waitFor({ timeout: 20_000 }),
    ])

    // Open the "Hello World" challenge on both pages.
    await Promise.all([
      user1Page.getByText('Hello World').first().click(),
      user2Page.getByText('Hello World').first().click(),
    ])

    // Wait for the modal to open on both pages.
    await Promise.all([
      user1Page.getByTestId(tid.challengeModal.root).waitFor({ timeout: 10_000 }),
      user2Page.getByTestId(tid.challengeModal.root).waitFor({ timeout: 10_000 }),
    ])

    // Fill the correct flag on both pages.
    await Promise.all([
      user1Page.getByTestId(tid.flagSubmit.input).fill('CTF{hello_world}'),
      user2Page.getByTestId(tid.flagSubmit.input).fill('CTF{hello_world}'),
    ])

    // Submit from both simultaneously.
    await Promise.all([
      user1Page.getByTestId(tid.flagSubmit.button).click(),
      user2Page.getByTestId(tid.flagSubmit.button).click(),
    ])

    // Both must show a flag-submit-status element - the specific text (correct /
    // already-solved / error) depends on prior test state and is not asserted here.
    await Promise.all([
      expect(user1Page.getByTestId(tid.flagSubmit.status)).toBeVisible({ timeout: 15_000 }),
      expect(user2Page.getByTestId(tid.flagSubmit.status)).toBeVisible({ timeout: 15_000 }),
    ])
  })
})
