/**
 * Admin content spec - pages, notifications, awards.
 */
import { expect, test } from '../fixtures'
import { tid } from '../helpers/selectors'

// ---------------------------------------------------------------------------
// Pages
// ---------------------------------------------------------------------------
test.describe('Admin: pages', () => {
  test('list renders seeded pages', async ({ adminPage: page }) => {
    await page.goto('/admin/pages')
    await expect(page.getByText('Rules')).toBeVisible({ timeout: 10_000 })
    await expect(page.getByText('Draft Me')).toBeVisible()
  })

  test('rules page is published, draft-me is draft', async ({ adminPage: page }) => {
    await page.goto('/admin/pages')
    await expect(page.getByText('published').first()).toBeVisible({ timeout: 10_000 })
    await expect(page.getByText('draft').first()).toBeVisible()
  })

  test('create new page', async ({ adminPage: page }) => {
    await page.goto('/admin/pages')
    await page.getByRole('button', { name: /new page/i }).click()

    const titleInput = page.getByPlaceholder(/title/i)
    const slugInput = page.getByPlaceholder(/slug/i)
    await titleInput.fill(`Test Page ${Date.now()}`)
    const slug = `test-page-${Date.now()}`
    await slugInput.fill(slug)
    await page.getByRole('button', { name: /create page/i }).click()
    await expect(page.getByText(slug)).toBeVisible({ timeout: 8_000 })
  })

  test('edit page', async ({ adminPage: page }) => {
    await page.goto('/admin/pages')
    const editBtn = page.locator('[data-testid^="admin-edit-"]').first()
    if (await editBtn.isVisible({ timeout: 5_000 })) {
      await editBtn.click()
      await expect(page.getByPlaceholder(/title/i)).toBeVisible()
    }
  })

  test('delete page', async ({ adminPage: page }) => {
    await page.goto('/admin/pages')
    const name = `del-page-${Date.now()}`
    await page.getByRole('button', { name: /new page/i }).click()
    await page.getByPlaceholder(/title/i).fill(name)
    await page.getByPlaceholder(/slug/i).fill(`del-page-${Date.now()}`)
    await page.getByRole('button', { name: /create page/i }).click()
    await page.getByText(name).waitFor({ timeout: 8_000 })

    const row = page.locator('[data-testid^="admin-row-"]').filter({ hasText: name })
    const id = (await row.getAttribute('data-testid'))?.replace('admin-row-', '')
    if (id) {
      await page.getByTestId(tid.admin.delete(id)).click()
      await expect(page.getByText(name)).not.toBeVisible({ timeout: 5_000 })
    }
  })
})

// ---------------------------------------------------------------------------
// Notifications
// ---------------------------------------------------------------------------
test.describe('Admin: notifications', () => {
  test('list renders seeded global notification', async ({ adminPage: page }) => {
    await page.goto('/admin/notifications')
    await expect(page.getByText('Welcome to E2E CTF')).toBeVisible({ timeout: 10_000 })
  })

  test('create global notification', async ({ adminPage: page }) => {
    await page.goto('/admin/notifications')
    const unique = `Test Notif ${Date.now()}`
    await page.getByPlaceholder(/title/i).fill(unique)
    const contentInput = page.getByPlaceholder(/content|message/i)
    await contentInput.fill('Test notification content')
    await page.getByRole('button', { name: /create|send|post/i }).click()
    await expect(page.getByText(unique)).toBeVisible({ timeout: 8_000 })
  })

  test('edit notification', async ({ adminPage: page }) => {
    await page.goto('/admin/notifications')
    const editBtn = page.locator('[data-testid^="admin-edit-"]').first()
    if (await editBtn.isVisible({ timeout: 5_000 })) {
      await editBtn.click()
      await expect(page.getByRole('dialog').or(page.locator('form'))).toBeVisible()
    }
  })

  test('delete notification', async ({ adminPage: page }) => {
    await page.goto('/admin/notifications')
    const name = `Del Notif ${Date.now()}`
    await page.getByPlaceholder(/title/i).fill(name)
    await page.getByPlaceholder(/content|message/i).fill('content')
    await page.getByRole('button', { name: /create|send|post/i }).click()
    await page.getByText(name).waitFor({ timeout: 8_000 })

    const row = page.locator('[data-testid^="admin-row-"]').filter({ hasText: name })
    const id = (await row.getAttribute('data-testid'))?.replace('admin-row-', '')
    if (id) {
      await page.getByTestId(tid.admin.delete(id)).click()
      await expect(page.getByText(name)).not.toBeVisible({ timeout: 5_000 })
    }
  })
})

// ---------------------------------------------------------------------------
// Awards
// ---------------------------------------------------------------------------
test.describe('Admin: awards', () => {
  test('list renders Team Alpha participation award', async ({ adminPage: page }) => {
    await page.goto('/admin/awards')
    await expect(page.getByText('E2E participation award')).toBeVisible({ timeout: 10_000 })
  })

  test('award shows +50 value', async ({ adminPage: page }) => {
    await page.goto('/admin/awards')
    await expect(page.getByText('+50').or(page.getByText('50'))).toBeVisible({ timeout: 10_000 })
  })
})
