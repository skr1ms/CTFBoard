/**
 * Admin taxonomy spec - brackets, tags, fields CRUD.
 */
import { expect, test } from '../fixtures'
import { tid } from '../helpers/selectors'

// ---------------------------------------------------------------------------
// Brackets
// ---------------------------------------------------------------------------
test.describe('Admin: brackets', () => {
  test('list renders Rookie and Pro brackets', async ({ adminPage: page }) => {
    await page.goto('/admin/brackets')
    await expect(page.getByText('Rookie')).toBeVisible({ timeout: 10_000 })
    await expect(page.getByText('Pro')).toBeVisible()
  })

  test('Rookie is marked as default', async ({ adminPage: page }) => {
    await page.goto('/admin/brackets')
    await expect(page.getByText('default')).toBeVisible({ timeout: 10_000 })
  })

  test('create new bracket', async ({ adminPage: page }) => {
    await page.goto('/admin/brackets')
    const unique = `TestBracket-${Date.now()}`
    await page.getByPlaceholder(/e\.g\. Open|bracket name/i).fill(unique)
    await page.getByRole('button', { name: /create bracket/i }).click()
    await expect(page.getByText(unique)).toBeVisible({ timeout: 8_000 })
  })

  test('edit bracket name', async ({ adminPage: page }) => {
    await page.goto('/admin/brackets')
    const editBtn = page.locator('[data-testid^="admin-edit-"]').last()
    if (await editBtn.isVisible({ timeout: 5_000 })) {
      await editBtn.click()
      const nameInput = page.locator('input[value]').first()
      await nameInput.fill(`Edited-${Date.now()}`)
      await page.getByRole('button', { name: /save/i }).click()
      await expect(page.getByText(/edited-/i)).toBeVisible({ timeout: 5_000 })
    }
  })

  test('delete bracket', async ({ adminPage: page }) => {
    await page.goto('/admin/brackets')
    // Create a bracket to delete
    const name = `ToDelete-${Date.now()}`
    await page.getByPlaceholder(/e\.g\. Open|bracket name/i).fill(name)
    await page.getByRole('button', { name: /create bracket/i }).click()
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
// Tags
// ---------------------------------------------------------------------------
test.describe('Admin: tags', () => {
  test('list renders web-easy tag', async ({ adminPage: page }) => {
    await page.goto('/admin/tags')
    await expect(page.getByText('web-easy')).toBeVisible({ timeout: 10_000 })
  })

  test('create new tag', async ({ adminPage: page }) => {
    await page.goto('/admin/tags')
    const unique = `e2e-tag-${Date.now()}`
    await page.getByPlaceholder(/e\.g\. web|tag name/i).fill(unique)
    await page.getByRole('button', { name: /add tag/i }).click()
    await expect(page.getByText(unique)).toBeVisible({ timeout: 8_000 })
  })

  test('delete tag', async ({ adminPage: page }) => {
    await page.goto('/admin/tags')
    const name = `del-tag-${Date.now()}`
    await page.getByPlaceholder(/e\.g\. web|tag name/i).fill(name)
    await page.getByRole('button', { name: /add tag/i }).click()
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
// Fields
// ---------------------------------------------------------------------------
test.describe('Admin: fields', () => {
  test('list renders country field', async ({ adminPage: page }) => {
    await page.goto('/admin/fields')
    await expect(page.getByText('country')).toBeVisible({ timeout: 10_000 })
  })

  test('filter by user entity shows country', async ({ adminPage: page }) => {
    await page.goto('/admin/fields')
    const userFilter = page.getByRole('button', { name: /^user$/i })
    if (await userFilter.isVisible()) {
      await userFilter.click()
      await expect(page.getByText('country')).toBeVisible()
    }
  })

  test('create new text field', async ({ adminPage: page }) => {
    await page.goto('/admin/fields')
    const unique = `e2e-field-${Date.now()}`
    await page.getByPlaceholder(/e\.g\. Discord|field name/i).fill(unique)
    await page.getByRole('button', { name: /create field/i }).click()
    await expect(page.getByText(unique)).toBeVisible({ timeout: 8_000 })
  })
})
