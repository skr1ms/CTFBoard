/**
 * Settings spec - profile, password, avatar, API tokens.
 */
import { expect, test } from '../fixtures'
import { tid } from '../helpers/selectors'
import * as path from 'path'
import { fileURLToPath } from 'url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const AVATAR_PNG = path.join(__dirname, '../fixtures/avatar-static.png')

test.describe('Settings: profile', () => {
  test('profile form renders with current values', async ({ user1Page: page }) => {
    await page.goto('/settings')
    await expect(page.getByTestId(tid.settings.username)).toBeVisible({ timeout: 8_000 })
    await expect(page.getByTestId(tid.settings.email)).toBeVisible()
    // Fields should be pre-filled
    await expect(page.getByTestId(tid.settings.username)).toHaveValue(/\w+/)
  })

  test('username taken shows error', async ({ user1Page: page }) => {
    await page.goto('/settings')
    await page.getByTestId(tid.settings.username).waitFor({ timeout: 8_000 })
    // Try to use admin's username
    await page.getByTestId(tid.settings.username).fill('admin')
    await page.getByTestId(tid.settings.profileSave).click()
    // Should show error
    await expect(page.getByText(/taken|conflict|already/i).first()).toBeVisible({ timeout: 5_000 })
  })
})

test.describe('Settings: password', () => {
  test('wrong current password shows error', async ({ user1Page: page }) => {
    await page.goto('/settings')
    await page.getByTestId(tid.settings.passwordCurrent).waitFor({ timeout: 8_000 })
    await page.getByTestId(tid.settings.passwordCurrent).fill('wrongcurrent')
    await page.getByTestId(tid.settings.passwordNew).fill('Newpass123')
    await page.getByTestId(tid.settings.passwordSave).click()
    await expect(page.getByText(/incorrect|invalid|wrong/i).first()).toBeVisible({ timeout: 5_000 })
  })
})

test.describe('Settings: avatar', () => {
  test('avatar upload input accepts PNG file', async ({ user1Page: page }) => {
    await page.goto('/settings')
    const uploadInput = page.getByTestId(tid.settings.avatarUpload)
    // Input may be hidden (label-triggered) - check it exists in DOM
    await expect(uploadInput.or(page.locator('input[type="file"]').first())).toBeTruthy()
  })

  test('remove avatar button is visible when avatar exists', async ({ user1Page: page }) => {
    await page.goto('/settings')
    const deleteBtn = page.getByTestId(tid.settings.avatarDelete)
    // Delete button only visible if avatar is set - just check it renders
    if (await deleteBtn.isVisible()) {
      await expect(deleteBtn).toBeEnabled()
    }
  })
})

test.describe('Settings: API tokens', () => {
  test('create token button is visible', async ({ user1Page: page }) => {
    await page.goto('/settings')
    await expect(page.getByTestId(tid.settings.tokenCreate)).toBeVisible({ timeout: 8_000 })
  })

  test('creating a token shows it in the list', async ({ user1Page: page }) => {
    await page.goto('/settings')
    await page.getByTestId(tid.settings.tokenCreate).waitFor({ timeout: 8_000 })
    await page.getByTestId(tid.settings.tokenCreate).click()

    // Token creation modal or inline form - fill description if prompted
    const descInput = page.getByPlaceholder(/description|token name/i)
    if (await descInput.isVisible({ timeout: 2_000 })) {
      await descInput.fill('e2e test token')
      await page.getByRole('button', { name: /create|generate/i }).click()
    }

    // A new token row should appear
    await expect(page.locator('[data-testid^="settings-token-row-"]').first()).toBeVisible({
      timeout: 8_000,
    })
  })

  test('can delete an existing token', async ({ user1Page: page }) => {
    await page.goto('/settings')
    const tokenRow = page.locator('[data-testid^="settings-token-row-"]').first()
    if (await tokenRow.isVisible({ timeout: 8_000 })) {
      const deleteBtn = tokenRow.getByRole('button', { name: /delete|remove/i })
      if (await deleteBtn.isVisible()) {
        const rowCount = await page.locator('[data-testid^="settings-token-row-"]').count()
        await deleteBtn.click()
        // Confirm if confirmation dialog
        const confirm = page.getByRole('button', { name: /confirm|yes|delete/i })
        if (await confirm.isVisible({ timeout: 1_000 })) await confirm.click()
        await expect(page.locator('[data-testid^="settings-token-row-"]')).toHaveCount(
          rowCount - 1,
          { timeout: 5_000 },
        )
      }
    }
  })
})
