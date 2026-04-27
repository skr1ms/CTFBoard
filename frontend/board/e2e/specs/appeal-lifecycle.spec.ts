/**
 * Appeal lifecycle spec - end-to-end flow:
 * 1. Banned user lands on /banned and submits an appeal.
 * 2. Admin reviews and resolves the appeal via /admin/appeals.
 * 3. User regains access to protected pages.
 */
import { expect, test } from '../fixtures'
import { adminFetch, banUser } from '../helpers/api'

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

async function getBannedUserId(): Promise<string> {
  const res = (await adminFetch('/admin/users?q=banned&field=username&per_page=1')) as {
    data?: Array<{ id: string }>
  }
  const id = res.data?.[0]?.id
  if (!id) throw new Error('banned user not found')
  return id
}

// ---------------------------------------------------------------------------

test.describe('Ban appeal lifecycle', () => {
  let bannedUserId: string

  test.beforeAll(async () => {
    bannedUserId = await getBannedUserId()
  })

  test.afterEach(async () => {
    // Restore: re-ban the user so subsequent specs using bannedPage find the
    // expected state.
    await banUser(bannedUserId, 'E2E banned fixture')
  })

  test('banned user submits appeal, admin resolves it, user regains access', async ({
    bannedPage,
    adminPage,
  }) => {
    // ── Step 1: user is on /banned ─────────────────────────────────────────
    await bannedPage.goto('/challenges')
    await expect(bannedPage).toHaveURL(/\/banned/, { timeout: 10_000 })

    // ── Step 2: submit appeal if form is visible ───────────────────────────
    // The form is hidden when a pending appeal already exists.
    const textarea = bannedPage.locator('textarea').first()
    const formVisible = await textarea.isVisible().catch(() => false)
    if (formVisible) {
      await textarea.fill(
        'I am appealing this ban. I believe it was applied in error and I have not violated any competition rules.',
      )
      await bannedPage.getByRole('button', { name: /submit.*appeal|submit appeal/i }).click()
      await expect(bannedPage.getByText(/pending/i).first()).toBeVisible({ timeout: 8_000 })
    }
    // If form is not visible a pending appeal already exists - proceed to admin review.

    // ── Step 3: admin resolves the appeal ─────────────────────────────────
    await adminPage.goto('/admin/appeals')
    const pendingTab = adminPage.getByRole('button', { name: /^pending$/i })
    if (await pendingTab.isVisible({ timeout: 3_000 }).catch(() => false)) {
      await pendingTab.click()
    }

    const reviewBtn = adminPage.getByRole('button', { name: /^review$/i }).first()
    await expect(reviewBtn).toBeVisible({ timeout: 8_000 })
    await reviewBtn.click()

    // Default decision is "resolved (unban)" - submit directly
    await adminPage.getByRole('button', { name: /^submit$/i }).click()
    await expect(adminPage.getByText(/appeal reviewed/i)).toBeVisible({ timeout: 8_000 })

    // ── Step 4: full page reload makes authStore re-hydrate with unbanned state
    await bannedPage.goto('/challenges')
    await expect(bannedPage).toHaveURL(/\/challenges/, { timeout: 15_000 })
  })
})
