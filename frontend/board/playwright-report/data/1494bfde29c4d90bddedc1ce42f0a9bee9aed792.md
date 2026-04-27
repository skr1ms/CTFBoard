# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: admin-content.spec.ts >> Admin: pages >> list renders seeded pages
- Location: e2e/specs/admin-content.spec.ts:11:3

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: getByText('Rules')
Expected: visible
Error: strict mode violation: getByText('Rules') resolved to 2 elements:
    1) <td class="px-3 py-2.5 text-sm text-text-primary font-medium">Rules</td> aka getByRole('cell', { name: 'Rules', exact: true })
    2) <td class="px-3 py-2.5 text-xs font-mono text-nebula-purple">/rules</td> aka getByRole('cell', { name: '/rules' })

Call log:
  - Expect "toBeVisible" with timeout 10000ms
  - waiting for getByText('Rules')

```

# Page snapshot

```yaml
- generic [ref=e2]:
  - region "Notifications alt+T"
  - generic [ref=e3]:
    - banner [ref=e4]:
      - navigation [ref=e5]:
        - link "✦ AstroCTFb" [ref=e6] [cursor=pointer]:
          - /url: /
          - generic [ref=e7]: ✦
          - generic [ref=e8]: AstroCTFb
        - list [ref=e9]:
          - listitem [ref=e10]:
            - link "Challenges" [ref=e11] [cursor=pointer]:
              - /url: /challenges
          - listitem [ref=e12]:
            - link "Scoreboard" [ref=e13] [cursor=pointer]:
              - /url: /scoreboard
          - listitem [ref=e14]:
            - link "Teams" [ref=e15] [cursor=pointer]:
              - /url: /teams
          - listitem [ref=e16]:
            - link "Users" [ref=e17] [cursor=pointer]:
              - /url: /users
          - listitem [ref=e18]:
            - link "Activity" [ref=e19] [cursor=pointer]:
              - /url: /activity
          - listitem [ref=e20]:
            - link "Admin" [ref=e21] [cursor=pointer]:
              - /url: /admin
        - generic [ref=e22]:
          - button "Notifications" [ref=e23]:
            - img [ref=e24]
          - button "User menu" [ref=e27]:
            - generic "admin" [ref=e28]:
              - generic [ref=e29]: A
    - main [ref=e31]:
      - generic [ref=e32]:
        - complementary [ref=e34]:
          - generic [ref=e35]:
            - generic [ref=e36]: Admin Panel
            - button "Collapse sidebar" [ref=e37]:
              - img [ref=e38]
          - navigation [ref=e40]:
            - list [ref=e41]:
              - listitem [ref=e42]:
                - link "📊 Statistics" [ref=e43] [cursor=pointer]:
                  - /url: /admin/statistics
                  - generic [ref=e44]: 📊
                  - generic [ref=e45]: Statistics
              - listitem [ref=e46]:
                - link "🏆 Competition" [ref=e47] [cursor=pointer]:
                  - /url: /admin/competition
                  - generic [ref=e48]: 🏆
                  - generic [ref=e49]: Competition
              - listitem [ref=e50]:
                - link "🎯 Challenges" [ref=e51] [cursor=pointer]:
                  - /url: /admin/challenges
                  - generic [ref=e52]: 🎯
                  - generic [ref=e53]: Challenges
              - listitem [ref=e54]:
                - link "👥 Users" [ref=e55] [cursor=pointer]:
                  - /url: /admin/users
                  - generic [ref=e56]: 👥
                  - generic [ref=e57]: Users
              - listitem [ref=e58]:
                - link "🛡️ Teams" [ref=e59] [cursor=pointer]:
                  - /url: /admin/teams
                  - generic [ref=e60]: 🛡️
                  - generic [ref=e61]: Teams
              - listitem [ref=e62]:
                - link "📝 Submissions" [ref=e63] [cursor=pointer]:
                  - /url: /admin/submissions
                  - generic [ref=e64]: 📝
                  - generic [ref=e65]: Submissions
              - listitem [ref=e66]:
                - link "🔓 Unlocks" [ref=e67] [cursor=pointer]:
                  - /url: /admin/unlocks
                  - generic [ref=e68]: 🔓
                  - generic [ref=e69]: Unlocks
              - listitem [ref=e70]:
                - link "⚖️ Appeals" [ref=e71] [cursor=pointer]:
                  - /url: /admin/appeals
                  - generic [ref=e72]: ⚖️
                  - generic [ref=e73]: Appeals
              - listitem [ref=e74]:
                - link "🎖️ Awards" [ref=e75] [cursor=pointer]:
                  - /url: /admin/awards
                  - generic [ref=e76]: 🎖️
                  - generic [ref=e77]: Awards
              - listitem [ref=e78]:
                - link "📋 Scoreboard" [ref=e79] [cursor=pointer]:
                  - /url: /admin/scoreboard
                  - generic [ref=e80]: 📋
                  - generic [ref=e81]: Scoreboard
              - listitem [ref=e82]:
                - link "🔔 Notifications" [ref=e83] [cursor=pointer]:
                  - /url: /admin/notifications
                  - generic [ref=e84]: 🔔
                  - generic [ref=e85]: Notifications
              - listitem [ref=e86]:
                - link "📄 Pages" [ref=e87] [cursor=pointer]:
                  - /url: /admin/pages
                  - generic [ref=e88]: 📄
                  - generic [ref=e89]: Pages
              - listitem [ref=e90]:
                - link "🏷️ Tags" [ref=e91] [cursor=pointer]:
                  - /url: /admin/tags
                  - generic [ref=e92]: 🏷️
                  - generic [ref=e93]: Tags
              - listitem [ref=e94]:
                - link "🗂️ Brackets" [ref=e95] [cursor=pointer]:
                  - /url: /admin/brackets
                  - generic [ref=e96]: 🗂️
                  - generic [ref=e97]: Brackets
              - listitem [ref=e98]:
                - link "🗃️ Fields" [ref=e99] [cursor=pointer]:
                  - /url: /admin/fields
                  - generic [ref=e100]: 🗃️
                  - generic [ref=e101]: Fields
              - listitem [ref=e102]:
                - link "⚙️ Settings" [ref=e103] [cursor=pointer]:
                  - /url: /admin/settings
                  - generic [ref=e104]: ⚙️
                  - generic [ref=e105]: Settings
              - listitem [ref=e106]:
                - link "🗄️ Storage" [ref=e107] [cursor=pointer]:
                  - /url: /admin/storage
                  - generic [ref=e108]: 🗄️
                  - generic [ref=e109]: Storage
              - listitem [ref=e110]:
                - link "💾 Backup" [ref=e111] [cursor=pointer]:
                  - /url: /admin/backup
                  - generic [ref=e112]: 💾
                  - generic [ref=e113]: Backup
          - link "Back to site" [ref=e115] [cursor=pointer]:
            - /url: /challenges
            - img [ref=e116]
            - generic [ref=e118]: Back to site
        - main [ref=e120]:
          - generic [ref=e121]:
            - generic [ref=e122]:
              - heading "Pages" [level=1] [ref=e123]
              - button "+ New page" [ref=e124]
            - table [ref=e126]:
              - rowgroup [ref=e127]:
                - row "Title Slug Status Order Updated Actions" [ref=e128]:
                  - columnheader "Title" [ref=e129]
                  - columnheader "Slug" [ref=e130]
                  - columnheader "Status" [ref=e131]
                  - columnheader "Order" [ref=e132]
                  - columnheader "Updated" [ref=e133]
                  - columnheader "Actions" [ref=e134]
              - rowgroup [ref=e135]:
                - row "Rules /rules published 0 4/19/2026 Edit Delete" [ref=e136]:
                  - cell "Rules" [ref=e137]
                  - cell "/rules" [ref=e138]
                  - cell "published" [ref=e139]:
                    - generic [ref=e140]: published
                  - cell "0" [ref=e141]
                  - cell "4/19/2026" [ref=e142]
                  - cell "Edit Delete" [ref=e143]:
                    - button "Edit" [ref=e144]
                    - button "Delete" [ref=e145]
                - row "Draft Me /draft-me draft 99 4/19/2026 Edit Delete" [ref=e146]:
                  - cell "Draft Me" [ref=e147]
                  - cell "/draft-me" [ref=e148]
                  - cell "draft" [ref=e149]:
                    - generic [ref=e150]: draft
                  - cell "99" [ref=e151]
                  - cell "4/19/2026" [ref=e152]
                  - cell "Edit Delete" [ref=e153]:
                    - button "Edit" [ref=e154]
                    - button "Delete" [ref=e155]
            - paragraph [ref=e156]:
              - text: 2 pages total
              - generic [ref=e157]: (1 published)
