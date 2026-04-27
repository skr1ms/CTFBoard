# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: banned.spec.ts >> Banned: appeal form >> appeal textarea is present
- Location: e2e/specs/banned.spec.ts:24:3

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: locator('textarea').first()
Expected: visible
Timeout: 8000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" with timeout 8000ms
  - waiting for locator('textarea').first()

```

# Page snapshot

```yaml
- generic [ref=e2]:
  - region "Notifications alt+T"
  - generic [ref=e3]:
    - generic [ref=e4]:
      - generic [ref=e5]: AstroCTF
      - button "Sign out" [ref=e6]
    - generic [ref=e8]:
      - generic [ref=e9]: 🚫
      - heading "Account Banned" [level=1] [ref=e11]
    - generic [ref=e12]:
      - heading "Your Appeals" [level=2] [ref=e13]
      - paragraph [ref=e14]: You have not submitted any appeals yet.
```

# Test source

```ts
  1  | /**
  2  |  * Banned user spec - landing page, appeal form, sign out.
  3  |  */
  4  | import { expect, test } from '../fixtures'
  5  | 
  6  | test.describe('Banned: landing page', () => {
  7  |   test('banned user lands on /banned', async ({ bannedPage: page }) => {
  8  |     await page.goto('/challenges')
  9  |     await expect(page).toHaveURL(/\/banned/, { timeout: 10_000 })
  10 |   })
  11 | 
  12 |   test('banned page shows reason and date', async ({ bannedPage: page }) => {
  13 |     await page.goto('/banned')
  14 |     await expect(page.getByText(/E2E banned fixture/i)).toBeVisible({ timeout: 8_000 })
  15 |   })
  16 | 
  17 |   test('banned user cannot access /challenges', async ({ bannedPage: page }) => {
  18 |     await page.goto('/challenges')
  19 |     await expect(page).toHaveURL(/\/banned/, { timeout: 10_000 })
  20 |   })
  21 | })
  22 | 
  23 | test.describe('Banned: appeal form', () => {
  24 |   test('appeal textarea is present', async ({ bannedPage: page }) => {
  25 |     await page.goto('/banned')
> 26 |     await expect(page.locator('textarea').first()).toBeVisible({ timeout: 8_000 })
     |                                                    ^ Error: expect(locator).toBeVisible() failed
  27 |   })
  28 | 
  29 |   test('too-short appeal shows validation error', async ({ bannedPage: page }) => {
  30 |     await page.goto('/banned')
  31 |     const textarea = page.locator('textarea').first()
  32 |     await textarea.fill('short')
  33 |     await page.getByRole('button', { name: /submit|appeal|send/i }).click()
  34 |     // Should show validation error (min 20 chars per plan)
  35 |     await expect(page.getByText(/too short|minimum|at least/i).first()).toBeVisible({
  36 |       timeout: 3_000,
  37 |     })
  38 |   })
  39 | 
  40 |   test('valid appeal submission shows pending state', async ({ bannedPage: page }) => {
  41 |     await page.goto('/banned')
  42 |     const textarea = page.locator('textarea').first()
  43 |     await textarea.fill('I am appealing this ban because I did not violate any rules.')
  44 |     await page.getByRole('button', { name: /submit|appeal|send/i }).click()
  45 |     // Should show pending/submitted state
  46 |     await expect(
  47 |       page.getByText(/submitted|pending|review|received/i).first(),
  48 |     ).toBeVisible({ timeout: 5_000 })
  49 |   })
  50 | })
  51 | 
  52 | test.describe('Banned: sign out', () => {
  53 |   test('sign out clears session', async ({ bannedPage: page }) => {
  54 |     await page.goto('/banned')
  55 |     const signOut = page.getByRole('button', { name: /sign out|logout/i })
  56 |     if (await signOut.isVisible()) {
  57 |       await signOut.click()
  58 |       await expect(page).toHaveURL(/\/login/, { timeout: 10_000 })
  59 |     }
  60 |   })
  61 | })
  62 | 
```