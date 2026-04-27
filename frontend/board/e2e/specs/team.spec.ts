/**
 * Team spec - enroll (create/join/solo), my-team management.
 */
import { expect, test } from '../fixtures'
import { adminFetch } from '../helpers/api'
import { tid } from '../helpers/selectors'
import * as fs from 'fs'
import * as path from 'path'
import { fileURLToPath } from 'url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

function getTeamAlphaToken(): string | null {
  const tokenFile = path.join(__dirname, '../.auth/team-alpha-token.txt')
  try {
    return fs.readFileSync(tokenFile, 'utf-8').trim()
  } catch {
    return null
  }
}

// ---------------------------------------------------------------------------
// Enroll: create
// ---------------------------------------------------------------------------
test.describe('Team: enroll create', () => {
  test('user2 can create a team and lands on /team', async ({ user2Page: page }) => {
    // Ensure user2 is not on a team (kick if needed)
    const userList = await adminFetch('/admin/users?q=user2&field=username&per_page=5') as {
      data?: Array<{ id?: string; team_id?: string }>
    }
    const user2 = userList.data?.[0]
    if (user2?.team_id) {
      await adminFetch(`/admin/teams/${user2.team_id}/members/${user2.id}`, { method: 'DELETE' })
    }

    await page.goto('/team/enroll')
    await page.getByTestId(tid.teamEnroll.createName).fill(`Team-${Date.now()}`)
    await page.getByTestId(tid.teamEnroll.createSubmit).click()
    await expect(page).toHaveURL(/\/team$/, { timeout: 10_000 })
  })
})

// ---------------------------------------------------------------------------
// Enroll: join by token
// ---------------------------------------------------------------------------
test.describe('Team: enroll join by token', () => {
  test('user2 can join Team Alpha via invite token', async ({ user2Page: page }) => {
    const inviteToken = getTeamAlphaToken()
    if (!inviteToken) {
      test.skip(true, 'Team Alpha token not available - seed may have failed')
    }

    // Ensure user2 is not on a team
    const userList2 = await adminFetch('/admin/users?q=user2&field=username&per_page=5') as {
      data?: Array<{ id?: string; team_id?: string }>
    }
    const user2 = userList2.data?.[0]
    if (user2?.team_id) {
      await adminFetch(`/admin/teams/${user2.team_id}/members/${user2.id}`, { method: 'DELETE' })
    }

    await page.goto('/team/enroll')
    await page.getByTestId(tid.teamEnroll.joinToken).fill(inviteToken!)
    await page.getByTestId(tid.teamEnroll.joinSubmit).click()
    await expect(page).toHaveURL(/\/team$/, { timeout: 10_000 })
  })
})

// ---------------------------------------------------------------------------
// Enroll: URL auto-submit
// ---------------------------------------------------------------------------
test.describe('Team: enroll URL auto-submit', () => {
  test('?token= in URL auto-submits join', async ({ user2Page: page }) => {
    const inviteToken = getTeamAlphaToken()
    if (!inviteToken) {
      test.skip(true, 'Token not available')
    }

    // Ensure user2 is not on a team
    const userList3 = await adminFetch('/admin/users?q=user2&field=username&per_page=5') as {
      data?: Array<{ id?: string; team_id?: string }>
    }
    const user2url = userList3.data?.[0]
    if (user2url?.team_id) {
      await adminFetch(`/admin/teams/${user2url.team_id}/members/${user2url.id}`, { method: 'DELETE' })
    }

    await page.goto(`/team/enroll?token=${inviteToken}`)
    // Auto-submits - should redirect to /team or show confirmation
    await expect(page).toHaveURL(/\/team(\?|$)/, { timeout: 10_000 })
  })
})

// ---------------------------------------------------------------------------
// My team
// ---------------------------------------------------------------------------
test.describe('Team: my-team page', () => {
  test('user1 sees Team Alpha on /team', async ({ user1Page: page }) => {
    await page.goto('/team')
    await expect(page.getByText('Team Alpha')).toBeVisible({ timeout: 10_000 })
  })

  test('invite token copy button is present', async ({ user1Page: page }) => {
    await page.goto('/team')
    await expect(page.getByTestId(tid.myTeam.inviteCopy)).toBeVisible({ timeout: 10_000 })
  })

  test('invite token regenerate button is present', async ({ user1Page: page }) => {
    await page.goto('/team')
    await expect(page.getByTestId(tid.myTeam.inviteRegen)).toBeVisible({ timeout: 10_000 })
  })

  test('user1 member row is visible', async ({ user1Page: page }) => {
    await page.goto('/team')
    await page.waitForTimeout(1000) // Let data load
    const memberRows = page.locator('[data-testid^="team-member-row-"]')
    await expect(memberRows.first()).toBeVisible({ timeout: 10_000 })
  })

  test('leave button is visible for captain', async ({ user1Page: page }) => {
    await page.goto('/team')
    await expect(page.getByTestId(tid.myTeam.leave)).toBeVisible({ timeout: 10_000 })
  })

  test('disband button is visible for captain', async ({ user1Page: page }) => {
    await page.goto('/team')
    await expect(page.getByTestId(tid.myTeam.disband)).toBeVisible({ timeout: 10_000 })
  })
})