```

# Test source

```ts
  1   | /**
  2   |  * Admin content spec - pages, notifications, awards.
  3   |  */
  4   | import { expect, test } from '../fixtures'
  5   | import { tid } from '../helpers/selectors'
  6   | 
  7   | // ---------------------------------------------------------------------------
  8   | // Pages
  9   | // ---------------------------------------------------------------------------
  10  | test.describe('Admin: pages', () => {
  11  |   test('list renders seeded pages', async ({ adminPage: page }) => {
  12  |     await page.goto('/admin/pages')
> 13  |     await expect(page.getByText('Rules')).toBeVisible({ timeout: 10_000 })
      |                                           ^ Error: expect(locator).toBeVisible() failed
  14  |     await expect(page.getByText('Draft Me')).toBeVisible()
  15  |   })
  16  | 
  17  |   test('rules page is published, draft-me is draft', async ({ adminPage: page }) => {
  18  |     await page.goto('/admin/pages')
  19  |     await expect(page.getByText('published').first()).toBeVisible({ timeout: 10_000 })
  20  |     await expect(page.getByText('draft').first()).toBeVisible()
  21  |   })
  22  | 
  23  |   test('create new page', async ({ adminPage: page }) => {
  24  |     await page.goto('/admin/pages')
  25  |     await page.getByRole('button', { name: /new page/i }).click()
  26  | 
  27  |     const titleInput = page.getByPlaceholder(/title/i)
  28  |     const slugInput = page.getByPlaceholder(/slug/i)
  29  |     await titleInput.fill(`Test Page ${Date.now()}`)
  30  |     const slug = `test-page-${Date.now()}`
  31  |     await slugInput.fill(slug)
  32  |     await page.getByRole('button', { name: /create page/i }).click()
  33  |     await expect(page.getByText(slug)).toBeVisible({ timeout: 8_000 })
  34  |   })
  35  | 
  36  |   test('edit page', async ({ adminPage: page }) => {
  37  |     await page.goto('/admin/pages')
  38  |     const editBtn = page.locator('[data-testid^="admin-edit-"]').first()
  39  |     if (await editBtn.isVisible({ timeout: 5_000 })) {
  40  |       await editBtn.click()
  41  |       await expect(page.getByPlaceholder(/title/i)).toBeVisible()
  42  |     }
  43  |   })
  44  | 
  45  |   test('delete page', async ({ adminPage: page }) => {
  46  |     await page.goto('/admin/pages')
  47  |     const name = `del-page-${Date.now()}`
  48  |     await page.getByRole('button', { name: /new page/i }).click()
  49  |     await page.getByPlaceholder(/title/i).fill(name)
  50  |     await page.getByPlaceholder(/slug/i).fill(`del-page-${Date.now()}`)
  51  |     await page.getByRole('button', { name: /create page/i }).click()
  52  |     await page.getByText(name).waitFor({ timeout: 8_000 })
  53  | 
  54  |     const row = page.locator('[data-testid^="admin-row-"]').filter({ hasText: name })
  55  |     const id = (await row.getAttribute('data-testid'))?.replace('admin-row-', '')
  56  |     if (id) {
  57  |       await page.getByTestId(tid.admin.delete(id)).click()
  58  |       await expect(page.getByText(name)).not.toBeVisible({ timeout: 5_000 })
  59  |     }
  60  |   })
  61  | })
  62  | 
  63  | // ---------------------------------------------------------------------------
  64  | // Notifications
  65  | // ---------------------------------------------------------------------------
  66  | test.describe('Admin: notifications', () => {
  67  |   test('list renders seeded global notification', async ({ adminPage: page }) => {
  68  |     await page.goto('/admin/notifications')
  69  |     await expect(page.getByText('Welcome to E2E CTF')).toBeVisible({ timeout: 10_000 })
  70  |   })
  71  | 
  72  |   test('create global notification', async ({ adminPage: page }) => {
  73  |     await page.goto('/admin/notifications')
  74  |     const unique = `Test Notif ${Date.now()}`
  75  |     await page.getByPlaceholder(/title/i).fill(unique)
  76  |     const contentInput = page.getByPlaceholder(/content|message/i)
  77  |     await contentInput.fill('Test notification content')
  78  |     await page.getByRole('button', { name: /create|send|post/i }).click()
  79  |     await expect(page.getByText(unique)).toBeVisible({ timeout: 8_000 })
  80  |   })
  81  | 
  82  |   test('edit notification', async ({ adminPage: page }) => {
  83  |     await page.goto('/admin/notifications')
  84  |     const editBtn = page.locator('[data-testid^="admin-edit-"]').first()
  85  |     if (await editBtn.isVisible({ timeout: 5_000 })) {
  86  |       await editBtn.click()
  87  |       await expect(page.getByRole('dialog').or(page.locator('form'))).toBeVisible()
  88  |     }
  89  |   })
  90  | 
  91  |   test('delete notification', async ({ adminPage: page }) => {
  92  |     await page.goto('/admin/notifications')
  93  |     const name = `Del Notif ${Date.now()}`
  94  |     await page.getByPlaceholder(/title/i).fill(name)
  95  |     await page.getByPlaceholder(/content|message/i).fill('content')
  96  |     await page.getByRole('button', { name: /create|send|post/i }).click()
  97  |     await page.getByText(name).waitFor({ timeout: 8_000 })
  98  | 
  99  |     const row = page.locator('[data-testid^="admin-row-"]').filter({ hasText: name })
  100 |     const id = (await row.getAttribute('data-testid'))?.replace('admin-row-', '')
  101 |     if (id) {
  102 |       await page.getByTestId(tid.admin.delete(id)).click()
  103 |       await expect(page.getByText(name)).not.toBeVisible({ timeout: 5_000 })
  104 |     }
  105 |   })
  106 | })
  107 | 
  108 | // ---------------------------------------------------------------------------
  109 | // Awards
  110 | // ---------------------------------------------------------------------------
  111 | test.describe('Admin: awards', () => {
  112 |   test('list renders Team Alpha participation award', async ({ adminPage: page }) => {
  113 |     await page.goto('/admin/awards')
```