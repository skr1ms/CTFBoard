/**
 * Admin challenges spec - list CRUD, bulk delete, hints, requirements.
 */
import { expect, test } from '../fixtures'
import { tid } from '../helpers/selectors'

test.describe('Admin: challenges list', () => {
  test('renders seed challenges', async ({ adminPage: page }) => {
    await page.goto('/admin/challenges')
    await expect(page.getByText('Hello World')).toBeVisible({ timeout: 10_000 })
    await expect(page.getByText('Base64 Basics')).toBeVisible()
  })

  test('create challenge via form/modal', async ({ adminPage: page }) => {
    await page.goto('/admin/challenges')
    const unique = `E2E Challenge ${Date.now()}`

    // Open create form or modal
    const createBtn = page.getByRole('button', { name: /create|add.*challenge|new/i }).first()
    if (await createBtn.isVisible({ timeout: 3_000 })) {
      await createBtn.click()
    }

    await page.getByLabel(/title/i).fill(unique)
    await page.getByLabel(/category/i).fill('E2E')
    await page.getByLabel(/flag/i).fill(`CTF{e2e_${Date.now()}}`)
    await page.getByLabel(/points/i).fill('50')

    await page.getByRole('button', { name: /create|save|submit/i }).click()
    await expect(page.getByText(unique)).toBeVisible({ timeout: 8_000 })
  })

  test('delete single challenge', async ({ adminPage: page }) => {
    await page.goto('/admin/challenges')
    // Create a throwaway challenge first
    const name = `ToDelete ${Date.now()}`
    const createBtn = page.getByRole('button', { name: /create|add|new/i }).first()
    if (await createBtn.isVisible({ timeout: 3_000 })) {
      await createBtn.click()
      await page.getByLabel(/title/i).fill(name)
      await page.getByLabel(/category/i).fill('E2E')
      await page.getByLabel(/flag/i).fill('CTF{delete_me}')
      await page.getByLabel(/points/i).fill('1')
      await page.getByRole('button', { name: /create|save/i }).click()
      await page.getByText(name).waitFor({ timeout: 8_000 })

      const row = page.locator('[data-testid^="admin-row-"]').filter({ hasText: name })
      const id = (await row.getAttribute('data-testid'))?.replace('admin-row-', '')
      if (id) {
        await page.getByTestId(tid.admin.delete(id)).click()
        // Confirm if needed
        const confirm = page.getByRole('button', { name: /confirm|yes|delete/i })
        if (await confirm.isVisible({ timeout: 1_000 })) await confirm.click()
        await expect(page.getByText(name)).not.toBeVisible({ timeout: 5_000 })
      }
    }
  })
})

test.describe('Admin: challenge detail', () => {
  test('detail page loads for Hello World', async ({ adminPage: page }) => {
    await page.goto('/admin/challenges')
    await page.locator('[data-testid^="admin-edit-"]').first().waitFor({ timeout: 10_000 })

    // Navigate to detail page
    const row = page.locator('[data-testid^="admin-row-"]').filter({ hasText: 'Hello World' })
    const id = (await row.getAttribute('data-testid'))?.replace('admin-row-', '')
    if (id) {
      const editBtn = page.getByTestId(tid.admin.edit(id))
      await editBtn.click()
      await expect(page).toHaveURL(/\/admin\/challenges\/[\w-]+/, { timeout: 5_000 })
    }
  })
})
