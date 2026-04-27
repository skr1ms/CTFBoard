/**
 * Team invite spec - full invite lifecycle via UI.
 *
 * 1. user1 (captain of Team Alpha) is on /team and copies the invite link.
 *    We retrieve the current invite token via the API instead of the clipboard.
 * 2. user2 navigates to /team/enroll and joins using the token.
 * 3. user2 lands on /team and sees Team Alpha.
 * 4. user1 refreshes /team and sees user2 in the member list.
 *
 * Cleanup: user2 is removed from Team Alpha after the test.
 */
import { expect, test } from '../fixtures'
import { adminFetch } from '../helpers/api'

async function getTeamAlphaInviteToken(): Promise<string> {
  const userList = (await adminFetch('/admin/users?q=user1&field=username&per_page=5')) as {
    data?: Array<{ id?: string; team_id?: string }>
  }
  const user1 = userList.data?.[0]
  if (!user1?.team_id) throw new Error('user1 is not on a team')

  const team = (await adminFetch(`/admin/teams/${user1.team_id}`)) as {
    invite_token?: string | null
  }
  if (!team.invite_token) throw new Error('Team Alpha has no invite_token')
  return team.invite_token
}

async function removeUser2FromTeam(): Promise<void> {
  const userList = (await adminFetch('/admin/users?q=user2&field=username&per_page=5')) as {
    data?: Array<{ id?: string; team_id?: string }>
  }
  const user2 = userList.data?.[0]
  if (user2?.id && user2.team_id) {
    await adminFetch(`/admin/teams/${user2.team_id}/members/${user2.id}`, { method: 'DELETE' })
  }
}

test.describe('Team: invite lifecycle', () => {
  test.beforeEach(async () => {
    await removeUser2FromTeam()
  })

  test.afterEach(async () => {
    await removeUser2FromTeam()
  })

  test('user2 joins Team Alpha via invite token, both see membership', async ({
    user1Page,
    user2Page,
  }) => {
    // ── Step 1: get the invite token ──────────────────────────────────────────
    const inviteToken = await getTeamAlphaInviteToken()

    // ── Step 2: user1 sees /team and the invite copy button ───────────────────
    await user1Page.goto('/team')
    await expect(user1Page.getByText('Team Alpha')).toBeVisible({ timeout: 10_000 })
    await expect(user1Page.getByTestId('team-invite-copy')).toBeVisible({ timeout: 5_000 })

    // ── Step 3: user2 joins via the enroll form ───────────────────────────────
    await user2Page.goto('/team/enroll')

    // Fill the token input and submit
    await user2Page.getByTestId('team-join-token').fill(inviteToken)
    await user2Page.getByTestId('team-join-submit').click()

    // user2 should land on /team after successfully joining
    await expect(user2Page).toHaveURL(/\/team$/, { timeout: 10_000 })
    await expect(user2Page.getByText('Team Alpha')).toBeVisible({ timeout: 8_000 })

    // ── Step 4: user1 refreshes and sees user2 in the member list ─────────────
    await user1Page.reload()
    await user1Page.waitForTimeout(1_000) // allow query refetch
    const memberRows = user1Page.locator('[data-testid^="team-member-row-"]')
    await expect(memberRows).toHaveCount(2, { timeout: 10_000 })
  })
})
